package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"github.com/geekjourneyx/md2wechat-skill/internal/converter"
	"github.com/geekjourneyx/md2wechat-skill/internal/layoutcatalog"
	"golang.org/x/net/html"
)

type e2eWitness struct {
	Module           string
	Variant          string
	EffectiveVariant string
	Markdown         string
	Probe            string
	ProbeInImageAlt  bool
	RowDelimiter     string
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type e2eSettings struct {
	BaseURL             string
	APIKey              string
	CLICommit           string
	ExpectedBuildID     string
	FieldContractSHA    string
	FieldContractResult string
	ConformanceMode     string
}

const layoutConformanceRequestTimeout = 30 * time.Second

const pinnedUpstreamFieldContractSHA = "052346a43deb83d211471bb7b423318f6f6ff6c1"

func layoutConformanceCatalog() (*layoutcatalog.Catalog, error) {
	catalog := layoutcatalog.NewCatalog()
	if err := catalog.Load(); err != nil {
		return nil, err
	}
	return catalog, nil
}

func layoutConformanceHTTPClient() *http.Client {
	return &http.Client{Timeout: layoutConformanceRequestTimeout}
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
		BaseURL:             converter.ResolveAPIConvertURL(cfg.MD2WechatBaseURL),
		APIKey:              strings.TrimSpace(cfg.MD2WechatAPIKey),
		CLICommit:           strings.TrimSpace(os.Getenv("MD2WECHAT_CLI_COMMIT")),
		ExpectedBuildID:     strings.TrimSpace(os.Getenv("MD2WECHAT_API_BUILD_ID")),
		FieldContractSHA:    strings.TrimSpace(os.Getenv("MD2WECHAT_UPSTREAM_FIELD_CONTRACT_SHA")),
		FieldContractResult: strings.TrimSpace(os.Getenv("MD2WECHAT_UPSTREAM_FIELD_CONTRACT_RESULT")),
		ConformanceMode:     strings.TrimSpace(os.Getenv("MD2WECHAT_LAYOUT_CONFORMANCE_MODE")),
	}
	if settings.ConformanceMode == "" {
		settings.ConformanceMode = "smoke"
	}
	if settings.ConformanceMode != "smoke" && settings.ConformanceMode != "release" {
		return e2eSettings{}, fmt.Errorf("invalid layout conformance mode %q", settings.ConformanceMode)
	}
	if settings.APIKey == "" {
		return e2eSettings{}, fmt.Errorf("authentication failure: MD2WECHAT_API_KEY is not configured")
	}
	if settings.CLICommit == "" {
		return e2eSettings{}, fmt.Errorf("MD2WECHAT_CLI_COMMIT is required for conformance evidence")
	}
	if (settings.FieldContractSHA == "") != (settings.FieldContractResult == "") {
		return e2eSettings{}, fmt.Errorf("upstream field-contract evidence requires both SHA and result")
	}
	if settings.FieldContractResult != "" && settings.FieldContractResult != "passed" && settings.FieldContractResult != "failed" {
		return e2eSettings{}, fmt.Errorf("invalid upstream field-contract result %q", settings.FieldContractResult)
	}
	if settings.FieldContractResult == "failed" {
		return e2eSettings{}, fmt.Errorf("upstream field-contract fixture failed at %s", settings.FieldContractSHA)
	}
	if settings.ConformanceMode == "release" {
		if settings.FieldContractSHA == "" || settings.FieldContractResult != "passed" {
			return e2eSettings{}, fmt.Errorf("release conformance requires passed upstream field-contract SHA/result evidence")
		}
		if settings.FieldContractSHA != pinnedUpstreamFieldContractSHA {
			return e2eSettings{}, fmt.Errorf("release conformance must use pinned SHA %s, got %s", pinnedUpstreamFieldContractSHA, settings.FieldContractSHA)
		}
	}
	return settings, nil
}

func validateRemoteBuildIdentity(got, expected, first string) error {
	if expected != "" {
		if got == "" {
			return fmt.Errorf("API drift: expected build id %q but response had no recognized build header", expected)
		}
		if got != expected {
			return fmt.Errorf("API drift: response build id %q, want %q", got, expected)
		}
	}
	if first != "" && got != first {
		return fmt.Errorf("API drift: conflicting response build ids %q and %q", first, got)
	}
	return nil
}

