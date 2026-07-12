package layoutcatalog

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/assets"
)

func TestLoadBuiltinIncludesHero(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	spec, ok := c.Get("hero")
	if !ok {
		t.Fatalf("expected hero module to be present")
	}
	if spec.Category != "opening" {
		t.Errorf("hero.Category = %q, want opening", spec.Category)
	}
}

func TestCatalogIgnoresEnvironmentLayoutDirectory(t *testing.T) {
	dir := t.TempDir()
	writeLayoutOverride(t, dir)
	t.Setenv("MD2WECHAT_LAYOUT_DIR", dir)

	assertCatalogIgnoresOverride(t)
}

func TestCatalogIgnoresUserLayoutDirectory(t *testing.T) {
	home := t.TempDir()
	writeLayoutOverride(t, filepath.Join(home, ".config", "md2wechat", "layout"))
	t.Setenv("HOME", home)

	assertCatalogIgnoresOverride(t)
}

func TestCatalogIgnoresWorkingDirectoryLayout(t *testing.T) {
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	writeLayoutOverride(t, filepath.Join(root, "layout"))
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	assertCatalogIgnoresOverride(t)
}

func writeLayoutOverride(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := []byte(`schema_version: "1"
name: user-only
version: "1.0.0"
category: opening
serves: [attention]
metadata:
  author: test
  provenance: external-fixture
example: |
  :::user-only
  :::
`)
	if err := os.WriteFile(filepath.Join(dir, "user-only.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "broken.yaml"), []byte("not: [valid"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertCatalogIgnoresOverride(t *testing.T) {
	t.Helper()
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if _, ok := c.Get("user-only"); ok {
		t.Fatal("user-only module was loaded outside the embedded catalog")
	}
}

func TestParseLayoutSpecRejectsInvalidServes(t *testing.T) {
	yaml := []byte(`schema_version: "1"
name: bad
version: "1.0.0"
category: opening
serves: [bogus]
metadata:
  author: x
  provenance: builtin
`)
	_, err := parseLayoutSpec(yaml)
	if err == nil {
		t.Fatal("expected error for invalid serves value")
	}
}

func TestParseLayoutSpecRejectsReservedModuleName(t *testing.T) {
	yaml := []byte(`schema_version: "1"
name: block
version: "1.0.0"
category: opening
serves: [attention]
metadata:
  author: test
  provenance: test-fixture
`)
	_, err := parseLayoutSpec(yaml)
	if err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("parseLayoutSpec() error = %v, want reserved-name rejection", err)
	}
}

func TestInvalidModuleNamesCannotParseRenderOrValidate(t *testing.T) {
	for _, name := range []string{"Bad", "bad.name", "_bad", "bad/name", "bad name"} {
		t.Run(name, func(t *testing.T) {
			yaml := []byte(`schema_version: "1"
name: "` + name + `"
body_format: fields
version: "1.0.0"
category: opening
serves: [attention]
fields:
  required:
    - name: title
metadata:
  author: test
  provenance: test-fixture
example: |
  :::fixture-current
  :::
`)
			if _, err := parseLayoutSpec(yaml); err == nil || !strings.Contains(err.Error(), "invalid layout module name") {
				t.Fatalf("parseLayoutSpec(%q) error = %v", name, err)
			}

			c := NewCatalog()
			c.modules[name] = &LayoutSpec{
				Name: name, BodyFormat: BodyFormatFields,
				Fields: &FieldsSpec{Required: []FieldSpec{{Name: "title"}}},
			}
			if _, err := c.RenderBlock(name, RenderInput{Fields: map[string]any{"title": "Value"}}); !errors.Is(err, ErrInvalidFieldValue) {
				t.Fatalf("RenderBlock(%q) error = %v, want ErrInvalidFieldValue", name, err)
			}
			report := c.Validate(":::" + name + "\ntitle: Value\n:::\n")
			if len(report.Errors) == 0 && len(report.Warnings) == 0 {
				t.Fatalf("Validate(%q) resolved an invalid module name without diagnostics", name)
			}
		})
	}
}

func TestParseLayoutSpecRejectsInvalidBodyFormat(t *testing.T) {
	yaml := []byte(`schema_version: "1"
name: bad
body_format: xml
version: "1.0.0"
category: opening
serves: [attention]
metadata:
  author: x
  provenance: builtin
`)
	_, err := parseLayoutSpec(yaml)
	if err == nil {
		t.Fatal("expected error for invalid body_format")
	}
}

func TestParseLayoutSpecRejectsUnknownYAMLFields(t *testing.T) {
	yaml := []byte(`schema_version: "1"
name: strict-yaml
body_format: fields
body_fomat: rows
version: "1.0.0"
category: opening
serves: [attention]
metadata:
  author: test
  provenance: test-fixture
`)
	if _, err := parseLayoutSpec(yaml); err == nil || !strings.Contains(err.Error(), "body_fomat") {
		t.Fatalf("parseLayoutSpec() error = %v, want unknown field rejection", err)
	}
}

func TestParseLayoutSpecDefaultsLegacyBodyFormat(t *testing.T) {
	yaml := []byte(`schema_version: "1"
name: legacy
version: "1.0.0"
category: opening
serves: [attention]
fields:
  required:
    - name: title
metadata:
  author: x
  provenance: test-fixture
`)
	spec, err := parseLayoutSpec(yaml)
	if err != nil {
		t.Fatalf("parseLayoutSpec failed: %v", err)
	}
	if spec.BodyFormat != BodyFormatFields {
		t.Fatalf("BodyFormat = %q, want %q", spec.BodyFormat, BodyFormatFields)
	}
}

func TestParseLayoutSpecDefaultsBlankLifecycleToRecommended(t *testing.T) {
	yaml := []byte(`schema_version: "1"
name: current
version: "1.0.0"
category: opening
serves: [attention]
example: |
  :::current
  :::
metadata:
  author: test
  provenance: test-fixture
`)
	spec, err := parseLayoutSpec(yaml)
	if err != nil {
		t.Fatalf("parseLayoutSpec() failed: %v", err)
	}
	if spec.Lifecycle != LifecycleRecommended {
		t.Fatalf("Lifecycle = %q, want %q", spec.Lifecycle, LifecycleRecommended)
	}
}

func TestParseLayoutSpecRejectsInvalidLifecycle(t *testing.T) {
	yaml := []byte(`schema_version: "1"
name: experimental
lifecycle: experimental
version: "1.0.0"
category: opening
serves: [attention]
metadata:
  author: test
  provenance: test-fixture
`)
	_, err := parseLayoutSpec(yaml)
	if err == nil {
		t.Fatal("expected invalid lifecycle to be rejected")
	}
	if !strings.Contains(err.Error(), `invalid lifecycle "experimental"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAllBuiltinModulesLoadCleanly(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if got := len(c.modules); got < 38 {
		t.Errorf("expected at least 38 modules, got %d", got)
	}
	for name, m := range c.modules {
		if m.BodyFormat == "" {
			t.Errorf("%s missing body_format", name)
		}
		if m.Metadata.Provenance == "" {
			t.Errorf("%s missing provenance", name)
		}
		if m.Metadata.InspiredBy == "" && m.Metadata.Provenance == "builtin" {
			t.Errorf("%s builtin module missing inspired_by", name)
		}
	}
}

func TestAllBuiltinModulesDeclareBodyFormat(t *testing.T) {
	cats, err := assets.ListBuiltinLayoutCategories()
	if err != nil {
		t.Fatal(err)
	}
	for _, cat := range cats {
		names, err := assets.ListBuiltinLayouts(cat)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range names {
			data, err := assets.ReadBuiltinLayout(cat, name)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), "\nbody_format: ") {
				t.Errorf("%s/%s missing explicit body_format", cat, name)
			}
		}
	}
}
