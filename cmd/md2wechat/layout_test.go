package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/layoutcatalog"
)

func TestLayoutListJSONIncludesHero(t *testing.T) {
	oldJSON := jsonOutput
	t.Cleanup(func() {
		jsonOutput = oldJSON
		layoutcatalog.ResetDefaultCatalogForTests()
	})

	jsonOutput = true
	layoutcatalog.ResetDefaultCatalogForTests()

	stdout := captureStdout(t, func() {
		if err := layoutListCmd.RunE(layoutListCmd, nil); err != nil {
			t.Fatalf("layoutListCmd.RunE() error = %v", err)
		}
	})

	if !strings.Contains(string(stdout), `"hero"`) {
		t.Errorf("expected hero in list output, got:\n%s", stdout)
	}
	if !strings.Contains(string(stdout), `"body_format"`) {
		t.Errorf("expected body_format in list output, got:\n%s", stdout)
	}
}

func TestLayoutListDefaultsToRecommendedLifecycle(t *testing.T) {
	setupLayoutListLifecycleTest(t)

	stdout := captureStdout(t, func() {
		if err := layoutListCmd.RunE(layoutListCmd, nil); err != nil {
			t.Fatalf("layoutListCmd.RunE() error = %v", err)
		}
	})

	output := string(stdout)
	if strings.Contains(output, `"compatibility-only"`) {
		t.Fatalf("default list unexpectedly included compatibility module:\n%s", stdout)
	}
	if !strings.Contains(output, `"lifecycle": "recommended"`) {
		t.Fatalf("default list did not include recommended lifecycle summaries:\n%s", stdout)
	}
}

func TestLayoutListSelectsCompatibilityLifecycle(t *testing.T) {
	setupLayoutListLifecycleTest(t)
	layoutListFilters.lifecycle = layoutcatalog.LifecycleCompatibility

	stdout := captureStdout(t, func() {
		if err := layoutListCmd.RunE(layoutListCmd, nil); err != nil {
			t.Fatalf("layoutListCmd.RunE() error = %v", err)
		}
	})

	output := string(stdout)
	if !strings.Contains(output, `"compatibility-only"`) {
		t.Fatalf("compatibility list did not include compatibility module:\n%s", stdout)
	}
	if strings.Contains(output, `"hero"`) {
		t.Fatalf("compatibility list unexpectedly included recommended module:\n%s", stdout)
	}
	if !strings.Contains(output, `"lifecycle": "compatibility"`) {
		t.Fatalf("compatibility list did not include lifecycle summary:\n%s", stdout)
	}
}

