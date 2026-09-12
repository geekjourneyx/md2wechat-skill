// Package syncprepare prepares local semantic HTML for a host agent.
package syncprepare

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"html"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	_ "golang.org/x/image/webp"
	"gopkg.in/yaml.v3"
)

type Result struct {
	ExecutionOwner string   `json:"execution_owner"`
	Title          string   `json:"title"`
	BodyHTML       string   `json:"body_html"`
	Images         []string `json:"images"`
	HeadingLevels  []int    `json:"heading_levels"`
}

var obsidianCallout = regexp.MustCompile(`(?m)^\s*\[![A-Za-z][A-Za-z0-9_-]*\]`)

// Prepare validates everything before creating a new output directory.
func Prepare(article, output string) (*Result, error) {
	if strings.TrimSpace(output) == "" {
		return nil, fmt.Errorf("--output must name a new directory")
	}
	source, err := os.ReadFile(article)
	if err != nil {
		return nil, err
	}
	title, source, err := frontmatter(source)
	if err != nil {
		return nil, err
	}
	md := goldmark.New(goldmark.WithExtensions(extension.Table))
	doc := md.Parser().Parse(text.NewReader(source))
	result := &Result{ExecutionOwner: "host_agent", Images: []string{}, HeadingLevels: []int{}}
	levels := map[int]bool{}
	seen := map[string]bool{}
	meaningful := false
	var firstHeading *ast.Heading
	err = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.FencedCodeBlock, *ast.CodeBlock:
			if n.Lines().Len() > 0 {
				meaningful = true
			}
			return ast.WalkSkipChildren, nil
		case *ast.CodeSpan:
			if !insideHeading(n) {
				meaningful = true
			}
			return ast.WalkSkipChildren, nil
		case *ast.HTMLBlock, *ast.RawHTML:
			return ast.WalkStop, fmt.Errorf("raw HTML is unsupported; convert it to Markdown first")
		case *ast.Heading:
			if firstHeading == nil {
				var headingText strings.Builder
				collectTitle(v, source, &headingText)
				if strings.TrimSpace(headingText.String()) != "" {
					firstHeading = v
				}
			}
			levels[v.Level] = true
		case *ast.Image:
			path, e := localImage(string(v.Destination), filepath.Dir(article))
			if e != nil {
				return ast.WalkStop, e
			}
			v.Destination = []byte(strings.ReplaceAll((&url.URL{Path: path}).EscapedPath(), "&", "%26"))
			if !insideHeading(n) {
				meaningful = true
			}
			if !seen[path] {
				result.Images = append(result.Images, path)
				seen[path] = true
			}
		case *ast.Link:
			dest := string(markdownUnescape(v.Destination))
			u, e := url.Parse(dest)
			if e != nil || (u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https") {
				return ast.WalkStop, fmt.Errorf("unsupported hyperlink destination %q", dest)
			}
		case *ast.AutoLink:
			u, e := url.Parse(string(v.URL(source)))
			if e != nil || (u.Scheme != "http" && u.Scheme != "https") {
				return ast.WalkStop, fmt.Errorf("only HTTP(S) automatic links are supported")
			}
			if !insideHeading(n) {
				meaningful = true
			}
		case *ast.Text:
			if strings.TrimSpace(string(v.Segment.Value(source))) != "" && !insideHeading(n) {
				meaningful = true
			}
		}
		// Inspect each inline container as a whole so split Markdown text nodes
		// cannot hide unresolved syntax. Code spans are excluded below.
		if n.Type() == ast.TypeBlock && n.HasChildren() {
			var plain strings.Builder
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if c.Type() == ast.TypeInline {
					collectText(c, source, &plain)
				}
			}
			s := plain.String()
			if strings.Contains(s, "[[") || strings.Contains(s, "__generate:") || obsidianCallout.MatchString(s) {
				return ast.WalkStop, fmt.Errorf("unresolved Obsidian or image-generation syntax; materialize it as ordinary Markdown first")
			}
			for _, line := range strings.Split(s, "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), ":::") {
					return ast.WalkStop, fmt.Errorf("advanced layout blocks are unsupported; convert them to ordinary Markdown first")
				}
			}
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, err
	}
	if title == "" && firstHeading != nil {
		var b strings.Builder
		collectTitle(firstHeading, source, &b)
		title = strings.TrimSpace(b.String())
	}
	if title == "" {
		return nil, fmt.Errorf("article requires a nonempty frontmatter title or Markdown heading")
	}
	if !meaningful {
		return nil, fmt.Errorf("article requires body content beyond its title or headings")
	}
	result.Title = title
	for level := range levels {
		result.HeadingLevels = append(result.HeadingLevels, level)
	}
	sort.Ints(result.HeadingLevels)
	var rendered bytes.Buffer
	if err = md.Renderer().Render(&rendered, source, doc); err != nil {
		return nil, err
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return nil, err
	}
	if err = os.Mkdir(output, 0700); err != nil {
		return nil, fmt.Errorf("output must be a new directory: %w", err)
	}
	result.BodyHTML = filepath.Join(output, "body.html")
	file, err := os.CreateTemp(output, ".body-*")
	if err != nil {
		_ = os.Remove(output)
		return nil, err
	}
	tmp := file.Name()
	_, err = file.Write(rendered.Bytes())
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmp, result.BodyHTML)
	}
	if err != nil {
		_ = os.Remove(tmp)
		_ = os.Remove(output)
		return nil, err
	}
	return result, nil
}

