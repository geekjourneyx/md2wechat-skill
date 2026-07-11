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

func TestEnvOverrideTrumpsBuiltin(t *testing.T) {
	ResetDefaultCatalogForTests()
	t.Cleanup(ResetDefaultCatalogForTests)

	dir := t.TempDir()
	override := filepath.Join(dir, "opening")
	if err := os.MkdirAll(override, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := []byte(`schema_version: "1"
name: hero
version: "999.0.0"
category: opening
serves: [attention]
metadata:
  author: test
  provenance: override
`)
	if err := os.WriteFile(filepath.Join(override, "hero.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MD2WECHAT_LAYOUT_DIR", dir)

	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	spec, _ := c.Get("hero")
	if spec.Version != "999.0.0" {
		t.Errorf("override not applied: version = %q", spec.Version)
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
	for _, provenance := range []string{"builtin", "custom"} {
		t.Run(provenance, func(t *testing.T) {
			yaml := []byte(`schema_version: "1"
name: block
version: "1.0.0"
category: opening
serves: [attention]
metadata:
  author: test
  provenance: ` + provenance + `
`)
			_, err := parseLayoutSpec(yaml)
			if err == nil || !strings.Contains(err.Error(), "reserved") {
				t.Fatalf("parseLayoutSpec() error = %v, want reserved-name rejection", err)
			}
		})
	}
}

func TestInvalidCustomModuleNamesCannotLoadRenderOrValidate(t *testing.T) {
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
  provenance: custom
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

func TestLoadFromDirRejectsInvalidModuleName(t *testing.T) {
	dir := t.TempDir()
	yaml := []byte(`schema_version: "1"
name: Bad
version: "1.0.0"
category: opening
serves: [attention]
metadata:
  author: test
  provenance: custom
`)
	if err := os.WriteFile(filepath.Join(dir, "bad.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	c := NewCatalog()
	if err := c.loadFromDir(dir); err == nil || !strings.Contains(err.Error(), "invalid layout module name") {
		t.Fatalf("loadFromDir() error = %v", err)
	}
}

func TestLoadFromDirRejectsReservedModuleName(t *testing.T) {
	dir := t.TempDir()
	yaml := []byte(`schema_version: "1"
name: block
version: "1.0.0"
category: opening
serves: [attention]
metadata:
  author: test
  provenance: custom
`)
	if err := os.WriteFile(filepath.Join(dir, "block.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	c := NewCatalog()
	if err := c.loadFromDir(dir); err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("loadFromDir() error = %v, want reserved-name rejection", err)
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
  provenance: custom
`)
	spec, err := parseLayoutSpec(yaml)
	if err != nil {
		t.Fatalf("parseLayoutSpec failed: %v", err)
	}
	if spec.BodyFormat != BodyFormatFields {
		t.Fatalf("BodyFormat = %q, want %q", spec.BodyFormat, BodyFormatFields)
	}
}

func TestCustomLayoutBlankLifecycleDefaultsToRecommended(t *testing.T) {
	ResetDefaultCatalogForTests()
	t.Cleanup(ResetDefaultCatalogForTests)

	dir := t.TempDir()
	yaml := []byte(`schema_version: "1"
name: custom-current
version: "1.0.0"
category: opening
serves: [attention]
metadata:
  author: test
  provenance: custom
`)
	if err := os.WriteFile(filepath.Join(dir, "custom-current.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MD2WECHAT_LAYOUT_DIR", dir)

	c, err := DefaultCatalog()
	if err != nil {
		t.Fatalf("DefaultCatalog() failed: %v", err)
	}
	spec, ok := c.Get("custom-current")
	if !ok {
		t.Fatal("expected custom-current module to be present")
	}
	if spec.Lifecycle != LifecycleRecommended {
		t.Fatalf("Lifecycle = %q, want %q", spec.Lifecycle, LifecycleRecommended)
	}
}

func TestCustomLayoutRejectsInvalidLifecycle(t *testing.T) {
	ResetDefaultCatalogForTests()
	t.Cleanup(ResetDefaultCatalogForTests)

	dir := t.TempDir()
	yaml := []byte(`schema_version: "1"
name: custom-experimental
lifecycle: experimental
version: "1.0.0"
category: opening
serves: [attention]
metadata:
  author: test
  provenance: custom
`)
	if err := os.WriteFile(filepath.Join(dir, "custom-experimental.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MD2WECHAT_LAYOUT_DIR", dir)

	_, err := DefaultCatalog()
	if err == nil {
		t.Fatal("expected invalid lifecycle to be rejected")
	}
	if !strings.Contains(err.Error(), `invalid lifecycle "experimental"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnvOverrideBeatsLocalDir(t *testing.T) {
	localDir := t.TempDir()
	localOpening := filepath.Join(localDir, "opening")
	if err := os.MkdirAll(localOpening, 0o755); err != nil {
		t.Fatal(err)
	}
	localYAML := []byte(`schema_version: "1"
name: hero
version: "2.0.0"
category: opening
serves: [attention]
metadata:
  author: local
  provenance: local
`)
	if err := os.WriteFile(filepath.Join(localOpening, "hero.yaml"), localYAML, 0o644); err != nil {
		t.Fatal(err)
	}

	envDir := t.TempDir()
	envOpening := filepath.Join(envDir, "opening")
	if err := os.MkdirAll(envOpening, 0o755); err != nil {
		t.Fatal(err)
	}
	envYAML := []byte(`schema_version: "1"
name: hero
version: "3.0.0"
category: opening
serves: [attention]
metadata:
  author: env
  provenance: env
`)
	if err := os.WriteFile(filepath.Join(envOpening, "hero.yaml"), envYAML, 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("MD2WECHAT_LAYOUT_DIR", envDir)
	ResetDefaultCatalogForTests()
	t.Cleanup(ResetDefaultCatalogForTests)

	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	spec, _ := c.Get("hero")
	if spec.Version != "3.0.0" {
		t.Errorf("env override should win, got version %q", spec.Version)
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