func setupLayoutListLifecycleTest(t *testing.T) {
	t.Helper()
	oldJSON := jsonOutput
	oldFilters := layoutListFilters
	t.Cleanup(func() {
		jsonOutput = oldJSON
		layoutListFilters = oldFilters
		layoutcatalog.ResetDefaultCatalogForTests()
	})

	dir := t.TempDir()
	yaml := []byte(`schema_version: "1"
name: compatibility-only
lifecycle: compatibility
version: "1.0.0"
category: opening
serves: [attention]
metadata:
  author: test
  provenance: custom
`)
	if err := os.WriteFile(filepath.Join(dir, "compatibility-only.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MD2WECHAT_LAYOUT_DIR", dir)
	jsonOutput = true
	layoutListFilters = struct {
		category    string
		serves      string
		contentType string
		industry    string
		tag         string
		lifecycle   string
	}{}
	layoutcatalog.ResetDefaultCatalogForTests()
}

func TestLayoutShowJSONReturnsSpec(t *testing.T) {
	oldJSON := jsonOutput
	t.Cleanup(func() {
		jsonOutput = oldJSON
		layoutcatalog.ResetDefaultCatalogForTests()
	})

	jsonOutput = true
	layoutcatalog.ResetDefaultCatalogForTests()

	stdout := captureStdout(t, func() {
		if err := layoutShowCmd.RunE(layoutShowCmd, []string{"hero"}); err != nil {
			t.Fatalf("layoutShowCmd.RunE() error = %v", err)
		}
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout)
	}
	if response["success"] != true {
		t.Fatalf("expected success in response: %#v", response)
	}
	data, _ := response["data"].(map[string]any)
	if data["spec"] == nil {
		t.Fatalf("expected spec in response data: %#v", data)
	}
	spec, _ := data["spec"].(map[string]any)
	if spec["body_format"] != layoutcatalog.BodyFormatFields {
		t.Fatalf("expected hero body_format fields, got %#v", spec["body_format"])
	}
}

func TestLayoutListCompatibilityIsolation(t *testing.T) {
	oldJSON := jsonOutput
	oldFilters := layoutListFilters
	t.Cleanup(func() {
		jsonOutput = oldJSON
		layoutListFilters = oldFilters
		layoutcatalog.ResetDefaultCatalogForTests()
	})
	jsonOutput = true
	layoutcatalog.ResetDefaultCatalogForTests()

	listNames := func(lifecycle string) []string {
		t.Helper()
		layoutListFilters = struct {
			category, serves, contentType, industry, tag, lifecycle string
		}{lifecycle: lifecycle}
		stdout := captureStdout(t, func() {
			if err := layoutListCmd.RunE(layoutListCmd, nil); err != nil {
				t.Fatalf("layoutListCmd.RunE() error = %v", err)
			}
		})
		var response struct {
			Data struct {
				Count   int `json:"count"`
				Modules []struct {
					Name string `json:"name"`
				} `json:"modules"`
			} `json:"data"`
		}
		if err := json.Unmarshal(stdout, &response); err != nil {
			t.Fatalf("invalid json: %v\n%s", err, stdout)
		}
		names := make([]string, 0, len(response.Data.Modules))
		for _, module := range response.Data.Modules {
			names = append(names, module.Name)
		}
		if response.Data.Count != len(names) {
			t.Fatalf("count = %d, modules = %d", response.Data.Count, len(names))
		}
		return names
	}

	defaultNames := listNames("")
	if len(defaultNames) != 53 {
		t.Fatalf("default count = %d, want 53", len(defaultNames))
	}
	if got := buildLayoutCapabilityData()["module_count"]; got != 53 {
		t.Fatalf("capability module_count = %#v, want 53", got)
	}
	for _, legacy := range []string{"dialogue", "gallery", "longimage"} {
		if slices.Contains(defaultNames, legacy) {
			t.Fatalf("default list includes compatibility module %q", legacy)
		}
	}
	compatibilityNames := listNames(layoutcatalog.LifecycleCompatibility)
	if want := []string{"dialogue", "gallery", "longimage"}; !slices.Equal(compatibilityNames, want) {
		t.Fatalf("compatibility modules = %v, want %v", compatibilityNames, want)
	}
}

func TestLayoutShowCompatibilityGallery(t *testing.T) {
	oldJSON := jsonOutput
	t.Cleanup(func() {
		jsonOutput = oldJSON
		layoutcatalog.ResetDefaultCatalogForTests()
	})
	jsonOutput = true
	layoutcatalog.ResetDefaultCatalogForTests()

	stdout := captureStdout(t, func() {
		if err := layoutShowCmd.RunE(layoutShowCmd, []string{"gallery"}); err != nil {
			t.Fatalf("layoutShowCmd.RunE() error = %v", err)
		}
	})
	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout)
	}
	data, _ := response["data"].(map[string]any)
	spec, _ := data["spec"].(map[string]any)
	if spec["Name"] != "gallery" || spec["Lifecycle"] != layoutcatalog.LifecycleCompatibility {
		t.Fatalf("gallery spec = %#v", spec)
	}
}

