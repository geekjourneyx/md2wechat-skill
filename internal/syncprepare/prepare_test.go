package syncprepare

import (
	"image"
	"image/png"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPrepareSemanticContent(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "local image.png")
	f, err := os.Create(img)
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(f, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	source := "---\ntitle: Metadata title\n---\n# Heading\n\nBody [link](https://example.com).\n\n## Detail\n\n| A | B |\n|---|---|\n| 1 | 2 |\n\n- item\n\n![local][pic]\n\n[pic]: <local%20image.png>\n\n```html\n<div>[[x]] :::hero __generate:foo</div>\n```\n\n`[[x]]` and `<b>`\n"
	article := filepath.Join(dir, "article.md")
	writeSource(t, article, source)
	result, err := Prepare(article, filepath.Join(dir, "out"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Title != "Metadata title" || !reflect.DeepEqual(result.HeadingLevels, []int{1, 2}) || !reflect.DeepEqual(result.Images, []string{img}) {
		t.Fatalf("%+v", result)
	}
	body, _ := os.ReadFile(result.BodyHTML)
	for _, want := range []string{"<table>", "<h1>Heading</h1>", "<h2>Detail</h2>", "<ul>", "&lt;div&gt;", "src=\"" + strings.ReplaceAll(img, " ", "%20") + "\""} {
		if !strings.Contains(string(body), want) {
			t.Errorf("missing %s in %s", want, body)
		}
	}
	if strings.Contains(string(body), "title: Metadata") {
		t.Fatal("frontmatter leaked")
	}
	unchanged, _ := os.ReadFile(article)
	if string(unchanged) != source {
		t.Fatal("source changed")
	}
	entries, _ := os.ReadDir(filepath.Dir(result.BodyHTML))
	if len(entries) != 1 {
		t.Fatal("extra artifacts")
	}
}

func TestPreparePreservesEscapedImageIdentity(t *testing.T) {
	for _, name := range []string{"literal%20name.png", "a&copy;.png", "a(b).png", "a#b.png"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, name)
			f, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			if err = png.Encode(f, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
				t.Fatal(err)
			}
			if err = f.Close(); err != nil {
				t.Fatal(err)
			}
			article := filepath.Join(dir, "a.md")
			writeSource(t, article, "# T\n\n![image](<"+(&url.URL{Path: name}).EscapedPath()+">)")
			// Ampersands in Markdown destinations must be escaped to remain literal.
			if strings.Contains(name, "&") {
				writeSource(t, article, "# T\n\n![image](<"+strings.ReplaceAll((&url.URL{Path: name}).EscapedPath(), "&", "%26")+">)")
			}
			result, err := Prepare(article, filepath.Join(dir, "out"))
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Images) != 1 || result.Images[0] != path {
				t.Fatalf("wrong local file: %v", result.Images)
			}
			body, _ := os.ReadFile(result.BodyHTML)
			want := strings.ReplaceAll((&url.URL{Path: path}).EscapedPath(), "&", "%26")
			if !strings.Contains(string(body), `src="`+want+`"`) {
				t.Fatalf("image identity changed: %s", body)
			}
		})
	}
}

func TestPrepareTitleAndOrdinaryPunctuation(t *testing.T) {
	dir := t.TempDir()
	article := filepath.Join(dir, "a.md")
	source := "# A \\*literal\\* &amp; `&amp;` \\&amp; <https://example.com>\n\nOrdinary punctuation: ]] and [! wow].\n"
	writeSource(t, article, source)
	result, err := Prepare(article, filepath.Join(dir, "out"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Title != "A *literal* & &amp; &amp; https://example.com" {
		t.Fatalf("title changed meaning: %q", result.Title)
	}
}

func TestPrepareRejectsTruncatedImage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(path, 33); err != nil {
		t.Fatal(err)
	}
	article := filepath.Join(dir, "article.md")
	writeSource(t, article, "# Title\n\n![image](broken.png)")
	out := filepath.Join(dir, "out")
	if _, err := Prepare(article, out); err == nil {
		t.Fatal("accepted PNG header without image data")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("created output for corrupt image")
	}
}

func TestPrepareRejectsWithoutOutput(t *testing.T) {
	for name, source := range map[string]string{
		"empty": "", "title only": "# Title\n", "code title only": "# `Title`\n", "no title": "Body", "fake heading": "```\n# Title\n```\nBody",
		"unclosed metadata": "---\ntitle: T\n# Heading\nBody", "broken metadata": "---\ntitle: [broken\n---\nBody", "numeric title": "---\ntitle: 123\n---\nBody", "blank title": "---\ntitle: ''\n---\n# T\nBody",
		"html": "# T\n\n<div>Body</div>", "inline html": "# T\n\nBody <b>bold</b>", "layout": "# T\n\n:::hero\nBody\n:::", "wiki": "# T\n\n[[Some page]]", "embed": "# T\n\n![[image.png]]", "callout": "# T\n\n> [!NOTE]\n> Body", "generated": "# T\n\n![x](__generate:cat__)",
		"remote": "# T\n\n![x](https://example.com/a.png)", "unsafe link": "# T\n\n[click](javascript:alert)", "unsafe autolink": "# T\n\n<javascript:alert>", "missing image": "# T\n\n![x](missing.png)",
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			article := filepath.Join(dir, "a.md")
			out := filepath.Join(dir, "out")
			writeSource(t, article, source)
			if _, err := Prepare(article, out); err == nil {
				t.Fatal("accepted invalid article")
			}
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Fatal("created output for rejected source")
			}
		})
	}
}

func TestPrepareExistingOutputAndInvalidRaster(t *testing.T) {
	dir := t.TempDir()
	article := filepath.Join(dir, "a.md")
	writeSource(t, article, "# T\n\nBody")
	sentinel := filepath.Join(dir, "body.html")
	writeSource(t, sentinel, "keep")
	if _, err := Prepare(article, dir); err == nil {
		t.Fatal("overwrote existing directory")
	}
	b, _ := os.ReadFile(sentinel)
	if string(b) != "keep" {
		t.Fatal("changed existing file")
	}
	writeSource(t, article, "# T\n\n![x](body.html)")
	if _, err := Prepare(article, filepath.Join(dir, "out")); err == nil {
		t.Fatal("accepted non-image")
	}
}

func TestPrepareHeadingTitle(t *testing.T) {
	for _, source := range []string{"# A **real** title\n\nBody", "# A `real` title\n\nBody", "A **real** title\n===\n\nBody", "---\nauthor: Someone\n---\n# A **real** title\n\nBody"} {
		dir := t.TempDir()
		article := filepath.Join(dir, "a.md")
		writeSource(t, article, source)
		result, err := Prepare(article, filepath.Join(dir, "out"))
		if err != nil {
			t.Fatal(err)
		}
		if result.Title != "A real title" {
			t.Fatalf("title %q", result.Title)
		}
	}
}

func writeSource(t *testing.T, path, source string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
}
