package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
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
	Module          string
	Variant         string
	Markdown        string
	Probe           string
	ProbeInImageAlt bool
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

const layoutConformanceBaseURL = "https://md2wechat.app"

type e2eSettings struct {
	BaseURL   string
	APIKey    string
	CLICommit string
}

func e2eGate(t *testing.T) e2eSettings {
	t.Helper()
	if os.Getenv("MD2WECHAT_E2E") != "1" {
		t.Skip("set MD2WECHAT_E2E=1 to enable")
	}
	settings, err := loadE2ESettings()
	if err != nil {
		t.Fatal(err)
	}
	return settings
}

func loadE2ESettings() (e2eSettings, error) {
	config.SetQuiet(true)
	cfg, err := config.Load()
	if err != nil {
		return e2eSettings{}, fmt.Errorf("load E2E configuration: %w", err)
	}
	settings := e2eSettings{
		BaseURL:   layoutConformanceBaseURL,
		APIKey:    strings.TrimSpace(cfg.MD2WechatAPIKey),
		CLICommit: strings.TrimSpace(os.Getenv("MD2WECHAT_CLI_COMMIT")),
	}
	if settings.APIKey == "" {
		return e2eSettings{}, fmt.Errorf("authentication failure: MD2WECHAT_API_KEY is not configured")
	}
	if settings.CLICommit == "" {
		return e2eSettings{}, fmt.Errorf("MD2WECHAT_CLI_COMMIT is required for conformance evidence")
	}
	return settings, nil
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
		{name: "missing marker", html: "<p>Visible probe</p>", want: "module marker"},
		{name: "marker text is not an attribute", html: "<p>data-mpa-action-id Visible probe</p>", want: "module marker"},
		{name: "wrong module marker", html: `<section data-mpa-action-id="other">Visible probe</section>`, want: "module marker"},
		{name: "missing probe", html: `<section data-mpa-action-id="demo">other</section>`, want: "visible probe"},
		{name: "probe in attribute is not visible", html: `<section data-mpa-action-id="demo" title="Visible probe">other</section>`, want: "visible probe"},
		{name: "raw fence", html: `<section data-mpa-action-id="demo">Visible probe :::demo</section>`, want: "raw fence"},
		{name: "success", html: `<section data-mpa-action-id="demo">Visible probe</section>`},
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

func TestVariantConformanceRequiresExactRendererBranch(t *testing.T) {
	tests := []struct {
		module, variant, attribute, value string
	}{
		{module: "hero", variant: "editorial", attribute: "data-hero-variant", value: "editorial"},
		{module: "quote", variant: "proof", attribute: "data-quote-variant", value: "proof"},
		{module: "summary", variant: "decision", attribute: "data-summary-variant", value: "decision"},
		{module: "cta", variant: "trial", attribute: "data-cta-variant", value: "trial"},
		{module: "infographic", variant: "mini-case", attribute: "data-infographic-type", value: "micro-case"},
	}
	for _, tt := range tests {
		t.Run(tt.module+"/"+tt.variant, func(t *testing.T) {
			witness := e2eWitness{Module: tt.module, Variant: tt.variant, Probe: "Probe"}
			valid := fmt.Sprintf(`<section data-mpa-action-id="%s" %s="%s">Probe</section>`, tt.module, tt.attribute, tt.value)
			if err := checkConformanceHTML(witness, valid); err != nil {
				t.Fatal(err)
			}
			fallback := fmt.Sprintf(`<section data-mpa-action-id="%s">Probe</section>`, tt.module)
			if err := checkConformanceHTML(witness, fallback); err == nil || !strings.Contains(err.Error(), tt.attribute) {
				t.Fatalf("fallback error = %v, want exact branch attribute", err)
			}
			wrong := fmt.Sprintf(`<section data-mpa-action-id="%s" %s="fallback">Probe</section>`, tt.module, tt.attribute)
			if err := checkConformanceHTML(witness, wrong); err == nil || !strings.Contains(err.Error(), tt.value) {
				t.Fatalf("wrong branch error = %v, want value %q", err, tt.value)
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

func TestSVGGallerySemanticConformanceCountsSVGImageElements(t *testing.T) {
	witness := e2eWitness{Module: "svg-swipe-gallery", Probe: "第一张"}
	tests := []struct {
		name     string
		rendered string
		wantErr  bool
	}{
		{name: "two SVG images", rendered: `<svg><image href="one"></image><image href="two"></image></svg>`},
		{name: "one SVG image", rendered: `<svg><image href="one"></image></svg>`, wantErr: true},
		{name: "HTML images do not satisfy SVG branch", rendered: `<div><img src="one"><img src="two"></div>`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkSemanticConformance(witness, tt.rendered)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkSemanticConformance() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestImageWitnessRetainsStableAltContent(t *testing.T) {
	witness := e2eWitness{Module: "gallery", Probe: "图一", ProbeInImageAlt: true}
	if err := checkConformanceHTML(witness, `<section data-mpa-action-id="gallery"><img alt="图一"></section>`); err != nil {
		t.Fatal(err)
	}
	if err := checkConformanceHTML(witness, `<section data-mpa-action-id="gallery"><img alt="其他"></section>`); err == nil || !strings.Contains(err.Error(), "image alt probe") {
		t.Fatalf("missing image alt error = %v", err)
	}
}

func TestConformanceRequestUsesLocalTransport(t *testing.T) {
	const markdown = ":::demo\ntitle: Visible probe\n:::\n"
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.String() != "http://layout.invalid/api/convert" {
			t.Fatalf("request = %s %s", req.Method, req.URL)
		}
		if got := req.Header.Get("X-API-Key"); got != "secret" {
			t.Fatalf("X-API-Key = %q", got)
		}
		if got := req.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization must be empty, got %q", got)
		}
		if got := req.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q", got)
		}
		var payload map[string]string
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["markdown"] != markdown {
			t.Fatalf("markdown payload = %q", payload["markdown"])
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"code":0,"data":{"html":"<section data-mpa-action-id=\"demo\">Visible probe</section>"}}`)),
			Request:    req,
		}, nil
	})}
	witness := e2eWitness{Module: "demo", Markdown: markdown, Probe: "Visible probe"}
	if _, err := runConformanceRequest(client, "http://layout.invalid", "secret", witness); err != nil {
		t.Fatal(err)
	}
}

func TestE2ESettingsUseProductionTargetAndConfigCredential(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MD2WECHAT_API_KEY", "environment-key")
	t.Setenv("MD2WECHAT_BASE_URL", "https://wrong.example")
	t.Setenv("MD2WECHAT_CLI_COMMIT", "cli-commit")
	dir := filepath.Join(home, ".config", "md2wechat")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("api:\n  md2wechat_key: file-key\n"), 0600); err != nil {
		t.Fatal(err)
	}
	settings, err := loadE2ESettings()
	if err != nil {
		t.Fatal(err)
	}
	if settings.BaseURL != "https://md2wechat.app" {
		t.Fatalf("BaseURL = %q", settings.BaseURL)
	}
	if settings.APIKey != "environment-key" || settings.CLICommit != "cli-commit" {
		t.Fatalf("settings did not honor safe config/env precedence")
	}
}

func TestE2EFailureCategories(t *testing.T) {
	auth := &http.Response{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"code":401,"msg":"invalid"}`))}
	if _, err := decodeE2EResponse(auth); err == nil || !strings.Contains(err.Error(), "authentication failure") {
		t.Fatalf("authentication error = %v", err)
	}
	api := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"code":42,"msg":"changed"}`))}
	if _, err := decodeE2EResponse(api); err == nil || !strings.Contains(err.Error(), "API drift") {
		t.Fatalf("API drift error = %v", err)
	}
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, io.ErrUnexpectedEOF
	})}
	if _, err := postConvert(client, "https://md2wechat.app", "secret", "body"); err == nil || !strings.Contains(err.Error(), "network failure") {
		t.Fatalf("network error = %v", err)
	}
}

func TestRemoteBuildIdentityPrefersResponseHeader(t *testing.T) {
	header := make(http.Header)
	header.Set("X-API-Build-ID", "remote-build")
	if got, ok := remoteBuildIdentity(header); !ok || got != "remote-build" {
		t.Fatalf("remote identity = %q, %v", got, ok)
	}
	if got, ok := remoteBuildIdentity(make(http.Header)); ok || got != "" {
		t.Fatalf("missing remote identity = %q, %v", got, ok)
	}
}

func TestDeterministicWitnessProbe(t *testing.T) {
	tests := []struct {
		name, format, markdown, want string
	}{
		{name: "fields skip variant control", format: layoutcatalog.BodyFormatFields, markdown: ":::demo\nvariant: compact\ntitle: Field probe\n:::\n", want: "Field probe"},
		{name: "markdown fields", format: layoutcatalog.BodyFormatMarkdownFields, markdown: ":::demo\nvariant: compact\ntitle: Markdown field probe\n:::\n", want: "Markdown field probe"},
		{name: "JSON object", format: layoutcatalog.BodyFormatJSONObject, markdown: ":::demo\n{\"type\":\"compact\",\"title\":\"JSON probe\"}\n:::\n", want: "JSON probe"},
		{name: "JSON array", format: layoutcatalog.BodyFormatJSONArray, markdown: ":::demo\n[{\"title\":\"Array probe\"}]\n:::\n", want: "Array probe"},
		{name: "rows", format: layoutcatalog.BodyFormatRows, markdown: ":::demo\n Row probe | second\n:::\n", want: "Row probe"},
		{name: "image alt", format: layoutcatalog.BodyFormatMarkdownImages, markdown: ":::demo\n![Image probe](https://example.com/a.png)\n:::\n", want: "Image probe"},
		{name: "split", format: layoutcatalog.BodyFormatSplit, markdown: ":::demo\nSplit probe\n---\nright\n:::\n", want: "Split probe"},
		{name: "lines", format: layoutcatalog.BodyFormatLines, markdown: ":::demo\nLine probe\n:::\n", want: "Line probe"},
		{name: "dialogue", format: layoutcatalog.BodyFormatDialogue, markdown: ":::demo\nA: Dialogue probe\n:::\n", want: "A: Dialogue probe"},
	}
	testedFormats := make(map[string]bool, len(tests))
	for _, tt := range tests {
		testedFormats[tt.format] = true
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
	for format := range layoutcatalog.ValidBodyFormats {
		if !testedFormats[format] {
			t.Errorf("body format %q has no deterministic probe case", format)
		}
	}
	if len(testedFormats) != len(layoutcatalog.ValidBodyFormats) {
		t.Fatalf("probe formats = %v, valid formats = %v", testedFormats, layoutcatalog.ValidBodyFormats)
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
		Name: "editorial", Example: ":::hero\nvariant: editorial\neyebrow: Test\ntitle: Variant probe\n:::\n",
	}}
	witnesses, err := witnessesForSpec(c, &spec)
	if err != nil {
		t.Fatal(err)
	}
	if len(witnesses) != 2 || witnesses[0].Variant != "" || witnesses[1].Variant != "editorial" {
		t.Fatalf("witnesses = %+v", witnesses)
	}
}

func TestCollectE2EWitnessesPrefersExplicitAssertion(t *testing.T) {
	c := layoutcatalog.NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	builtin, _ := c.Get("hero")
	spec := *builtin
	spec.Example = ":::hero\neyebrow: Derived but not visible\ntitle: Explicit visible\n:::\n"
	spec.ExampleAssertContains = "Explicit visible"
	witnesses, err := witnessesForSpec(c, &spec)
	if err != nil {
		t.Fatal(err)
	}
	if got := witnesses[0].Probe; got != "Explicit visible" {
		t.Fatalf("probe = %q, want explicit assertion", got)
	}
	if err := checkConformanceHTML(witnesses[0], `<section data-mpa-action-id="hero">Explicit visible</section>`); err != nil {
		t.Fatalf("explicit visible assertion should pass without derived probe: %v", err)
	}
}

func TestCollectE2EImageWitnessPrefersExplicitVisibleAssertion(t *testing.T) {
	c := layoutcatalog.NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	builtin, _ := c.Get("gallery-grid")
	spec := *builtin
	spec.Example = ":::gallery-grid\n![Derived alt](https://example.com/a.png) | Explicit visible caption\n:::\n"
	spec.ExampleAssertContains = "Explicit visible caption"
	witnesses, err := witnessesForSpec(c, &spec)
	if err != nil {
		t.Fatal(err)
	}
	if got := witnesses[0]; got.Probe != "Explicit visible caption" || got.ProbeInImageAlt {
		t.Fatalf("image witness = %+v, want explicit visible probe", got)
	}
	if err := checkConformanceHTML(witnesses[0], `<section data-mpa-action-id="gallery-grid"><img alt="Derived alt"><p>Explicit visible caption</p></section>`); err != nil {
		t.Fatal(err)
	}
}

func TestCollectE2EWitnessesBindsModuleAndVariant(t *testing.T) {
	c := layoutcatalog.NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	builtin, _ := c.Get("hero")
	tests := []struct {
		name    string
		example string
		variant layoutcatalog.VariantSpec
		wantErr bool
	}{
		{name: "wrong canonical opener", example: ":::part\neyebrow: Test\ntitle: Probe\n:::\n", wantErr: true},
		{name: "variant missing selector", example: ":::hero\neyebrow: Test\ntitle: Probe\n:::\n", variant: layoutcatalog.VariantSpec{Name: "editorial"}, wantErr: true},
		{name: "variant canonical selector", example: ":::hero\nvariant: editorial\neyebrow: Test\ntitle: Probe\n:::\n", variant: layoutcatalog.VariantSpec{Name: "editorial"}},
		{name: "variant alias selector", example: ":::hero\nvariant: story\neyebrow: Test\ntitle: Probe\n:::\n", variant: layoutcatalog.VariantSpec{Name: "editorial", Aliases: []string{"story"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := *builtin
			if tt.variant.Name == "" {
				spec.Example = tt.example
			} else {
				spec.Example = ":::hero\neyebrow: Test\ntitle: Canonical\n:::\n"
				tt.variant.Example = tt.example
				spec.Variants = []layoutcatalog.VariantSpec{tt.variant}
			}
			_, err := witnessesForSpec(c, &spec)
			if (err != nil) != tt.wantErr {
				t.Fatalf("witnessesForSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuiltinExecutableWitnessProbesAreComplete(t *testing.T) {
	c := layoutcatalog.NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	for _, spec := range c.ListFiltered(layoutcatalog.ListFilter{}) {
		t.Run(spec.Name, func(t *testing.T) {
			witnesses, err := witnessesForSpec(c, spec)
			if err != nil {
				t.Fatal(err)
			}
			if len(witnesses) != 1+len(spec.Variants) {
				t.Fatalf("witness count = %d, want %d", len(witnesses), 1+len(spec.Variants))
			}
			for _, witness := range witnesses {
				if strings.TrimSpace(witness.Probe) == "" {
					t.Fatalf("%s/%s has an empty probe", witness.Module, witness.Variant)
				}
			}
		})
	}
}

func TestE2ELayoutConformance(t *testing.T) {
	runE2ELifecycle(t, layoutcatalog.LifecycleRecommended)
}

func TestE2ECompatibilityConformance(t *testing.T) {
	runE2ELifecycle(t, layoutcatalog.LifecycleCompatibility)
}

func runE2ELifecycle(t *testing.T, lifecycle string) {
	t.Helper()
	settings := e2eGate(t)
	c, err := layoutcatalog.DefaultCatalog()
	if err != nil {
		t.Fatal(err)
	}
	client := http.DefaultClient
	var remoteIdentity string
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
				identity, err := runConformanceRequest(client, settings.BaseURL, settings.APIKey, witness)
				if err != nil {
					t.Fatal(err)
				}
				if remoteIdentity == "" {
					remoteIdentity = identity
				}
			})
		}
	}
	t.Logf("conformance target=%s cli_commit=%s", settings.BaseURL, settings.CLICommit)
	if remoteIdentity != "" {
		t.Logf("remote_build_id=%s source=response_header", remoteIdentity)
	} else {
		t.Logf("remote_identity=target+utc_timestamp target=%s observed_at=%s non_commit_evidence=true", settings.BaseURL, time.Now().UTC().Format(time.RFC3339))
	}
}

func TestE2EOpinionPieceFixture(t *testing.T) {
	settings := e2eGate(t)
	data, err := os.ReadFile("../../internal/layoutcatalog/testdata/integration/opinion-piece.md")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := postConvert(http.DefaultClient, settings.BaseURL, settings.APIKey, string(data))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeE2EResponse(resp); err != nil {
		t.Fatal(err)
	}
}

func TestE2EValidatorVsAPIConsistency(t *testing.T) {
	settings := e2eGate(t)
	bad := ":::hero\neyebrow: only\n:::\n"
	c, err := layoutcatalog.DefaultCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if r := c.Validate(bad); len(r.Errors) == 0 {
		t.Fatal("validator should flag missing title")
	}
	resp, err := postConvert(http.DefaultClient, settings.BaseURL, settings.APIKey, bad)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeE2EResponse(resp); err == nil {
		t.Fatal("API drift: validator rejected input that the API accepted")
	}
}

var witnessImageRE = regexp.MustCompile(`!\[([^]]*)\]\((?:https?://[^)\s]+|<https?://[^>]+>)\)`)

func decodeE2EResponse(resp *http.Response) (string, error) {
	if resp == nil {
		return "", fmt.Errorf("nil response")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return "", fmt.Errorf("authentication failure: http status %d", resp.StatusCode)
		}
		return "", fmt.Errorf("API drift: http status %d", resp.StatusCode)
	}
	var envelope e2eAPIEnvelope
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&envelope); err != nil {
		return "", fmt.Errorf("API drift: decode envelope: %w", err)
	}
	if envelope.Code != 0 {
		if envelope.Code == http.StatusUnauthorized || envelope.Code == http.StatusForbidden {
			return "", fmt.Errorf("authentication failure: api code %d: %s", envelope.Code, envelope.Msg)
		}
		return "", fmt.Errorf("API drift: api code %d: %s", envelope.Code, envelope.Msg)
	}
	html := strings.TrimSpace(envelope.Data.HTML)
	if html == "" {
		return "", fmt.Errorf("API drift: empty html in successful response")
	}
	return html, nil
}

func checkConformanceHTML(witness e2eWitness, html string) error {
	doc, err := htmlpkgParse(html)
	if err != nil {
		return fmt.Errorf("%s response is not parseable HTML: %w", witness.Module, err)
	}
	if !hasDOMAttributeValue(doc, "data-mpa-action-id", witness.Module) {
		return fmt.Errorf("API drift: %s response missing module marker data-mpa-action-id=%q", witness.Module, witness.Module)
	}
	probe := strings.TrimSpace(witness.Probe)
	if probe == "" {
		return fmt.Errorf("%s witness has no visible probe", witness.Module)
	}
	if witness.ProbeInImageAlt && !hasImageAltProbe(doc, probe) {
		return fmt.Errorf("%s response missing image alt probe %q", witness.Module, probe)
	}
	if !witness.ProbeInImageAlt && !strings.Contains(strings.TrimSpace(visibleDOMText(doc)), probe) {
		return fmt.Errorf("%s response missing visible probe %q", witness.Module, probe)
	}
	if strings.Contains(html, ":::"+witness.Module) {
		return fmt.Errorf("%s response retained raw fence", witness.Module)
	}
	if attribute, value, ok := expectedVariantBranch(witness); ok && !hasDOMAttributeValue(doc, attribute, value) {
		return fmt.Errorf("%s/%s response missing %s=%q", witness.Module, witness.Variant, attribute, value)
	}
	return checkSemanticConformance(witness, html)
}

var variantBranchAttributes = map[string]string{
	"hero":        "data-hero-variant",
	"quote":       "data-quote-variant",
	"summary":     "data-summary-variant",
	"cta":         "data-cta-variant",
	"infographic": "data-infographic-type",
}

func expectedVariantBranch(witness e2eWitness) (string, string, bool) {
	attribute, ok := variantBranchAttributes[witness.Module]
	if !ok || witness.Variant == "" {
		return "", "", false
	}
	value := witness.Variant
	if witness.Module == "infographic" && value == "mini-case" {
		value = "micro-case"
	}
	return attribute, value, true
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

func hasDOMAttributeValue(node *html.Node, name, value string) bool {
	if node.Type == html.ElementNode {
		for _, attr := range node.Attr {
			if attr.Key == name && attr.Val == value {
				return true
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if hasDOMAttributeValue(child, name, value) {
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

func hasImageAltProbe(node *html.Node, probe string) bool {
	if node.Type == html.ElementNode && node.Data == "img" {
		for _, attr := range node.Attr {
			if attr.Key == "alt" && strings.Contains(attr.Val, probe) {
				return true
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if hasImageAltProbe(child, probe) {
			return true
		}
	}
	return false
}

func checkSemanticConformance(witness e2eWitness, rendered string) error {
	minimumImages := 0
	imageElement := "img"
	switch witness.Module {
	case "svg-swipe-gallery":
		minimumImages = 2
		imageElement = "image"
	case "image-compare":
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
	if count := countDOMElements(doc, imageElement); count < minimumImages {
		return fmt.Errorf("%s response has %d %s element(s), want at least %d", witness.Module, count, imageElement, minimumImages)
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

func runConformanceRequest(client *http.Client, baseURL, apiKey string, witness e2eWitness) (string, error) {
	resp, err := postConvert(client, baseURL, apiKey, witness.Markdown)
	if err != nil {
		return "", err
	}
	identity, _ := remoteBuildIdentity(resp.Header)
	html, err := decodeE2EResponse(resp)
	if err != nil {
		return "", err
	}
	if err := checkConformanceHTML(witness, html); err != nil {
		return "", fmt.Errorf("API drift: %w", err)
	}
	return identity, nil
}

func remoteBuildIdentity(header http.Header) (string, bool) {
	for _, name := range []string{"X-API-Build-ID", "X-MD2Wechat-Build-ID", "X-Build-ID", "X-Commit-SHA", "X-Deployment-ID"} {
		if value := strings.TrimSpace(header.Get(name)); value != "" {
			return value, true
		}
	}
	return "", false
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
		req.Header.Set("X-API-Key", apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network failure: send request: %w", err)
	}
	return resp, nil
}

func witnessesForSpec(c *layoutcatalog.Catalog, spec *layoutcatalog.LayoutSpec) ([]e2eWitness, error) {
	declared := make([]struct {
		variant, markdown, assertion string
		aliases                      []string
	}, 0, 1+len(spec.Variants))
	declared = append(declared, struct {
		variant, markdown, assertion string
		aliases                      []string
	}{markdown: spec.Example, assertion: spec.ExampleAssertContains})
	for _, variant := range spec.Variants {
		declared = append(declared, struct {
			variant, markdown, assertion string
			aliases                      []string
		}{variant: variant.Name, aliases: variant.Aliases, markdown: variant.Example, assertion: variant.AssertContains})
	}

	witnesses := make([]e2eWitness, 0, len(declared))
	for _, item := range declared {
		if err := c.ValidateWitness(layoutcatalog.WitnessContract{
			Module: spec.Name, Variant: item.variant, VariantAliases: item.aliases,
			Example: item.markdown, AssertContains: item.assertion,
		}); err != nil {
			return nil, err
		}
		probe := strings.TrimSpace(item.assertion)
		probeInImageAlt := false
		if probe == "" {
			var err error
			probe, err = deterministicWitnessProbe(spec.BodyFormat, item.markdown)
			if err != nil {
				return nil, fmt.Errorf("%s witness %q probe: %w", spec.Name, item.variant, err)
			}
			probeInImageAlt = spec.BodyFormat == layoutcatalog.BodyFormatMarkdownImages
		}
		if probe == "" {
			return nil, fmt.Errorf("%s witness %q has no deterministic probe or explicit assertion", spec.Name, item.variant)
		}
		witnesses = append(witnesses, e2eWitness{
			Module: spec.Name, Variant: item.variant, Markdown: item.markdown,
			Probe: probe, ProbeInImageAlt: probeInImageAlt,
		})
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