func TestLayoutRenderHeroProducesBlock(t *testing.T) {
	oldJSON := jsonOutput
	oldVars := append([]string(nil), layoutRenderVars...)
	t.Cleanup(func() {
		jsonOutput = oldJSON
		layoutRenderVars = oldVars
		layoutcatalog.ResetDefaultCatalogForTests()
	})

	jsonOutput = true
	layoutcatalog.ResetDefaultCatalogForTests()
	layoutRenderVars = []string{"eyebrow=深度观察", "title=公众号排版的真问题"}

	stdout := captureStdout(t, func() {
		if err := layoutRenderCmd.RunE(layoutRenderCmd, []string{"hero"}); err != nil {
			t.Fatalf("layoutRenderCmd.RunE() error = %v", err)
		}
	})

	if !strings.Contains(string(stdout), `:::hero`) {
		t.Errorf("expected :::hero in output:\n%s", stdout)
	}
}

func TestLayoutRenderBodyFileComposesBlock(t *testing.T) {
	setupComplexLayoutRenderTest(t)
	bodyPath := filepath.Join(t.TempDir(), "gallery.md")
	if err := os.WriteFile(bodyPath, []byte("![产品界面](https://example.com/a.jpg) | 移动端首页\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	layoutRenderCaption = "产品截图"
	layoutRenderBodyFile = bodyPath

	stdout := captureStdout(t, func() {
		if err := layoutRenderCmd.RunE(layoutRenderCmd, []string{"gallery-caption"}); err != nil {
			t.Fatalf("layoutRenderCmd.RunE() error = %v", err)
		}
	})

	for _, want := range []string{
		`:::gallery-caption[产品截图]`,
		`![产品界面](https://example.com/a.jpg) | 移动端首页`,
	} {
		if !strings.Contains(string(stdout), want) {
			t.Fatalf("output missing %q:\n%s", want, stdout)
		}
	}
}

func TestLayoutRenderBodyFileDashReadsStdin(t *testing.T) {
	setupComplexLayoutRenderTest(t)
	layoutRenderParams = []string{"columns=2"}
	layoutRenderBodyFile = "-"
	stdinReader = strings.NewReader("![移动端](https://example.com/mobile.jpg)\n")

	stdout := captureStdout(t, func() {
		if err := layoutRenderCmd.RunE(layoutRenderCmd, []string{"gallery-grid"}); err != nil {
			t.Fatalf("layoutRenderCmd.RunE() error = %v", err)
		}
	})

	for _, want := range []string{
		`:::gallery-grid{columns=2}`,
		`![移动端](https://example.com/mobile.jpg)`,
	} {
		if !strings.Contains(string(stdout), want) {
			t.Fatalf("output missing %q:\n%s", want, stdout)
		}
	}
}

func setupComplexLayoutRenderTest(t *testing.T) {
	t.Helper()
	oldJSON := jsonOutput
	oldVars := append([]string(nil), layoutRenderVars...)
	oldParams := append([]string(nil), layoutRenderParams...)
	oldCaption := layoutRenderCaption
	oldBodyFile := layoutRenderBodyFile
	oldReader := stdinReader
	t.Cleanup(func() {
		jsonOutput = oldJSON
		layoutRenderVars = oldVars
		layoutRenderParams = oldParams
		layoutRenderCaption = oldCaption
		layoutRenderBodyFile = oldBodyFile
		stdinReader = oldReader
		layoutcatalog.ResetDefaultCatalogForTests()
	})

	dir := t.TempDir()
	yaml := []byte(`schema_version: "1"
name: gallery-grid
lifecycle: recommended
body_format: markdown_images
version: "1.0.0"
category: body
serves: [readability]
opener:
  param_style: braces
  params:
    - name: columns
      enum: ["2", "3"]
body:
  min_images: 1
metadata:
  author: test
  provenance: custom
`)
	if err := os.WriteFile(filepath.Join(dir, "gallery-grid.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	captionYAML := []byte(`schema_version: "1"
name: gallery-caption
lifecycle: recommended
body_format: markdown_images
version: "1.0.0"
category: body
serves: [readability]
opener:
  caption: true
body:
  min_images: 1
metadata:
  author: test
  provenance: custom
`)
	if err := os.WriteFile(filepath.Join(dir, "gallery-caption.yaml"), captionYAML, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MD2WECHAT_LAYOUT_DIR", dir)
	jsonOutput = true
	layoutRenderVars = nil
	layoutRenderParams = nil
	layoutRenderCaption = ""
	layoutRenderBodyFile = ""
	layoutcatalog.ResetDefaultCatalogForTests()
}

func TestLayoutValidateUnknownWarns(t *testing.T) {
	oldJSON := jsonOutput
	oldStdin := layoutValidateStdin
	oldReader := stdinReader
	t.Cleanup(func() {
		jsonOutput = oldJSON
		layoutValidateStdin = oldStdin
		stdinReader = oldReader
		layoutcatalog.ResetDefaultCatalogForTests()
	})

	jsonOutput = true
	layoutValidateStdin = true
	stdinReader = strings.NewReader(":::futuristic-block\nfoo: bar\n:::\n")
	layoutcatalog.ResetDefaultCatalogForTests()

	stdout := captureStdout(t, func() {
		// unknown block produces a warning, not an error — result is informational
		_ = layoutValidateCmd.RunE(layoutValidateCmd, nil)
	})

	if !strings.Contains(string(stdout), "futuristic-block") {
		t.Errorf("expected unknown module to appear in warnings:\n%s", stdout)
	}
}

func TestLayoutShowNotFound(t *testing.T) {
	oldJSON := jsonOutput
	t.Cleanup(func() {
		jsonOutput = oldJSON
		layoutcatalog.ResetDefaultCatalogForTests()
	})

	jsonOutput = true
	layoutcatalog.ResetDefaultCatalogForTests()

	if err := layoutShowCmd.RunE(layoutShowCmd, []string{"nonexistent-module-xyz"}); err == nil {
		t.Fatal("expected error for nonexistent module")
	} else if cliErr, ok := err.(*cliError); !ok || cliErr.Code != codeLayoutModuleNotFound {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func TestRenderCmdRowsJSONInput(t *testing.T) {
	oldJSON := jsonOutput
	oldVars := append([]string(nil), layoutRenderVars...)
	t.Cleanup(func() {
		jsonOutput = oldJSON
		layoutRenderVars = oldVars
		layoutcatalog.ResetDefaultCatalogForTests()
	})

	jsonOutput = true
	layoutcatalog.ResetDefaultCatalogForTests()
	// toc rows schema: number | title | description (min_columns: 2)
	layoutRenderVars = []string{`rows=[["01","第一章","概述"]]`}

	stdout := captureStdout(t, func() {
		if err := layoutRenderCmd.RunE(layoutRenderCmd, []string{"toc"}); err != nil {
			t.Fatalf("layoutRenderCmd.RunE() error = %v", err)
		}
	})

	if !strings.Contains(string(stdout), ":::toc") {
		t.Errorf("expected :::toc in output:\n%s", stdout)
	}
}

func TestLayoutRenderMissingRequiredField(t *testing.T) {
	oldJSON := jsonOutput
	oldVars := append([]string(nil), layoutRenderVars...)
	t.Cleanup(func() {
		jsonOutput = oldJSON
		layoutRenderVars = oldVars
		layoutcatalog.ResetDefaultCatalogForTests()
	})

	jsonOutput = true
	layoutcatalog.ResetDefaultCatalogForTests()
	// hero requires eyebrow and title — omit both
	layoutRenderVars = nil

	if err := layoutRenderCmd.RunE(layoutRenderCmd, []string{"hero"}); err == nil {
		t.Fatal("expected error for missing required field")
	} else if cliErr, ok := err.(*cliError); !ok || cliErr.Code != codeLayoutMissingRequiredField {
		t.Fatalf("unexpected error code: %#v", err)
	}
}