func collectTitle(n ast.Node, source []byte, b *strings.Builder) {
	if span, ok := n.(*ast.CodeSpan); ok {
		for c := span.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				b.Write(t.Segment.Value(source))
			}
		}
		return
	}
	if link, ok := n.(*ast.AutoLink); ok {
		b.Write(link.Label(source))
		return
	}
	if t, ok := n.(*ast.Text); ok {
		var escaped bytes.Buffer
		writer := bufio.NewWriter(&escaped)
		goldmarkhtml.DefaultWriter.Write(writer, t.Segment.Value(source))
		_ = writer.Flush()
		b.WriteString(html.UnescapeString(escaped.String()))
		return
	}
	if s, ok := n.(*ast.String); ok {
		b.Write(s.Value)
		return
	}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		collectTitle(c, source, b)
	}
}

func markdownUnescape(value []byte) []byte {
	return util.ResolveEntityNames(util.ResolveNumericReferences(util.UnescapePunctuations(value)))
}

func insideHeading(n ast.Node) bool {
	for p := n.Parent(); p != nil; p = p.Parent() {
		if _, ok := p.(*ast.Heading); ok {
			return true
		}
	}
	return false
}
func collectText(n ast.Node, source []byte, b *strings.Builder) {
	if _, ok := n.(*ast.CodeSpan); ok {
		return
	}
	if t, ok := n.(*ast.Text); ok {
		b.Write(t.Segment.Value(source))
		if t.SoftLineBreak() || t.HardLineBreak() {
			b.WriteByte('\n')
		}
		return
	}
	if s, ok := n.(*ast.String); ok {
		b.Write(s.Value)
		return
	}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		collectText(c, source, b)
	}
}

func frontmatter(source []byte) (string, []byte, error) {
	source = bytes.TrimPrefix(source, []byte{0xef, 0xbb, 0xbf})
	lines := bytes.Split(source, []byte("\n"))
	if strings.TrimSpace(string(lines[0])) != "---" {
		return "", source, nil
	}
	end := 0
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(string(lines[i])) == "---" {
			end = i
			break
		}
	}
	if end == 0 {
		return "", nil, fmt.Errorf("frontmatter has no closing ---")
	}
	var data map[string]any
	if err := yaml.Unmarshal(bytes.Join(lines[1:end], []byte("\n")), &data); err != nil {
		return "", nil, fmt.Errorf("invalid frontmatter: %w", err)
	}
	title := ""
	if value, ok := data["title"]; ok {
		var valid bool
		title, valid = value.(string)
		if !valid || strings.TrimSpace(title) == "" || strings.ContainsAny(title, "\r\n") {
			return "", nil, fmt.Errorf("frontmatter title must be a nonempty single-line string")
		}
		title = strings.TrimSpace(title)
	}
	return title, bytes.Join(lines[end+1:], []byte("\n")), nil
}

func localImage(raw, base string) (string, error) {
	raw = string(markdownUnescape([]byte(raw)))
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "" || u.Host != "" || strings.HasPrefix(raw, "//") || u.RawQuery != "" || u.Fragment != "" || strings.Contains(raw, "__generate:") {
		return "", fmt.Errorf("image %q must be materialized as a local raster file first", raw)
	}
	path := u.Path // url.Parse already decodes percent escapes exactly once.
	if path == "" {
		return "", fmt.Errorf("invalid local image path %q", raw)
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(base, path)
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("local image: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("image must be a regular raster file: %s", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	_, _, decodeErr := image.Decode(f)
	if err = errors.Join(decodeErr, f.Close()); err != nil {
		return "", fmt.Errorf("unsupported or invalid raster image %s: %w", path, err)
	}
	return path, nil
}
