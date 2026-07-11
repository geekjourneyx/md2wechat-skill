package layoutcatalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegrationOpinionPieceValidates(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join("testdata", "integration", "opinion-piece.md"))
	if err != nil {
		t.Fatal(err)
	}
	r := c.Validate(string(data))
	if len(r.Errors) > 0 {
		t.Errorf("opinion-piece should validate clean, got errors: %+v", r.Errors)
	}
}

func TestIntegrationDataReportValidates(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join("testdata", "integration", "data-report.md"))
	if err != nil {
		t.Fatal(err)
	}
	r := c.Validate(string(data))
	if len(r.Errors) > 0 {
		t.Errorf("data-report should validate clean, got errors: %+v", r.Errors)
	}
}

func TestIntegrationMixedWithUnknownOnlyWarns(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join("testdata", "integration", "mixed-with-unknown.md"))
	if err != nil {
		t.Fatal(err)
	}
	r := c.Validate(string(data))
	if len(r.Errors) > 0 {
		t.Errorf("unknown module must NOT error: %+v", r.Errors)
	}
	if len(r.Warnings) == 0 {
		t.Errorf("expected warnings for unknown module")
	}
}

func TestIntegrationRenderThenValidateRoundtrip(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	out, err := c.Render("hero", map[string]any{
		"eyebrow": "深度观察", "title": "真问题",
	})
	if err != nil {
		t.Fatal(err)
	}
	r := c.Validate(out)
	if len(r.Errors) > 0 {
		t.Errorf("rendered output should validate clean: %+v", r.Errors)
	}
}

func TestIntegrationFreeLayoutCanonicalExamplesValidate(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	markdown := strings.Join([]string{
		splitGuideSnippet,
		flowGuideSnippet,
		matrixGuideSnippet,
		dialoguePairGuideSnippet,
	}, "\n")
	result := c.Validate(markdown)
	if len(result.Errors) != 0 || len(result.Warnings) != 0 {
		t.Fatalf("free-layout canonical examples should validate cleanly: errors=%+v warnings=%+v", result.Errors, result.Warnings)
	}
}
