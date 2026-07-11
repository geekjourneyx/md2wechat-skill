package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/layoutcatalog"
	"golang.org/x/net/html"
)

type e2eAPIEnvelope struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		HTML string `json:"html"`
	} `json:"data"`
}

type e2eWitness struct {
	Module   string
	Variant  string
	Markdown string
	Probe    string
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func e2eGate(t *testing.T) {
	t.Helper()
	if os.Getenv("MD2WECHAT_E2E") != "1" {
		t.Skip("set MD2WECHAT_E2E=1 to enable")
	}
	if os.Getenv("MD2WECHAT_BASE_URL") == "" {
		t.Skip("MD2WECHAT_BASE_URL not set")
	}
}

func TestDecodeE2EResponse(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{name: "non-200", status: http.StatusBadGateway, body: `{}`, want: "http status"},
		{name: "invalid JSON", status: http.StatusOK, body: `{`, want: "decode envelope"},
		{name: "non-zero API code", status: http.StatusOK, body: `{"code":42,"msg":"rejected","data":{"html":"x"}}`, want: "api code"},
		{name: "empty HTML", status: http.StatusOK, body: `{"code":0,"data":{"html":""}}`, want: "empty html"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tt.status, Body: io.NopCloser(strings.NewReader(tt.body))}
			_, err := decodeE2EResponse(resp)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("decodeE2EResponse() error = %v, want category %q", err, tt.want)
			}
		})
	}
}

func TestConformanceResponse(t *testing.T) {
	const markdown = ":::demo\ntitle: Visible probe\n:::\n"
	tests := []struct {
		name, html, want string
	}{
		{name: "missing marker", html: "<p>Visible probe</p>", want: "action marker"},
		{name: "marker text is not an attribute", html: "<p>data-mpa-action-id Visible probe</p>", want: "action marker"},
		{name: "missing probe", html: `<section data-mpa-action-id="1">other</section>`, want: "visible probe"},
		{name: "probe in attribute is not visible", html: `<section data-mpa-action-id="1" title="Visible probe">other</section>`, want: "visible probe"},
		{name: "raw fence", html: `<section data-mpa-action-id="1">Visible probe :::demo</section>`, want: "raw fence"},
		{name: "success", html: `<section data-mpa-action-id="1">Visible probe</section>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkConformanceHTML(e2eWitness{Module: "demo", Markdown: markdown, Probe: "Visible probe"}, tt.html)
			if tt.want == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("checkConformanceHTML() error = %v, want category %q", err, tt.want)
			}
		})
	}
}

func TestSemanticConformanceRules(t *testing.T) {
	witness := e2eWitness{Module: "image-compare", Probe: "Before"}
	if err := checkSemanticConformance(witness, `<section><img alt="Before"></section>`); err == nil {
		t.Fatal("image-compare with one image must fail its semantic image-count contract")
	}
	if err := checkSemanticConformance(witness, `<section><img alt="Before"><img alt="After"></section>`); err != nil {
		t.Fatal(err)
	}
}

func TestConformanceRequestUsesLocalTransport(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.String() != "http://layout.invalid/api/convert" {
			t.Fatalf("request = %s %s", req.Method, req.URL)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"code":0,"data":{"html":"<section data-mpa-action-id=\"1\">Visible probe</section>"}}`)),
			Request:    req,
		}, nil
	})}
	witness := e2eWitness{Module: "demo", Markdown: ":::demo\ntitle: Visible probe\n:::\n", Probe: "Visible probe"}
	if err := runConformanceRequest(client, "http://layout.invalid", "", witness); err != nil {
		t.Fatal(err)
	}
}