func validateConformanceResult(identity string, responseReceived bool, requestErr error, expectedBuildID, firstBuildID string) error {
	if !responseReceived {
		if requestErr != nil {
			return requestErr
		}
		return fmt.Errorf("network failure: conformance request returned no response")
	}
	if err := validateRemoteBuildIdentity(identity, expectedBuildID, firstBuildID); err != nil {
		return err
	}
	return requestErr
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
		{name: "empty HTML", status: http.StatusOK, body: `{"code":0,"data":{"html":"","theme":"default","fontSize":"medium","backgroundType":"none","wordCount":1,"estimatedReadTime":1}}`, want: "empty HTML"},
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
		{name: "one of multiple markers conforms", html: `<section data-mpa-action-id="demo"></section><section data-mpa-action-id="demo">Visible probe</section>`},
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

func TestConformanceEvidenceMustShareOneModuleSubtree(t *testing.T) {
	tests := []struct {
		name    string
		witness e2eWitness
		html    string
	}{
		{
			name:    "visible probe outside marker",
			witness: e2eWitness{Module: "demo", Probe: "Probe"},
			html:    `<section data-mpa-action-id="demo"></section><p>Probe</p>`,
		},
		{
			name:    "variant on other marker",
			witness: e2eWitness{Module: "hero", Variant: "editorial", Probe: "Probe"},
			html:    `<section data-mpa-action-id="hero">Probe</section><section data-mpa-action-id="other" data-hero-variant="editorial"></section>`,
		},
		{
			name:    "image alt probe outside marker",
			witness: e2eWitness{Module: "gallery", Probe: "Probe", ProbeInImageAlt: true},
			html:    `<section data-mpa-action-id="gallery"></section><section data-mpa-action-id="other"><img alt="Probe"></section>`,
		},
		{
			name:    "images on other marker",
			witness: e2eWitness{Module: "image-compare", Probe: "Probe"},
			html:    `<section data-mpa-action-id="image-compare">Probe</section><section data-mpa-action-id="other"><img><img></section>`,
		},
		{
			name:    "different markers split evidence",
			witness: e2eWitness{Module: "hero", Variant: "editorial", Probe: "Probe"},
			html:    `<section data-mpa-action-id="hero">Probe</section><section data-mpa-action-id="hero" data-hero-variant="editorial"></section>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkConformanceHTML(tt.witness, tt.html); err == nil {
				t.Fatal("split conformance evidence must fail")
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
			if tt.module == "cta" && tt.variant == "trial" {
				witness.Markdown = "points: one"
			}
			valid := fmt.Sprintf(`<section data-mpa-action-id="%s" %s="%s">Probe`, tt.module, tt.attribute, tt.value)
			if tt.module == "cta" {
				valid = ctaSemanticFixture(tt.variant, true)
			}
			valid += `</section>`
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

func TestSemanticConformanceEnforcesHeroMastheadAndExactCTAVariantStructures(t *testing.T) {
	if err := checkSemanticConformance(e2eWitness{Module: "hero", Variant: "masthead"}, `<section></section>`); err == nil {
		t.Fatal("masthead without its structural part must fail")
	}
	if err := checkSemanticConformance(e2eWitness{Module: "hero", Variant: "masthead"}, `<section data-module-part="hero-masthead"></section>`); err != nil {
		t.Fatal(err)
	}
	valid := map[string]string{
		"save-follow": `<section data-cta-variant="save-follow"><section data-module-part="cta-save-confirmation"></section><section data-module-part="cta-save-actions"><span data-cta-action="primary"></span><span data-cta-action="secondary"></span><span data-cta-action="tertiary"></span></section></section>`,
		"consult":     `<section data-cta-variant="consult"><section data-module-part="cta-consult-layout"><section data-module-part="cta-consult-trust"></section><section data-module-part="cta-consult-primary"><span data-cta-action="primary"></span></section><section data-module-part="cta-consult-aux"><span data-cta-action="secondary"></span><span data-cta-action="tertiary"></span></section></section></section>`,
		"trial":       `<section data-cta-variant="trial"><section data-module-part="cta-trial-benefits"></section><section data-module-part="cta-trial-primary"><span data-cta-action="primary"></span></section><section data-module-part="cta-trial-secondary"><span data-cta-action="secondary"></span><span data-cta-action="tertiary"></span></section></section>`,
	}
	for variant, rendered := range valid {
		witness := e2eWitness{Module: "cta", Variant: variant, Markdown: ":::cta\nvariant: " + variant + "\ntitle: Probe\npoints: one\n:::", Probe: "Probe"}
		if err := checkSemanticConformance(witness, rendered); err != nil {
			t.Fatalf("%s valid structure: %v", variant, err)
		}
	}
	for _, rendered := range []string{
		`<section data-cta-variant="save-follow"><section data-module-part="cta-save-actions"><span data-cta-action="primary"></span><span data-cta-action="secondary"></span><span data-cta-action="tertiary"></span></section></section>`,
		`<section data-cta-variant="consult"><section data-module-part="cta-consult-layout"><section data-module-part="cta-consult-primary"><span data-cta-action="primary"></span></section><section data-module-part="cta-consult-aux"><span data-cta-action="secondary"></span><span data-cta-action="tertiary"></span></section></section></section>`,
		`<section data-cta-variant="trial"><section data-module-part="cta-trial-primary"><span data-cta-action="primary"></span></section><section data-module-part="cta-trial-secondary"><span data-cta-action="secondary"></span><span data-cta-action="tertiary"></span></section></section>`,
	} {
		if err := checkSemanticConformance(e2eWitness{Module: "cta", Variant: "trial", Markdown: "points: one"}, rendered); err == nil {
			t.Fatalf("CTA missing exact upstream structure must fail: %s", rendered)
		}
	}
	for _, tt := range []struct {
		name, variant, markdown, rendered string
	}{
		{
			name:     "save follow rejects consult subtree",
			variant:  "save-follow",
			markdown: ":::cta\nvariant: save-follow\ntitle: Probe\n:::",
			rendered: ctaSemanticFixture("save-follow", false) + `<section data-module-part="cta-consult-layout"></section>`,
		},
		{
			name:     "consult rejects trial subtree",
			variant:  "consult",
			markdown: ":::cta\nvariant: consult\ntitle: Probe\n:::",
			rendered: ctaSemanticFixture("consult", false) + `<section data-module-part="cta-trial-primary"></section>`,
		},
		{
			name:     "trial without points rejects benefits subtree",
			variant:  "trial",
			markdown: ":::cta\nvariant: trial\ntitle: Probe\n:::",
			rendered: ctaSemanticFixture("trial", true),
		},
		{
			name:     "save follow rejects fourth action marker",
			variant:  "save-follow",
			markdown: ":::cta\nvariant: save-follow\ntitle: Probe\n:::",
			rendered: strings.Replace(ctaSemanticFixture("save-follow", false), `</section></section>`, `<span data-cta-action="quaternary"></span></section></section>`, 1),
		},
		{
			name:     "consult requires note marker when note is supplied",
			variant:  "consult",
			markdown: ":::cta\nvariant: consult\ntitle: Probe\nnote: Context\n:::",
			rendered: ctaSemanticFixture("consult", false),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			witness := e2eWitness{Module: "cta", Variant: tt.variant, Markdown: tt.markdown, Probe: "Probe"}
			if err := checkSemanticConformance(witness, tt.rendered); err == nil {
				t.Fatal("CTA response with a non-variant upstream marker must fail")
			}
		})
	}
}

func TestCanonicalCTAWitnessUsesCatalogDefaultForExactStructureChecks(t *testing.T) {
	c, err := layoutConformanceCatalog()
	if err != nil {
		t.Fatal(err)
	}
	spec, ok := c.Get("cta")
	if !ok {
		t.Fatal("cta catalog entry not found")
	}
	witnesses, err := witnessesForSpec(c, spec)
	if err != nil {
		t.Fatal(err)
	}
	var canonical e2eWitness
	for _, witness := range witnesses {
		if witness.Variant == "" {
			canonical = witness
			break
		}
	}
	if canonical.Module != "cta" {
		t.Fatalf("canonical CTA witness = %+v", canonical)
	}
	defaultVariant := ""
	for _, field := range spec.Fields.Optional {
		if field.Name == "variant" {
			defaultVariant = field.Default
			break
		}
	}
	if canonical.EffectiveVariant != defaultVariant {
		t.Fatalf("canonical CTA effective variant = %q, want catalog selector default %q", canonical.EffectiveVariant, defaultVariant)
	}
	valid := strings.Replace(ctaSemanticFixture("save-follow", false), `</section></section>`, `<p data-module-part="cta-note"></p></section></section>`, 1)
	valid = strings.Replace(valid, "Probe", canonical.Probe, 1)
	if err := checkConformanceHTML(canonical, valid); err != nil {
		t.Fatalf("canonical CTA default structure rejected: %v", err)
	}
	crossBranch := strings.Replace(valid, `<p data-module-part="cta-note"></p>`, `<section data-module-part="cta-consult-layout"></section><p data-module-part="cta-note"></p>`, 1)
	if err := checkConformanceHTML(canonical, crossBranch); err == nil {
		t.Fatal("canonical CTA witness accepted a cross-branch marker")
	}
}

func TestCompactPR1BoundaryAndThemeProbeConstructionIsCatalogBacked(t *testing.T) {
	c, err := layoutConformanceCatalog()
	if err != nil {
		t.Fatal(err)
	}
	assertValid := func(markdown string) {
		t.Helper()
		if report := c.Validate(markdown); len(report.Errors) != 0 {
			t.Fatalf("validate %q: %+v", markdown, report.Errors)
		}
	}
	assertInvalid := func(markdown string) {
		t.Helper()
		if report := c.Validate(markdown); len(report.Errors) == 0 {
			t.Fatalf("expected invalid: %q", markdown)
		}
	}

	hero, ok := c.Get("hero")
	if !ok {
		t.Fatal("hero not found")
	}
	symbols := []string(nil)
	for _, field := range hero.Fields.Optional {
		if field.Name == "symbol" {
			symbols = field.Enum
		}
	}
	if len(symbols) != 12 {
		t.Fatalf("symbol enum count = %d, want 12", len(symbols))
	}
	for _, symbol := range symbols {
		assertValid(":::hero\nvariant: masthead\ntitle: Probe\nsymbol: " + symbol + "\n:::\n")
	}
	assertInvalid(":::hero\nvariant: masthead\ntitle: Probe\nsymbol: ✨\n:::\n")
	assertInvalid(":::hero\nvariant: masthead\ntitle: Probe\nkicker: ignored\n:::\n")

	assertValid(":::section-title\nvariant: numbered\nindex: 1\ntitle: Probe\n:::\n")
	assertValid(":::section-title\nvariant: numbered\nindex: 1234\ntitle: Probe\n:::\n")
	assertInvalid(":::section-title\nvariant: numbered\nindex: 12345\ntitle: Probe\n:::\n")
	assertValid(":::cta\nvariant: trial\ntitle: Probe\npoints: one | two | three\n:::\n")
	assertInvalid(":::cta\nvariant: trial\ntitle: Probe\npoints: one | two | three | four\n:::\n")
	assertInvalid(":::cta\nvariant: consult\ntitle: Probe\npoints: ignored\n:::\n")
	assertValid(":::faq\nQ: Question?\nA: Answer.\nQ: Another?\nA: Another answer.\n:::\n")
	assertInvalid(":::faq\nq: Question?\na: Answer.\n:::\n")
	for _, tone := range []string{"fit", "avoid", "risk", "require", "note"} {
		assertValid(":::notice\n" + tone + " | Label | Body\n:::\n")
	}
	for _, markdown := range []string{
		":::summary\nhighlight: One line\n:::\n",
		":::summary\nvariant: three\nitems: one | two | three\n:::\n",
		":::summary\nvariant: decision\nrecommendation: Decide\n:::\n",
		":::summary\nvariant: save\nitems: one | two | three\n:::\n",
	} {
		assertValid(markdown)
	}

	themes := converter.NewThemeManager()
	if err := themes.LoadThemes(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"default", "apple", "cyber", "bytedance", "sports", "chinese"} {
		if _, err := themes.ResolveThemeForMode(converter.ModeAPI, name); err != nil {
			t.Fatalf("representative API theme %q is not selectable: %v", name, err)
		}
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
			Body:       io.NopCloser(strings.NewReader(`{"code":0,"data":{"html":"<section data-mpa-action-id=\"demo\">Visible probe</section>","theme":"default","fontSize":"medium","backgroundType":"none","wordCount":2,"estimatedReadTime":1}}`)),
			Request:    req,
		}, nil
	})}
	witness := e2eWitness{Module: "demo", Markdown: markdown, Probe: "Visible probe"}
	if _, _, err := runConformanceRequest(client, "http://layout.invalid", "secret", witness); err != nil {
		t.Fatal(err)
	}
}

func TestE2ESettingsUseProductionTargetAndConfigCredential(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MD2WECHAT_API_KEY", "environment-key")
	t.Setenv("MD2WECHAT_BASE_URL", "")
	t.Setenv("MD2WECHAT_API_BUILD_ID", "expected-build")
	t.Setenv("MD2WECHAT_CLI_COMMIT", "cli-commit")
	t.Setenv("MD2WECHAT_UPSTREAM_FIELD_CONTRACT_SHA", "")
	t.Setenv("MD2WECHAT_UPSTREAM_FIELD_CONTRACT_RESULT", "")
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
	if settings.BaseURL != converter.DefaultAPIConvertURL {
		t.Fatalf("BaseURL = %q", settings.BaseURL)
	}
	if settings.APIKey != "environment-key" || settings.CLICommit != "cli-commit" || settings.ExpectedBuildID != "expected-build" {
		t.Fatalf("settings did not honor safe config/env precedence")
	}
}

func TestE2ESettingsAllowExplicitTargetOverride(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MD2WECHAT_API_KEY", "environment-key")
	t.Setenv("MD2WECHAT_BASE_URL", "http://localhost:3000/")
	t.Setenv("MD2WECHAT_CLI_COMMIT", "cli-commit")
	t.Setenv("MD2WECHAT_UPSTREAM_FIELD_CONTRACT_SHA", "")
	t.Setenv("MD2WECHAT_UPSTREAM_FIELD_CONTRACT_RESULT", "")
	settings, err := loadE2ESettings()
	if err != nil {
		t.Fatal(err)
	}
	if settings.BaseURL != "http://localhost:3000/api/convert" {
		t.Fatalf("BaseURL = %q", settings.BaseURL)
	}
}

func TestE2ESettingsValidateOptionalUpstreamFieldContractEvidence(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MD2WECHAT_API_KEY", "environment-key")
	t.Setenv("MD2WECHAT_CLI_COMMIT", "cli-commit")
	t.Setenv("MD2WECHAT_UPSTREAM_FIELD_CONTRACT_SHA", "abc123")
	t.Setenv("MD2WECHAT_UPSTREAM_FIELD_CONTRACT_RESULT", "passed")
	if settings, err := loadE2ESettings(); err != nil || settings.FieldContractSHA != "abc123" || settings.FieldContractResult != "passed" {
		t.Fatalf("loadE2ESettings() = %+v, %v", settings, err)
	}
	t.Setenv("MD2WECHAT_UPSTREAM_FIELD_CONTRACT_RESULT", "failed")
	if _, err := loadE2ESettings(); err == nil || !strings.Contains(err.Error(), "fixture failed") {
		t.Fatalf("failed field contract error = %v", err)
	}
}

func TestE2ESettingsRequirePassedUpstreamFieldContractEvidenceInReleaseMode(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MD2WECHAT_API_KEY", "environment-key")
	t.Setenv("MD2WECHAT_CLI_COMMIT", "cli-commit")
	t.Setenv("MD2WECHAT_LAYOUT_CONFORMANCE_MODE", "release")
	t.Setenv("MD2WECHAT_UPSTREAM_FIELD_CONTRACT_SHA", "")
	t.Setenv("MD2WECHAT_UPSTREAM_FIELD_CONTRACT_RESULT", "")
	if _, err := loadE2ESettings(); err == nil || !strings.Contains(err.Error(), "release conformance requires") {
		t.Fatalf("missing release evidence error = %v", err)
	}
	t.Setenv("MD2WECHAT_UPSTREAM_FIELD_CONTRACT_SHA", "edcde64")
	t.Setenv("MD2WECHAT_UPSTREAM_FIELD_CONTRACT_RESULT", "passed")
	if _, err := loadE2ESettings(); err == nil || !strings.Contains(err.Error(), "must use pinned SHA") {
		t.Fatalf("stale release SHA error = %v", err)
	}
	t.Setenv("MD2WECHAT_UPSTREAM_FIELD_CONTRACT_SHA", pinnedUpstreamFieldContractSHA)
	if _, err := loadE2ESettings(); err != nil {
		t.Fatalf("pinned release evidence rejected: %v", err)
	}
}

func TestCompactPR1CompositionContainsAllRequiredStructures(t *testing.T) {
	c, err := layoutConformanceCatalog()
	if err != nil {
		t.Fatal(err)
	}
	markdown, _, err := compactPR1Composition(c)
	if err != nil {
		t.Fatal(err)
	}
	if report := c.Validate(markdown); len(report.Errors) != 0 {
		t.Fatalf("compact composition does not validate: %+v", report.Errors)
	}
	for _, marker := range []string{
		":::hero", ":::section-title", ":::epilogue", ":::summary", ":::cta", ":::closing",
		"variant: marker", "variant: divider", "variant: numbered", "variant: frame", "variant: focus", "variant: vertical",
		"highlight: Compact one-line summary",
		"variant: three", "title: Compact three summary",
		"variant: decision", "title: Compact decision summary",
		"variant: save", "title: Compact save summary",
	} {
		if !strings.Contains(markdown, marker) {
			t.Errorf("compact composition missing %q", marker)
		}
	}
	if got := strings.Count(markdown, ":::summary\n"); got != 4 {
		t.Fatalf("compact composition summary block count = %d, want 4 distinct branches", got)
	}
}

func TestCompactPR1CompositionWitnessesCoverFAQNoticeAndSummaryBranches(t *testing.T) {
	c, err := layoutConformanceCatalog()
	if err != nil {
		t.Fatal(err)
	}
	notice, ok := c.Get("notice")
	if !ok || notice.Rows == nil || notice.Rows.Delimiter == "" {
		t.Fatal("notice catalog row delimiter not found")
	}
	markdown, witnesses, err := compactPR1Composition(c)
	if err != nil {
		t.Fatal(err)
	}
	validHTML := `<section data-mpa-action-id="faq">Compact FAQ question? Compact FAQ answer.</section>` +
		`<section data-mpa-action-id="notice">` +
		`<p data-notice-tone="fit">Compact fit notice</p>` +
		`<p data-notice-tone="avoid">Compact avoid notice</p>` +
		`<p data-notice-tone="risk">Compact risk notice</p>` +
		`<p data-notice-tone="require">Compact require notice</p>` +
		`<p data-notice-tone="note">Compact note notice</p>` +
		`</section>` +
		`<section data-mpa-action-id="summary" data-summary-variant="one-line">Compact one-line summary</section>` +
		`<section data-mpa-action-id="summary" data-summary-variant="three">Compact three summary</section>` +
		`<section data-mpa-action-id="summary" data-summary-variant="decision">Compact decision summary</section>` +
		`<section data-mpa-action-id="summary" data-summary-variant="save">Compact save summary</section>`
	want := map[string]bool{
		"faq/Compact FAQ question?":        true,
		"faq/Compact FAQ answer.":          true,
		"notice/Compact fit notice":        true,
		"notice/Compact avoid notice":      true,
		"notice/Compact risk notice":       true,
		"notice/Compact require notice":    true,
		"notice/Compact note notice":       true,
		"summary/Compact one-line summary": true,
		"summary/Compact three summary":    true,
		"summary/Compact decision summary": true,
		"summary/Compact save summary":     true,
	}
	seen := map[string]bool{}
	for _, witness := range witnesses {
		if witness.Module != "faq" && witness.Module != "notice" && witness.Module != "summary" {
			continue
		}
		key := witness.Module + "/" + witness.Probe
		if !want[key] {
			t.Errorf("unexpected compact semantic witness %q", key)
			continue
		}
		if seen[key] {
			t.Errorf("duplicate compact semantic witness %q", key)
			continue
		}
		seen[key] = true
		if witness.Markdown == "" || !strings.Contains(markdown, strings.TrimSpace(witness.Markdown)) {
			t.Errorf("%s witness is not bound to its submitted compact block", key)
		}
		if witness.Module == "faq" && (strings.HasPrefix(witness.Probe, "Q:") || strings.HasPrefix(witness.Probe, "A:")) {
			t.Errorf("FAQ witness probe must be visible payload, got %q", witness.Probe)
		}
		if witness.Module == "notice" && witness.RowDelimiter != notice.Rows.Delimiter {
			t.Errorf("notice witness delimiter = %q, want catalog delimiter %q", witness.RowDelimiter, notice.Rows.Delimiter)
		}
		if witness.Module == "summary" {
			attribute, _, ok := expectedVariantBranch(witness)
			if !ok || attribute != "data-summary-variant" {
				t.Errorf("%s witness lacks exact summary branch evidence", key)
			}
		}
		if err := checkConformanceHTML(witness, validHTML); err != nil {
			t.Errorf("%s witness rejected valid compact DOM: %v", key, err)
		}
	}
	missing := make([]string, 0)
	for key := range want {
		if !seen[key] {
			missing = append(missing, key)
		}
	}
	slices.Sort(missing)
	if len(missing) != 0 {
		t.Fatalf("compact semantic witnesses missing %v", missing)
	}
	missingNoteHTML := strings.Replace(validHTML, `<p data-notice-tone="note">Compact note notice</p>`, `Compact note notice`, 1)
	for _, witness := range witnesses {
		if witness.Module != "notice" {
			continue
		}
		if err := checkConformanceHTML(witness, missingNoteHTML); err == nil || !strings.Contains(err.Error(), `data-notice-tone="note"`) {
			t.Fatalf("notice without note tone marker error = %v", err)
		}
		break
	}
}

func TestValidateRemoteBuildIdentity(t *testing.T) {
	tests := []struct {
		name, got, expected, first string
		wantErr                    bool
	}{
		{name: "expected match", got: "build-a", expected: "build-a"},
		{name: "expected missing", expected: "build-a", wantErr: true},
		{name: "expected mismatch", got: "build-b", expected: "build-a", wantErr: true},
		{name: "consistent observed", got: "build-a", first: "build-a"},
		{name: "conflicting observed", got: "build-b", first: "build-a", wantErr: true},
		{name: "missing conflicts with observed", first: "build-a", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRemoteBuildIdentity(tt.got, tt.expected, tt.first)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateRemoteBuildIdentity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConformanceResultPreservesFailureCategoryOrder(t *testing.T) {
	networkErr := errors.New("network failure: connection refused")
	if err := validateConformanceResult("", false, networkErr, "expected-build", "prior-build"); !errors.Is(err, networkErr) {
		t.Fatalf("no-response error = %v, want original network failure", err)
	}

	apiErr := errors.New("API drift: decode envelope")
	if err := validateConformanceResult("wrong-build", true, apiErr, "expected-build", ""); err == nil || errors.Is(err, apiErr) || !strings.Contains(err.Error(), "build id") {
		t.Fatalf("response-bearing mismatch error = %v, want build identity failure", err)
	}
	if err := validateConformanceResult("expected-build", true, apiErr, "expected-build", ""); !errors.Is(err, apiErr) {
		t.Fatalf("response-bearing matched error = %v, want original API failure", err)
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
	if _, err := postConvert(client, converter.DefaultAPIConvertURL, "secret", "body"); err == nil || !strings.Contains(err.Error(), "network failure") {
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
		{name: "dialogue", format: layoutcatalog.BodyFormatDialogue, markdown: ":::demo\n甲：Dialogue https://example.com\n:::\n", want: "Dialogue https://example.com"},
	}
	testedFormats := make(map[string]bool, len(tests))
	for _, tt := range tests {
		testedFormats[tt.format] = true
		t.Run(tt.name, func(t *testing.T) {
			got, err := deterministicWitnessProbe(&layoutcatalog.LayoutSpec{BodyFormat: tt.format}, tt.markdown)
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

func TestCatalogControlFieldsAreNotVisibleWitnessProbes(t *testing.T) {
	c, err := layoutConformanceCatalog()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		module string
		want   string
	}{
		{module: "faq", want: "这些模块只能在某一个主题里用吗？"},
		{module: "notice", want: "适合"},
	}
	for _, tt := range tests {
		t.Run(tt.module, func(t *testing.T) {
			spec, ok := c.Get(tt.module)
			if !ok {
				t.Fatalf("catalog module %q not found", tt.module)
			}
			witnesses, err := witnessesForSpec(c, spec)
			if err != nil {
				t.Fatal(err)
			}
			if got := witnesses[0].Probe; got != tt.want {
				t.Fatalf("probe = %q, want visible payload %q", got, tt.want)
			}
		})
	}
}

func TestDeterministicRowsProbeSkipsSchemaEnumControls(t *testing.T) {
	spec := &layoutcatalog.LayoutSpec{
		BodyFormat: layoutcatalog.BodyFormatRows,
		Rows: &layoutcatalog.RowsSpec{
			Delimiter: ";",
			Schema: []layoutcatalog.FieldSpec{
				{Name: "tone", Enum: []string{"fit", "risk"}},
				{Name: "label"},
				{Name: "body"},
			},
		},
	}
	got, err := deterministicWitnessProbe(spec, ":::demo\nfit ; Visible label ; Body\n:::\n")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Visible label" {
		t.Fatalf("probe = %q, want first non-control row cell", got)
	}
}

func TestNoticeConformanceRequiresSemanticToneAndVisiblePayload(t *testing.T) {
	witness := e2eWitness{
		Module:       "notice",
		Markdown:     ":::notice\nfit | 适合 | 正文\n:::\n",
		Probe:        "适合",
		RowDelimiter: "|",
	}
	if err := checkConformanceHTML(witness, `<section data-mpa-action-id="notice">适合</section>`); err == nil {
		t.Fatal("notice without semantic tone evidence unexpectedly conformed")
	}
	if err := checkConformanceHTML(witness, `<section data-mpa-action-id="notice"><p data-notice-tone="fit">适合</p></section>`); err != nil {
		t.Fatalf("notice with visible payload and semantic tone should conform: %v", err)
	}
}

func TestNoticeConformanceUsesCatalogDeclaredRowDelimiter(t *testing.T) {
	c, err := layoutConformanceCatalog()
	if err != nil {
		t.Fatal(err)
	}
	spec, ok := c.Get("notice")
	if !ok || spec.Rows == nil {
		t.Fatal("notice row catalog entry not found")
	}
	spec.Rows.Delimiter = ";"
	spec.Example = ":::notice\nfit ; Visible label ; Visible body\n:::\n"

	witnesses, err := witnessesForSpec(c, spec)
	if err != nil {
		t.Fatal(err)
	}
	if len(witnesses) != 1 {
		t.Fatalf("notice witness count = %d, want 1", len(witnesses))
	}
	if err := checkConformanceHTML(witnesses[0], `<section data-mpa-action-id="notice"><p data-notice-tone="fit">Visible label</p></section>`); err != nil {
		t.Fatalf("catalog-delimited notice should conform: %v", err)
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

func TestWitnessCollectionRendersAndValidatesTheExactSubmittedBlock(t *testing.T) {
	c, err := layoutConformanceCatalog()
	if err != nil {
		t.Fatal(err)
	}
	spec, ok := c.Get("hero")
	if !ok {
		t.Fatal("hero not found")
	}
	witnesses, err := witnessesForSpec(c, spec)
	if err != nil {
		t.Fatal(err)
	}
	if len(witnesses) == 0 {
		t.Fatal("hero has no witness")
	}
	rendered, err := renderWitnessFromExample(c, spec, spec.Example)
	if err != nil {
		t.Fatal(err)
	}
	if witnesses[0].Markdown != rendered {
		t.Fatalf("submitted markdown differs from catalog-rendered witness\ngot:  %q\nwant: %q", witnesses[0].Markdown, rendered)
	}
	if report := c.Validate(witnesses[0].Markdown); len(report.Errors) != 0 {
		t.Fatalf("submitted witness did not validate: %+v", report.Errors)
	}
	if !strings.HasPrefix(witnesses[0].Markdown, ":::hero") {
		t.Fatalf("submitted markdown = %q", witnesses[0].Markdown)
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
			want := 1
			for _, variant := range spec.Variants {
				if strings.TrimSpace(variant.Example) != "" && (variant.Name != "default" || strings.TrimSpace(variant.Example) != strings.TrimSpace(spec.Example)) {
					want++
				}
			}
			if len(witnesses) != want {
				t.Fatalf("witness count = %d, want %d", len(witnesses), want)
			}
			for _, witness := range witnesses {
				if strings.TrimSpace(witness.Probe) == "" {
					t.Fatalf("%s/%s has an empty probe", witness.Module, witness.Variant)
				}
			}
		})
	}
}

type e2eWitnessGroup struct {
	Lifecycle string
	Witnesses []e2eWitness
}

func collectAllE2EWitnesses(c *layoutcatalog.Catalog) ([]e2eWitnessGroup, error) {
	groups := make([]e2eWitnessGroup, 0, 2)
	for _, lifecycle := range []string{layoutcatalog.LifecycleRecommended, layoutcatalog.LifecycleCompatibility} {
		group := e2eWitnessGroup{Lifecycle: lifecycle}
		for _, spec := range c.ListFiltered(layoutcatalog.ListFilter{Lifecycle: lifecycle}) {
			witnesses, err := witnessesForSpec(c, spec)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", spec.Name, err)
			}
			group.Witnesses = append(group.Witnesses, witnesses...)
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func TestCollectAllE2EWitnessesHasStable84WitnessContract(t *testing.T) {
	c, err := layoutConformanceCatalog()
	if err != nil {
		t.Fatal(err)
	}
	groups, err := collectAllE2EWitnesses(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 || groups[0].Lifecycle != layoutcatalog.LifecycleRecommended || groups[1].Lifecycle != layoutcatalog.LifecycleCompatibility {
		t.Fatalf("lifecycle groups = %+v", groups)
	}
	if got := len(groups[0].Witnesses) + len(groups[1].Witnesses); got != 84 {
		t.Fatalf("witness count = %d, want 84", got)
	}
	if got := len(groups[0].Witnesses); got != 81 {
		t.Fatalf("recommended witness count = %d, want 81 (56 canonical + 25 non-default branches)", got)
	}
	if got := len(c.ListFiltered(layoutcatalog.ListFilter{Lifecycle: layoutcatalog.LifecycleRecommended})); got != 56 {
		t.Fatalf("recommended canonical witness count = %d, want 56", got)
	}
	if got := len(groups[0].Witnesses) - len(c.ListFiltered(layoutcatalog.ListFilter{Lifecycle: layoutcatalog.LifecycleRecommended})); got != 25 {
		t.Fatalf("recommended non-default branch witness count = %d, want 25", got)
	}
	if got := len(groups[1].Witnesses); got != 3 {
		t.Fatalf("compatibility witness count = %d, want 3", got)
	}
}

func TestLayoutConformanceHTTPClientHasTimeout(t *testing.T) {
	client := layoutConformanceHTTPClient()
	if client.Timeout <= 0 {
		t.Fatalf("conformance HTTP timeout = %v, want positive", client.Timeout)
	}
}

func TestE2ELayoutConformance(t *testing.T) {
	settings := e2eGate(t)
	c, err := layoutConformanceCatalog()
	if err != nil {
		t.Fatal(err)
	}
	groups, err := collectAllE2EWitnesses(c)
	if err != nil {
		t.Fatal(err)
	}
	client := layoutConformanceHTTPClient()
	var remoteIdentity string
	identityInitialized := false
	for _, group := range groups {
		group := group
		t.Run(group.Lifecycle, func(t *testing.T) {
			for _, witness := range group.Witnesses {
				witness := witness
				name := witness.Module
				if witness.Variant != "" {
					name += "/" + witness.Variant
				}
				t.Run(name, func(t *testing.T) {
					identity, responseReceived, requestErr := runConformanceRequest(client, settings.BaseURL, settings.APIKey, witness)
					first := ""
					if identityInitialized {
						first = remoteIdentity
					}
					if err := validateConformanceResult(identity, responseReceived, requestErr, settings.ExpectedBuildID, first); err != nil {
						t.Fatal(err)
					}
					if identityInitialized && identity != remoteIdentity {
						t.Fatalf("API drift: conflicting response build ids %q and %q", remoteIdentity, identity)
					}
					if !identityInitialized {
						remoteIdentity = identity
						identityInitialized = true
					}
				})
			}
		})
	}
	t.Logf("conformance_target_normalized=%s cli_commit=%s", settings.BaseURL, settings.CLICommit)
	if settings.FieldContractSHA != "" {
		t.Logf("upstream_field_contract_sha=%s result=%s", settings.FieldContractSHA, settings.FieldContractResult)
	}
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
	resp, err := postConvert(layoutConformanceHTTPClient(), settings.BaseURL, settings.APIKey, string(data))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeE2EResponse(resp); err != nil {
		t.Fatal(err)
	}
}

func TestE2ECompactLayoutBoundaryAndThemeProbes(t *testing.T) {
	settings := e2eGate(t)
	c, err := layoutConformanceCatalog()
	if err != nil {
		t.Fatal(err)
	}
	markdown, witnesses, err := compactPR1Composition(c)
	if err != nil {
		t.Fatal(err)
	}
	if report := c.Validate(markdown); len(report.Errors) != 0 {
		t.Fatalf("compact probe does not validate: %+v", report.Errors)
	}
	for _, theme := range []string{"default", "apple", "cyber", "bytedance", "sports", "chinese"} {
		t.Run(theme, func(t *testing.T) {
			resp, err := postConvertRequest(layoutConformanceHTTPClient(), settings.BaseURL, settings.APIKey, converter.APIRequest{Markdown: markdown, Theme: theme, FontSize: "medium", BackgroundType: "none"})
			if err != nil {
				t.Fatal(err)
			}
			data, err := converter.DecodeAPIResponse(resp)
			if err != nil {
				t.Fatal(err)
			}
			if data.Theme != theme || data.FontSize != "medium" || data.BackgroundType != "none" {
				t.Fatalf("parameter echo = %+v", data)
			}
			for _, witness := range witnesses {
				if err := checkConformanceHTML(witness, data.HTML); err != nil {
					t.Fatal(err)
				}
			}
			if !strings.Contains(data.HTML, "Body evidence remains readable.") {
				t.Fatal("compact composition response omitted plain body content")
			}
			if strings.Contains(data.HTML, ":::") {
				t.Fatal("compact composition response retained a raw layout fence")
			}
		})
	}
}

func TestE2EValidatorVsAPIConsistency(t *testing.T) {
	settings := e2eGate(t)
	bad := ":::hero\neyebrow: only\n:::\n"
	c, err := layoutConformanceCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if r := c.Validate(bad); len(r.Errors) == 0 {
		t.Fatal("validator should flag missing title")
	}
	resp, err := postConvert(layoutConformanceHTTPClient(), settings.BaseURL, settings.APIKey, bad)
	if err != nil {
		t.Fatal(err)
	}
	html, err := decodeE2EResponse(resp)
	if err != nil {
		t.Fatalf("transport/API response failed for directive-level invalidity: %v", err)
	}
	if err := checkRejectedDirectiveHTML("hero", bad, html); err != nil {
		t.Fatal(err)
	}
}

func TestRejectedDirectiveHTMLAllowsSuccessfulArticleWithoutModuleReadiness(t *testing.T) {
	bad := ":::hero\neyebrow: only\n:::\n"
	if err := checkRejectedDirectiveHTML("hero", bad, `<p>fallback article</p>`); err != nil {
		t.Fatal(err)
	}
	for _, html := range []string{
		`<section data-mpa-action-id="hero">fallback article</section>`,
		`<p>:::hero</p>`,
	} {
		if err := checkRejectedDirectiveHTML("hero", bad, html); err == nil {
			t.Fatalf("rejected directive unexpectedly became ready for HTML %q", html)
		}
	}
}

var witnessImageRE = regexp.MustCompile(`!\[([^]]*)\]\((?:https?://[^)\s]+|<https?://[^>]+>)\)`)

func decodeE2EResponse(resp *http.Response) (string, error) {
	if resp == nil {
		return "", fmt.Errorf("nil response")
	}
	status := resp.StatusCode
	data, err := converter.DecodeAPIResponse(resp)
	if err != nil {
		if status == http.StatusUnauthorized || status == http.StatusForbidden || strings.Contains(err.Error(), "API returned error code 401") || strings.Contains(err.Error(), "API returned error code 403") {
			return "", fmt.Errorf("authentication failure: %w", err)
		}
		if strings.Contains(err.Error(), "API returned error code") {
			return "", fmt.Errorf("API drift: api code: %w", err)
		}
		return "", fmt.Errorf("API drift: %w", err)
	}
	return data.HTML, nil
}

// checkRejectedDirectiveHTML keeps local catalog rejection aligned with the
// remote article fallback contract: an invalid known directive is never a
// ready module, even when the overall article response succeeds.
func checkRejectedDirectiveHTML(module, rawDirective, html string) error {
	if strings.Contains(html, ":::"+module) || strings.Contains(html, rawDirective) {
		return fmt.Errorf("%s response retained rejected raw fence", module)
	}
	doc, err := htmlpkgParse(html)
	if err != nil {
		return fmt.Errorf("%s fallback response is not parseable HTML: %w", module, err)
	}
	if markers := findDOMAttributeValueNodes(doc, "data-mpa-action-id", module); len(markers) != 0 {
		return fmt.Errorf("%s response marked a locally rejected directive ready", module)
	}
	return nil
}

func checkConformanceHTML(witness e2eWitness, html string) error {
	if strings.Contains(html, ":::"+witness.Module) {
		return fmt.Errorf("%s response retained raw fence", witness.Module)
	}
	doc, err := htmlpkgParse(html)
	if err != nil {
		return fmt.Errorf("%s response is not parseable HTML: %w", witness.Module, err)
	}
	markers := findDOMAttributeValueNodes(doc, "data-mpa-action-id", witness.Module)
	if len(markers) == 0 {
		return fmt.Errorf("API drift: %s response missing module marker data-mpa-action-id=%q", witness.Module, witness.Module)
	}
	var lastErr error
	for _, marker := range markers {
		if err := checkConformanceSubtree(witness, marker); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return fmt.Errorf("%s response has no conforming module subtree: %w", witness.Module, lastErr)
}

func checkConformanceSubtree(witness e2eWitness, marker *html.Node) error {
	probe := strings.TrimSpace(witness.Probe)
	if probe == "" {
		return fmt.Errorf("%s witness has no visible probe", witness.Module)
	}
	if witness.ProbeInImageAlt && !hasImageAltProbe(marker, probe) {
		return fmt.Errorf("%s response missing image alt probe %q", witness.Module, probe)
	}
	if !witness.ProbeInImageAlt && !strings.Contains(strings.TrimSpace(visibleDOMText(marker)), probe) {
		return fmt.Errorf("%s response missing visible probe %q", witness.Module, probe)
	}
	if attribute, value, ok := expectedVariantBranch(witness); ok && !hasDOMAttributeValue(marker, attribute, value) {
		return fmt.Errorf("%s/%s response missing %s=%q", witness.Module, witness.Variant, attribute, value)
	}
	return checkSemanticConformanceNode(witness, marker)
}

var variantBranchAttributes = map[string]string{
	"hero":          "data-hero-variant",
	"quote":         "data-quote-variant",
	"summary":       "data-summary-variant",
	"cta":           "data-cta-variant",
	"infographic":   "data-infographic-type",
	"section-title": "data-section-title-variant",
}

func expectedVariantBranch(witness e2eWitness) (string, string, bool) {
	attribute, ok := variantBranchAttributes[witness.Module]
	variant := witness.Variant
	if witness.Module == "cta" && witness.EffectiveVariant != "" {
		variant = witness.EffectiveVariant
	}
	if !ok || variant == "" {
		return "", "", false
	}
	value := variant
	if witness.Module == "infographic" && value == "mini-case" {
		value = "micro-case"
	}
	return attribute, value, true
}

func htmlpkgParse(rendered string) (*html.Node, error) {
	return html.Parse(strings.NewReader(rendered))
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

func findDOMAttributeValueNodes(node *html.Node, name, value string) []*html.Node {
	var matches []*html.Node
	if node.Type == html.ElementNode {
		for _, attr := range node.Attr {
			if attr.Key == name && attr.Val == value {
				matches = append(matches, node)
				break
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		matches = append(matches, findDOMAttributeValueNodes(child, name, value)...)
	}
	return matches
}

func countDOMAttributes(node *html.Node, name string) int {
	count := 0
	if node.Type == html.ElementNode {
		for _, attr := range node.Attr {
			if attr.Key == name {
				count++
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		count += countDOMAttributes(child, name)
	}
	return count
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
	doc, err := html.Parse(strings.NewReader(rendered))
	if err != nil {
		return fmt.Errorf("%s response is not parseable HTML: %w", witness.Module, err)
	}
	return checkSemanticConformanceNode(witness, doc)
}

func checkSemanticConformanceNode(witness e2eWitness, node *html.Node) error {
	if witness.Module == "hero" && witness.Variant == "masthead" && !hasDOMAttributeValue(node, "data-module-part", "hero-masthead") {
		return fmt.Errorf("hero/masthead response missing data-module-part=%q", "hero-masthead")
	}
	if witness.Module == "notice" {
		if err := checkNoticeToneConformanceNode(witness, node); err != nil {
			return err
		}
	}
	if witness.Module == "cta" {
		if err := checkCTAConformanceNode(witness, node); err != nil {
			return err
		}
	}
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
	if count := countDOMElements(node, imageElement); count < minimumImages {
		return fmt.Errorf("%s response has %d %s element(s), want at least %d", witness.Module, count, imageElement, minimumImages)
	}
	return nil
}

func checkNoticeToneConformanceNode(witness e2eWitness, node *html.Node) error {
	if witness.RowDelimiter == "" {
		return fmt.Errorf("notice witness has no catalog row delimiter")
	}
	body, err := firstWitnessBody(witness.Markdown)
	if err != nil {
		return fmt.Errorf("notice witness body: %w", err)
	}
	seen := map[string]bool{}
	for _, line := range body {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" {
			continue
		}
		tone, _, ok := strings.Cut(line, witness.RowDelimiter)
		tone = strings.TrimSpace(tone)
		if !ok || tone == "" || seen[tone] {
			continue
		}
		seen[tone] = true
		if !hasDOMAttributeValue(node, "data-notice-tone", tone) {
			return fmt.Errorf("notice response missing data-notice-tone=%q", tone)
		}
	}
	if len(seen) == 0 {
		return fmt.Errorf("notice witness has no semantic tone controls")
	}
	return nil
}

func checkCTAConformanceNode(witness e2eWitness, node *html.Node) error {
	type actionGroup struct {
		part    string
		actions []string
	}
	type branchContract struct {
		required  []actionGroup
		forbidden []string
	}
	allBranchParts := []string{
		"cta-save-confirmation", "cta-save-actions",
		"cta-consult-layout", "cta-consult-trust", "cta-consult-primary", "cta-consult-aux",
		"cta-trial-benefits", "cta-trial-primary", "cta-trial-secondary",
	}
	variant := witness.EffectiveVariant
	if variant == "" {
		variant = witness.Variant
	}
	contract := branchContract{}
	switch variant {
	case "save-follow":
		contract.required = []actionGroup{{part: "cta-save-confirmation"}, {part: "cta-save-actions", actions: []string{"primary", "secondary", "tertiary"}}}
	case "consult":
		contract.required = []actionGroup{{part: "cta-consult-layout"}, {part: "cta-consult-trust"}, {part: "cta-consult-primary", actions: []string{"primary"}}, {part: "cta-consult-aux", actions: []string{"secondary", "tertiary"}}}
	case "trial":
		if strings.Contains(witness.Markdown, "points:") {
			contract.required = append(contract.required, actionGroup{part: "cta-trial-benefits"})
		} else {
			contract.forbidden = append(contract.forbidden, "cta-trial-benefits")
		}
		contract.required = append(contract.required, actionGroup{part: "cta-trial-primary", actions: []string{"primary"}}, actionGroup{part: "cta-trial-secondary", actions: []string{"secondary", "tertiary"}})
	default:
		return fmt.Errorf("cta witness has unknown variant %q", variant)
	}
	if strings.Contains(witness.Markdown, "\nnote:") {
		contract.required = append(contract.required, actionGroup{part: "cta-note"})
	} else {
		contract.forbidden = append(contract.forbidden, "cta-note")
	}
	requiredParts := make(map[string]bool, len(contract.required))
	for _, group := range contract.required {
		requiredParts[group.part] = true
		parts := findDOMAttributeValueNodes(node, "data-module-part", group.part)
		if len(parts) != 1 {
			return fmt.Errorf("cta/%s response has %d data-module-part=%q nodes, want exactly 1", variant, len(parts), group.part)
		}
		for _, action := range group.actions {
			if !hasDOMAttributeValue(parts[0], "data-cta-action", action) {
				return fmt.Errorf("cta/%s response missing %s action in data-module-part=%q", variant, action, group.part)
			}
		}
	}
	for _, part := range allBranchParts {
		if requiredParts[part] || slices.Contains(contract.forbidden, part) {
			continue
		}
		contract.forbidden = append(contract.forbidden, part)
	}
	for _, part := range contract.forbidden {
		if count := len(findDOMAttributeValueNodes(node, "data-module-part", part)); count != 0 {
			return fmt.Errorf("cta/%s response unexpectedly has data-module-part=%q", variant, part)
		}
	}
	if count := countDOMAttributes(node, "data-cta-action"); count != 3 {
		return fmt.Errorf("cta/%s response has %d data-cta-action markers, want exactly 3", variant, count)
	}
	return nil
}

func ctaSemanticFixture(variant string, withPoints bool) string {
	switch variant {
	case "save-follow":
		return `<section data-mpa-action-id="cta" data-cta-variant="save-follow">Probe<section data-module-part="cta-save-confirmation"></section><section data-module-part="cta-save-actions"><span data-cta-action="primary"></span><span data-cta-action="secondary"></span><span data-cta-action="tertiary"></span></section></section>`
	case "consult":
		return `<section data-mpa-action-id="cta" data-cta-variant="consult">Probe<section data-module-part="cta-consult-layout"><section data-module-part="cta-consult-trust"></section><section data-module-part="cta-consult-primary"><span data-cta-action="primary"></span></section><section data-module-part="cta-consult-aux"><span data-cta-action="secondary"></span><span data-cta-action="tertiary"></span></section></section></section>`
	case "trial":
		benefits := ""
		if withPoints {
			benefits = `<section data-module-part="cta-trial-benefits"></section>`
		}
		return `<section data-mpa-action-id="cta" data-cta-variant="trial">Probe` + benefits + `<section data-module-part="cta-trial-primary"><span data-cta-action="primary"></span></section><section data-module-part="cta-trial-secondary"><span data-cta-action="secondary"></span><span data-cta-action="tertiary"></span></section></section>`
	default:
		return ""
	}
}

func compactPR1Composition(c *layoutcatalog.Catalog) (string, []e2eWitness, error) {
	noticeSpec, ok := c.Get("notice")
	if !ok || noticeSpec.Rows == nil || noticeSpec.Rows.Delimiter == "" {
		return "", nil, fmt.Errorf("notice catalog row delimiter not found")
	}
	delimiter := noticeSpec.Rows.Delimiter
	block := func(lines ...string) string { return strings.Join(lines, "\n") }
	row := func(cells ...string) string { return strings.Join(cells, " "+delimiter+" ") }

	hero := block(":::hero", "variant: masthead", "title: Compact theme probe", "symbol: spark-solid", ":::")
	markerSection := block(":::section-title", "variant: marker", "symbol: diamond-outline", "title: Marker section", ":::")
	dividerSection := block(":::section-title", "variant: divider", "symbol: spark-outline", "title: Divider section", ":::")
	numberedSection := block(":::section-title", "variant: numbered", "index: 1234", "title: Numbered section", ":::")
	frameSection := block(":::section-title", "variant: frame", "title: Frame section", ":::")
	focusSection := block(":::section-title", "variant: focus", "symbol: double-circle", "title: Focus section", ":::")
	verticalSection := block(":::section-title", "variant: vertical", "symbol: diamond-solid", "title: Vertical section", ":::")
	faq := block(":::faq", "Q: Compact FAQ question?", "A: Compact FAQ answer.", "Q: Secondary compact FAQ question?", "A: Secondary compact FAQ answer.", ":::")
	notice := block(":::notice",
		row("fit", "Compact fit notice", "Fit body evidence."),
		row("avoid", "Compact avoid notice", "Avoid body evidence."),
		row("risk", "Compact risk notice", "Risk body evidence."),
		row("require", "Compact require notice", "Require body evidence."),
		row("note", "Compact note notice", "Note body evidence."),
		":::")
	epilogue := block(":::epilogue", "title: Epilogue transition", ":::")
	summaryOneLine := block(":::summary", "highlight: Compact one-line summary", ":::")
	summaryThree := block(":::summary", "variant: three", "title: Compact three summary", "items: Three alpha | Three beta | Three gamma", ":::")
	summaryDecision := block(":::summary", "variant: decision", "title: Compact decision summary", "recommendation: Choose the calibrated contract.", ":::")
	summarySave := block(":::summary", "variant: save", "title: Compact save summary", "items: Save alpha | Save beta | Save gamma", ":::")
	cta := block(":::cta", "variant: trial", "title: CTA probe", "points: one | two | three", ":::")
	closing := block(":::closing", "title: Quiet closing", ":::")

	markdown := strings.Join([]string{
		hero, markerSection, "Body evidence remains readable.", dividerSection, numberedSection, frameSection, focusSection, verticalSection,
		faq, notice, epilogue, summaryOneLine, summaryThree, summaryDecision, summarySave, cta, closing,
	}, "\n\n") + "\n"
	witnesses := []e2eWitness{
		{Module: "hero", Variant: "masthead", Probe: "Compact theme probe"},
		{Module: "section-title", Variant: "marker", Probe: "Marker section"},
		{Module: "section-title", Variant: "divider", Probe: "Divider section"},
		{Module: "section-title", Variant: "numbered", Probe: "Numbered section"},
		{Module: "section-title", Variant: "frame", Probe: "Frame section"},
		{Module: "section-title", Variant: "focus", Probe: "Focus section"},
		{Module: "section-title", Variant: "vertical", Probe: "Vertical section"},
		{Module: "faq", Markdown: faq, Probe: "Compact FAQ question?"},
		{Module: "faq", Markdown: faq, Probe: "Compact FAQ answer."},
		{Module: "notice", Markdown: notice, Probe: "Compact fit notice", RowDelimiter: delimiter},
		{Module: "notice", Markdown: notice, Probe: "Compact avoid notice", RowDelimiter: delimiter},
		{Module: "notice", Markdown: notice, Probe: "Compact risk notice", RowDelimiter: delimiter},
		{Module: "notice", Markdown: notice, Probe: "Compact require notice", RowDelimiter: delimiter},
		{Module: "notice", Markdown: notice, Probe: "Compact note notice", RowDelimiter: delimiter},
		{Module: "epilogue", Probe: "Epilogue transition"},
		{Module: "summary", Variant: "one-line", Markdown: summaryOneLine, Probe: "Compact one-line summary"},
		{Module: "summary", Variant: "three", Markdown: summaryThree, Probe: "Compact three summary"},
		{Module: "summary", Variant: "decision", Markdown: summaryDecision, Probe: "Compact decision summary"},
		{Module: "summary", Variant: "save", Markdown: summarySave, Probe: "Compact save summary"},
		{Module: "cta", Variant: "trial", Probe: "CTA probe"},
		{Module: "closing", Probe: "Quiet closing"},
	}
	return markdown, witnesses, nil
}

func TestCompactPR1CompositionSeparatesMarkdownBlocks(t *testing.T) {
	c, err := layoutConformanceCatalog()
	if err != nil {
		t.Fatal(err)
	}
	markdown, _, err := compactPR1Composition(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		":::\n\n:::section-title",
		":::\n\nBody evidence remains readable.\n\n:::section-title",
	} {
		if !strings.Contains(markdown, want) {
			t.Errorf("compact composition missing Markdown block boundary %q", want)
		}
	}
	if strings.Contains(markdown, ":::\n:::section-title") {
		t.Fatal("compact composition joins layout fences without a blank-line boundary")
	}
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

func runConformanceRequest(client *http.Client, baseURL, apiKey string, witness e2eWitness) (string, bool, error) {
	resp, err := postConvert(client, baseURL, apiKey, witness.Markdown)
	if err != nil {
		return "", false, err
	}
	identity, _ := remoteBuildIdentity(resp.Header)
	html, err := decodeE2EResponse(resp)
	if err != nil {
		return identity, true, err
	}
	if err := checkConformanceHTML(witness, html); err != nil {
		return identity, true, fmt.Errorf("API drift: %w", err)
	}
	return identity, true, nil
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
	return postConvertRequest(client, baseURL, apiKey, converter.APIRequest{Markdown: markdown, Theme: "default"})
}

func postConvertRequest(client *http.Client, baseURL, apiKey string, request converter.APIRequest) (*http.Response, error) {
	resp, err := converter.PostAPIConvert(client, baseURL, apiKey, request)
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
		// Only structural branches with executable examples receive separate
		// remote requests; defaults remain covered by the canonical witness.
		if strings.TrimSpace(variant.Example) == "" || (variant.Name == "default" && strings.TrimSpace(variant.Example) == strings.TrimSpace(spec.Example)) {
			continue
		}
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
		rendered, err := renderWitnessFromExample(c, spec, item.markdown)
		if err != nil {
			return nil, fmt.Errorf("%s witness %q render: %w", spec.Name, item.variant, err)
		}
		if report := c.Validate(rendered); len(report.Errors) != 0 {
			return nil, fmt.Errorf("%s witness %q emitted invalid block: %+v", spec.Name, item.variant, report.Errors)
		}
		probe := strings.TrimSpace(item.assertion)
		probeInImageAlt := false
		if probe == "" {
			var err error
			probe, err = deterministicWitnessProbe(spec, item.markdown)
			if err != nil {
				return nil, fmt.Errorf("%s witness %q probe: %w", spec.Name, item.variant, err)
			}
			probeInImageAlt = spec.BodyFormat == layoutcatalog.BodyFormatMarkdownImages
		}
		if probe == "" {
			return nil, fmt.Errorf("%s witness %q has no deterministic probe or explicit assertion", spec.Name, item.variant)
		}
		witnesses = append(witnesses, e2eWitness{
			Module: spec.Name, Variant: item.variant, EffectiveVariant: canonicalSelectorDefault(spec, item.variant), Markdown: rendered,
			Probe: probe, ProbeInImageAlt: probeInImageAlt, RowDelimiter: rowsDelimiter(spec),
		})
	}
	return witnesses, nil
}

func rowsDelimiter(spec *layoutcatalog.LayoutSpec) string {
	if spec.Rows == nil {
		return ""
	}
	return spec.Rows.Delimiter
}

func canonicalSelectorDefault(spec *layoutcatalog.LayoutSpec, declaredVariant string) string {
	if declaredVariant != "" || spec.Fields == nil {
		return declaredVariant
	}
	for _, field := range spec.Fields.Optional {
		if field.Name == "variant" && field.Default != "" {
			return field.Default
		}
	}
	return ""
}

// renderWitnessFromExample turns the catalog-owned executable example into the
// exact block submitted to the remote renderer. The source example remains the
// only hand-authored input: the conformance runner never keeps a second copy of
// an opener or body for rendering, validation, and conversion.
func renderWitnessFromExample(c *layoutcatalog.Catalog, spec *layoutcatalog.LayoutSpec, example string) (string, error) {
	lines := strings.Split(strings.TrimSpace(example), "\n")
	if len(lines) < 2 || !strings.HasPrefix(lines[0], ":::"+spec.Name) {
		return "", fmt.Errorf("missing %s witness opener", spec.Name)
	}
	closeAt := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(strings.TrimRight(lines[i], "\r")) == ":::" {
			closeAt = i
			break
		}
	}
	if closeAt < 0 {
		return "", fmt.Errorf("missing %s witness closing fence", spec.Name)
	}
	input := layoutcatalog.RenderInput{Body: strings.Join(lines[1:closeAt], "\n")}
	suffix := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[0]), ":::"+spec.Name))
	switch {
	case suffix == "":
	case strings.HasPrefix(suffix, "[") && strings.HasSuffix(suffix, "]"):
		if spec.Opener == nil {
			input.Fields = map[string]any{"caption": suffix[1 : len(suffix)-1]}
		} else {
			input.Caption = suffix[1 : len(suffix)-1]
		}
	case strings.HasPrefix(suffix, "{") && strings.HasSuffix(suffix, "}"):
		params, err := witnessOpenerParams(suffix[1 : len(suffix)-1])
		if err != nil {
			return "", err
		}
		input.Params = params
	default:
		params, err := witnessOpenerParams(suffix)
		if err != nil {
			return "", err
		}
		input.Params = params
	}
	return c.RenderBlock(spec.Name, input)
}

func witnessOpenerParams(raw string) (map[string]string, error) {
	params := map[string]string{}
	for _, field := range strings.Fields(raw) {
		key, value, ok := strings.Cut(field, "=")
		if !ok || key == "" || value == "" {
			return nil, fmt.Errorf("unsupported raw witness opener parameter %q", field)
		}
		params[key] = value
	}
	return params, nil
}

func deterministicWitnessProbe(spec *layoutcatalog.LayoutSpec, markdown string) (string, error) {
	body, err := firstWitnessBody(markdown)
	if err != nil {
		return "", err
	}
	switch spec.BodyFormat {
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
			delimiter := "|"
			if spec.Rows != nil && spec.Rows.Delimiter != "" {
				delimiter = spec.Rows.Delimiter
			}
			cells := strings.Split(line, delimiter)
			for i, cell := range cells {
				if spec.Rows != nil && i < len(spec.Rows.Schema) && len(spec.Rows.Schema[i].Enum) != 0 {
					continue
				}
				if cell = strings.TrimSpace(cell); cell != "" {
					return cell, nil
				}
			}
		}
	case layoutcatalog.BodyFormatMarkdownImages:
		for _, line := range body {
			match := witnessImageRE.FindStringSubmatch(line)
			if len(match) == 2 && strings.TrimSpace(match[1]) != "" {
				return strings.TrimSpace(match[1]), nil
			}
		}
	case layoutcatalog.BodyFormatDialogue:
		for _, line := range body {
			line = strings.TrimSpace(strings.TrimRight(line, "\r"))
			if line == "" {
				continue
			}
			separatorIndex := strings.Index(line, ":")
			separatorWidth := len(":")
			if fullWidth := strings.Index(line, "："); separatorIndex < 0 || (fullWidth >= 0 && fullWidth < separatorIndex) {
				separatorIndex = fullWidth
				separatorWidth = len("：")
			}
			if separatorIndex >= 0 {
				return strings.TrimSpace(line[separatorIndex+separatorWidth:]), nil
			}
		}
	case layoutcatalog.BodyFormatSplit, layoutcatalog.BodyFormatLines:
		for _, line := range body {
			line = strings.TrimSpace(strings.TrimRight(line, "\r"))
			if line != "" && line != "---" {
				return line, nil
			}
		}
	default:
		return "", fmt.Errorf("unsupported body format %q", spec.BodyFormat)
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