func TestDeterministicWitnessProbe(t *testing.T) {
	tests := []struct {
		name, format, markdown, want string
	}{
		{name: "fields skip variant control", format: layoutcatalog.BodyFormatFields, markdown: ":::demo\nvariant: compact\ntitle: Field probe\n:::\n", want: "Field probe"},
		{name: "JSON object", format: layoutcatalog.BodyFormatJSONObject, markdown: ":::demo\n{\"type\":\"compact\",\"title\":\"JSON probe\"}\n:::\n", want: "JSON probe"},
		{name: "JSON array", format: layoutcatalog.BodyFormatJSONArray, markdown: ":::demo\n[{\"title\":\"Array probe\"}]\n:::\n", want: "Array probe"},
		{name: "rows", format: layoutcatalog.BodyFormatRows, markdown: ":::demo\n Row probe | second\n:::\n", want: "Row probe"},
		{name: "image alt", format: layoutcatalog.BodyFormatMarkdownImages, markdown: ":::demo\n![Image probe](https://example.com/a.png)\n:::\n", want: "Image probe"},
		{name: "split", format: layoutcatalog.BodyFormatSplit, markdown: ":::demo\nSplit probe\n---\nright\n:::\n", want: "Split probe"},
		{name: "lines", format: layoutcatalog.BodyFormatLines, markdown: ":::demo\nLine probe\n:::\n", want: "Line probe"},
		{name: "dialogue", format: layoutcatalog.BodyFormatDialogue, markdown: ":::demo\nA: Dialogue probe\n:::\n", want: "A: Dialogue probe"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := deterministicWitnessProbe(tt.format, tt.markdown)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("probe = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCollectE2EWitnessesIncludesCanonicalAndVariants(t *testing.T) {
	c := layoutcatalog.NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	builtin, ok := c.Get("hero")
	if !ok {
		t.Fatal("hero builtin not found")
	}
	spec := *builtin
	spec.Example = ":::hero\neyebrow: Test\ntitle: Canonical probe\n:::\n"
	spec.Variants = []layoutcatalog.VariantSpec{{
		Name: "compact", Example: ":::hero\nvariant: compact\neyebrow: Test\ntitle: Variant probe\n:::\n",
	}}
	witnesses, err := witnessesForSpec(c, &spec)
	if err != nil {
		t.Fatal(err)
	}
	if len(witnesses) != 2 || witnesses[0].Variant != "" || witnesses[1].Variant != "compact" {
		t.Fatalf("witnesses = %+v", witnesses)
	}
}

func TestE2ELayoutConformance(t *testing.T) {
	e2eGate(t)
	c, err := layoutcatalog.DefaultCatalog()
	if err != nil {
		t.Fatal(err)
	}
	client := http.DefaultClient
	baseURL := os.Getenv("MD2WECHAT_BASE_URL")
	apiKey := os.Getenv("MD2WECHAT_API_KEY")
	for _, lifecycle := range []string{layoutcatalog.LifecycleRecommended, layoutcatalog.LifecycleCompatibility} {
		for _, spec := range c.ListFiltered(layoutcatalog.ListFilter{Lifecycle: lifecycle}) {
			witnesses, err := witnessesForSpec(c, spec)
			if err != nil {
				t.Errorf("%s: %v", spec.Name, err)
				continue
			}
			for _, witness := range witnesses {
				witness := witness
				name := witness.Module
				if witness.Variant != "" {
					name += "/" + witness.Variant
				}
				t.Run(name, func(t *testing.T) {
					if err := runConformanceRequest(client, baseURL, apiKey, witness); err != nil {
						t.Fatal(err)
					}
				})
			}
		}
	}
}

func TestE2EOpinionPieceFixture(t *testing.T) {
	e2eGate(t)
	data, err := os.ReadFile("../../internal/layoutcatalog/testdata/integration/opinion-piece.md")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := postConvert(http.DefaultClient, os.Getenv("MD2WECHAT_BASE_URL"), os.Getenv("MD2WECHAT_API_KEY"), string(data))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeE2EResponse(resp); err != nil {
		t.Fatal(err)
	}
}

func TestE2EValidatorVsAPIConsistency(t *testing.T) {
	e2eGate(t)
	bad := ":::hero\neyebrow: only\n:::\n"
	c, err := layoutcatalog.DefaultCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if r := c.Validate(bad); len(r.Errors) == 0 {
		t.Fatal("validator should flag missing title")
	}
	resp, err := postConvert(http.DefaultClient, os.Getenv("MD2WECHAT_BASE_URL"), os.Getenv("MD2WECHAT_API_KEY"), bad)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeE2EResponse(resp); err == nil {
		t.Log("WARNING: validator rejected witness that API accepted")
	}
}

var witnessImageRE = regexp.MustCompile(`!\[([^]]*)\]\((?:https?://[^)\s]+|<https?://[^>]+>)\)`)

func decodeE2EResponse(resp *http.Response) (string, error) {
	if resp == nil {
		return "", fmt.Errorf("nil response")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("http status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var envelope e2eAPIEnvelope
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&envelope); err != nil {
		return "", fmt.Errorf("decode envelope: %w", err)
	}
	if envelope.Code != 0 {
		return "", fmt.Errorf("api code %d: %s", envelope.Code, envelope.Msg)
	}
	html := strings.TrimSpace(envelope.Data.HTML)
	if html == "" {
		return "", fmt.Errorf("empty html in successful response")
	}
	return html, nil
}

func checkConformanceHTML(witness e2eWitness, html string) error {
	doc, err := htmlpkgParse(html)
	if err != nil {
		return fmt.Errorf("%s response is not parseable HTML: %w", witness.Module, err)
	}
	if !hasDOMAttribute(doc, "data-mpa-action-id") {
		return fmt.Errorf("%s response missing action marker", witness.Module)
	}
	probe := strings.TrimSpace(witness.Probe)
	if probe == "" {
		return fmt.Errorf("%s witness has no visible probe", witness.Module)
	}
	if !strings.Contains(strings.TrimSpace(visibleDOMText(doc)), probe) {
		return fmt.Errorf("%s response missing visible probe %q", witness.Module, probe)
	}
	if strings.Contains(html, ":::"+witness.Module) {
		return fmt.Errorf("%s response retained raw fence", witness.Module)
	}
	return checkSemanticConformance(witness, html)
}

func htmlpkgParse(rendered string) (*html.Node, error) {
	return html.Parse(strings.NewReader(rendered))
}

func hasDOMAttribute(node *html.Node, name string) bool {
	if node.Type == html.ElementNode {
		for _, attr := range node.Attr {
			if attr.Key == name {
				return true
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if hasDOMAttribute(child, name) {
			return true
		}
	}
	return false
}

func visibleDOMText(node *html.Node) string {
	if node.Type == html.ElementNode && (node.Data == "script" || node.Data == "style") {
		return ""
	}
	if node.Type == html.TextNode {
		return node.Data
	}
	var text strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		text.WriteString(visibleDOMText(child))
	}
	return text.String()
}

func checkSemanticConformance(witness e2eWitness, rendered string) error {
	minimumImages := 0
	switch witness.Module {
	case "image-compare", "svg-swipe-gallery":
		minimumImages = 2
	case "gallery-grid", "gallery-story", "image-phone-shot", "figure-caption":
		minimumImages = 1
	default:
		return nil
	}
	doc, err := html.Parse(strings.NewReader(rendered))
	if err != nil {
		return fmt.Errorf("%s response is not parseable HTML: %w", witness.Module, err)
	}
	if count := countDOMElements(doc, "img"); count < minimumImages {
		return fmt.Errorf("%s response has %d image element(s), want at least %d", witness.Module, count, minimumImages)
	}
	return nil
}

func countDOMElements(node *html.Node, element string) int {
	count := 0
	if node.Type == html.ElementNode && node.Data == element {
		count++
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		count += countDOMElements(child, element)
	}
	return count
}

func runConformanceRequest(client *http.Client, baseURL, apiKey string, witness e2eWitness) error {
	resp, err := postConvert(client, baseURL, apiKey, witness.Markdown)
	if err != nil {
		return err
	}
	html, err := decodeE2EResponse(resp)
	if err != nil {
		return err
	}
	return checkConformanceHTML(witness, html)
}

func postConvert(client *http.Client, baseURL, apiKey, markdown string) (*http.Response, error) {
	body, err := json.Marshal(map[string]string{"markdown": markdown})
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/convert", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	return resp, nil
}

func witnessesForSpec(c *layoutcatalog.Catalog, spec *layoutcatalog.LayoutSpec) ([]e2eWitness, error) {
	declared := make([]struct {
		variant, markdown, assertion string
	}, 0, 1+len(spec.Variants))
	declared = append(declared, struct {
		variant, markdown, assertion string
	}{markdown: spec.Example, assertion: spec.ExampleAssertContains})
	for _, variant := range spec.Variants {
		declared = append(declared, struct {
			variant, markdown, assertion string
		}{variant: variant.Name, markdown: variant.Example, assertion: variant.AssertContains})
	}

	witnesses := make([]e2eWitness, 0, len(declared))
	for _, item := range declared {
		if strings.TrimSpace(item.markdown) == "" {
			return nil, fmt.Errorf("%s witness %q has an empty example", spec.Name, item.variant)
		}
		if report := c.Validate(item.markdown); len(report.Errors) != 0 {
			return nil, fmt.Errorf("%s witness %q does not validate: %v", spec.Name, item.variant, report.Errors)
		}
		probe, err := deterministicWitnessProbe(spec.BodyFormat, item.markdown)
		if err != nil {
			return nil, fmt.Errorf("%s witness %q probe: %w", spec.Name, item.variant, err)
		}
		if probe == "" {
			probe = strings.TrimSpace(item.assertion)
		}
		if probe == "" {
			return nil, fmt.Errorf("%s witness %q has no deterministic probe or explicit assertion", spec.Name, item.variant)
		}
		witnesses = append(witnesses, e2eWitness{Module: spec.Name, Variant: item.variant, Markdown: item.markdown, Probe: probe})
	}
	return witnesses, nil
}

func deterministicWitnessProbe(format, markdown string) (string, error) {
	body, err := firstWitnessBody(markdown)
	if err != nil {
		return "", err
	}
	switch format {
	case layoutcatalog.BodyFormatFields, layoutcatalog.BodyFormatMarkdownFields:
		for _, line := range body {
			key, value, ok := strings.Cut(strings.TrimSpace(strings.TrimRight(line, "\r")), ":")
			if !ok || key == "variant" || key == "type" {
				continue
			}
			if value = strings.TrimSpace(value); value != "" {
				return value, nil
			}
		}
	case layoutcatalog.BodyFormatJSONObject, layoutcatalog.BodyFormatJSONArray:
		dec := json.NewDecoder(strings.NewReader(strings.Join(body, "\n")))
		return firstJSONScalar(dec, "")
	case layoutcatalog.BodyFormatRows:
		for _, line := range body {
			line = strings.TrimSpace(strings.TrimRight(line, "\r"))
			if line == "" {
				continue
			}
			cell, _, _ := strings.Cut(line, "|")
			return strings.TrimSpace(cell), nil
		}
	case layoutcatalog.BodyFormatMarkdownImages:
		for _, line := range body {
			match := witnessImageRE.FindStringSubmatch(line)
			if len(match) == 2 && strings.TrimSpace(match[1]) != "" {
				return strings.TrimSpace(match[1]), nil
			}
		}
	case layoutcatalog.BodyFormatSplit, layoutcatalog.BodyFormatLines, layoutcatalog.BodyFormatDialogue:
		for _, line := range body {
			line = strings.TrimSpace(strings.TrimRight(line, "\r"))
			if line != "" && line != "---" {
				return line, nil
			}
		}
	default:
		return "", fmt.Errorf("unsupported body format %q", format)
	}
	return "", nil
}

func firstJSONScalar(dec *json.Decoder, key string) (string, error) {
	token, err := dec.Token()
	if err != nil {
		return "", err
	}
	if delim, ok := token.(json.Delim); ok {
		switch delim {
		case '{':
			for dec.More() {
				keyToken, err := dec.Token()
				if err != nil {
					return "", err
				}
				value, err := firstJSONScalar(dec, keyToken.(string))
				if err != nil {
					return "", err
				}
				if value != "" {
					return value, nil
				}
			}
			_, err = dec.Token()
			return "", err
		case '[':
			for dec.More() {
				value, err := firstJSONScalar(dec, key)
				if err != nil {
					return "", err
				}
				if value != "" {
					return value, nil
				}
			}
			_, err = dec.Token()
			return "", err
		}
	}
	if key == "variant" || key == "type" || token == nil {
		return "", nil
	}
	return strings.TrimSpace(fmt.Sprint(token)), nil
}

func firstWitnessBody(markdown string) ([]string, error) {
	lines := strings.Split(strings.TrimSpace(markdown), "\n")
	if len(lines) < 2 || !strings.HasPrefix(strings.TrimRight(lines[0], " \t\r"), ":::") {
		return nil, fmt.Errorf("missing witness opener")
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(strings.TrimRight(lines[i], "\r")) == ":::" {
			return lines[1:i], nil
		}
	}
	return nil, fmt.Errorf("missing witness closing fence")
}
