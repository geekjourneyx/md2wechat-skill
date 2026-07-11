package layoutcatalog

import (
	"fmt"
	"strings"
	"testing"
)

func TestExecutableWitnessContract(t *testing.T) {
	c := NewCatalog()
	c.modules["demo"] = &LayoutSpec{
		Name: "demo", Lifecycle: LifecycleRecommended, BodyFormat: BodyFormatFields,
		Fields: &FieldsSpec{Required: []FieldSpec{{Name: "title"}}},
	}
	canonical := ":::demo\ntitle: Canonical witness\n:::\n"
	variant := ":::demo\nvariant: compact\ntitle: Compact witness\n:::\n"

	if err := checkExecutableWitness(c, "demo", "", canonical, "Canonical witness"); err != nil {
		t.Fatal(err)
	}
	if err := checkExecutableWitness(c, "demo", "compact", variant, "Compact witness"); err != nil {
		t.Fatal(err)
	}
	if err := checkExecutableWitness(c, "demo", "compact", canonical, "Canonical witness"); err == nil {
		t.Fatal("variant witness must select its variant")
	}
}

func TestExecutableWitnessRejectsInvalidContracts(t *testing.T) {
	c := NewCatalog()
	c.modules["demo"] = &LayoutSpec{
		Name: "demo", Lifecycle: LifecycleRecommended, BodyFormat: BodyFormatFields,
		Fields: &FieldsSpec{Required: []FieldSpec{{Name: "title"}}},
	}
	tests := []struct {
		name, module, example, assertion string
	}{
		{name: "empty example", module: "demo"},
		{name: "wrong opener", module: "demo", example: ":::other\ntitle: value\n:::\n"},
		{name: "invalid body", module: "demo", example: ":::demo\n:::\n"},
		{name: "missing assertion", module: "demo", example: ":::demo\ntitle: value\n:::\n", assertion: "absent"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkExecutableWitness(c, tt.module, "", tt.example, tt.assertion); err == nil {
				t.Fatal("expected witness contract error")
			}
		})
	}
}

func TestBuiltinCanonicalWitnesses(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	for _, spec := range c.ListFiltered(ListFilter{}) {
		if spec.ExampleAssertContains == "" {
			continue
		}
		t.Run(spec.Name, func(t *testing.T) {
			if err := checkExecutableWitness(c, spec.Name, "", spec.Example, spec.ExampleAssertContains); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestBuiltinVariantWitnesses(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	for _, spec := range c.ListFiltered(ListFilter{}) {
		for _, variant := range spec.Variants {
			variant := variant
			t.Run(spec.Name+"/"+variant.Name, func(t *testing.T) {
				if err := checkExecutableWitness(c, spec.Name, variant.Name, variant.Example, variant.AssertContains); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestCustomLayoutWithoutExamplesRemainsCompatible(t *testing.T) {
	spec, err := parseLayoutSpec([]byte(`schema_version: "1"
name: custom
body_format: fields
version: "1.0.0"
category: opening
serves: [attention]
metadata:
  author: test
  provenance: custom
`))
	if err != nil {
		t.Fatalf("custom layout without examples failed to load: %v", err)
	}
	if spec.Example != "" || len(spec.Variants) != 0 {
		t.Fatalf("unexpected witness metadata: example=%q variants=%v", spec.Example, spec.Variants)
	}
}

func TestParseLayoutSpecValidatesVariantIdentity(t *testing.T) {
	tests := []struct {
		name     string
		variants string
		want     string
	}{
		{name: "duplicate names", variants: "  - name: compact\n  - name: compact\n", want: "duplicate variant"},
		{name: "alias collides with name", variants: "  - name: compact\n    aliases: [dense]\n  - name: dense\n", want: "duplicate variant"},
		{name: "duplicate aliases", variants: "  - name: compact\n    aliases: [dense, dense]\n", want: "duplicate variant"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := baseVariantYAML(tt.variants)
			_, err := parseLayoutSpec([]byte(yaml))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("parseLayoutSpec() error = %v, want category %q", err, tt.want)
			}
		})
	}
}

func TestParseLayoutSpecValidatesWitnessAssertions(t *testing.T) {
	tests := []struct {
		name, extra, want string
	}{
		{name: "canonical", extra: "example: |\n  :::demo\n  title: Present\n  :::\nexample_assert_contains: Missing\n", want: "example_assert_contains"},
		{name: "variant", extra: "variants:\n  - name: compact\n    example: |\n      :::demo\n      variant: compact\n      title: Present\n      :::\n    assert_contains: Missing\n", want: "assert_contains"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseLayoutSpec([]byte(baseLayoutYAML + tt.extra))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("parseLayoutSpec() error = %v, want category %q", err, tt.want)
			}
		})
	}
}

func TestValidatorIndependentNegativeBoundaries(t *testing.T) {
	tests := []struct {
		name, markdown, category string
		spec                     *LayoutSpec
	}{
		{name: "missing required field", markdown: ":::demo\nother: value\n:::\n", category: "required field", spec: &LayoutSpec{Name: "demo", BodyFormat: BodyFormatFields, Fields: &FieldsSpec{Required: []FieldSpec{{Name: "title"}}}}},
		{name: "wrong row width", markdown: ":::demo\none\n:::\n", category: "columns", spec: &LayoutSpec{Name: "demo", BodyFormat: BodyFormatRows, Rows: &RowsSpec{Delimiter: "|", MinColumns: 2}}},
		{name: "incomplete repeated group", markdown: ":::demo\nitem: first\n:::\n", category: "group field", spec: &LayoutSpec{Name: "demo", BodyFormat: BodyFormatMarkdownFields, Fields: &FieldsSpec{Optional: []FieldSpec{{Name: "item"}, {Name: "detail"}}}, Body: &BodySpec{Group: &FieldGroupSpec{Start: "item", Required: []string{"item", "detail"}, Min: 1}}}},
		{name: "missing split divider", markdown: ":::demo\nleft\nright\n:::\n", category: "separator", spec: &LayoutSpec{Name: "demo", BodyFormat: BodyFormatSplit}},
		{name: "too few images", markdown: ":::demo\n![one](https://example.com/one.png)\n:::\n", category: "image(s)", spec: &LayoutSpec{Name: "demo", BodyFormat: BodyFormatMarkdownImages, Body: &BodySpec{MinImages: 2}}},
		{name: "unknown opener parameter", markdown: ":::demo{unknown=value}\ntitle: value\n:::\n", category: "undeclared opener parameter", spec: &LayoutSpec{Name: "demo", BodyFormat: BodyFormatFields, Opener: &OpenerSpec{ParamStyle: ParamStyleBraces, Params: []ParamSpec{{Name: "variant"}}}, Fields: &FieldsSpec{Required: []FieldSpec{{Name: "title"}}}}},
		{name: "invalid enum", markdown: ":::demo\nmode: wrong\n:::\n", category: "must be one of", spec: &LayoutSpec{Name: "demo", BodyFormat: BodyFormatFields, Fields: &FieldsSpec{Required: []FieldSpec{{Name: "mode", Enum: []string{"valid"}}}}}},
		{name: "malformed JSON", markdown: ":::demo\n{not-json}\n:::\n", category: "invalid character", spec: &LayoutSpec{Name: "demo", BodyFormat: BodyFormatJSONObject}},
		{name: "unmatched closing fence", markdown: ":::demo\ntitle: value\n", category: "unterminated", spec: &LayoutSpec{Name: "demo", BodyFormat: BodyFormatFields, Fields: &FieldsSpec{Required: []FieldSpec{{Name: "title"}}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCatalog()
			c.modules[tt.spec.Name] = tt.spec
			report := c.Validate(tt.markdown)
			if len(report.Errors) == 0 {
				t.Fatal("expected at least one validation error")
			}
			if !strings.Contains(report.Errors[0].Message, tt.category) {
				t.Fatalf("error category = %q, want %q", report.Errors[0].Message, tt.category)
			}
		})
	}
}

func checkExecutableWitness(c *Catalog, module, variant, example, assertion string) error {
	if strings.TrimSpace(example) == "" {
		return fmt.Errorf("%s witness example is empty", module)
	}
	lines := strings.Split(strings.TrimSpace(example), "\n")
	opener, err := parseBlockOpener(strings.TrimRight(lines[0], "\r"))
	if err != nil {
		return fmt.Errorf("parse %s witness opener: %w", module, err)
	}
	if opener.Name != module {
		return fmt.Errorf("witness opener %q does not match module %q", opener.Name, module)
	}
	report := c.Validate(example)
	if len(report.Errors) != 0 {
		return fmt.Errorf("%s witness validation errors: %v", module, report.Errors)
	}
	if variant != "" {
		spec, ok := c.Get(module)
		if !ok {
			return fmt.Errorf("module %q not found", module)
		}
		body := lines[1:]
		if len(body) > 0 && strings.TrimSpace(strings.TrimRight(body[len(body)-1], "\r")) == ":::" {
			body = body[:len(body)-1]
		}
		facts, issues := parseBodyFacts(spec, spec.BodyFormat, body)
		if len(issues) != 0 {
			return fmt.Errorf("parse %s witness facts: %v", module, issues)
		}
		selected := opener.Params["variant"] == variant || opener.Params["type"] == variant ||
			hasWitnessFact(facts, "variant", variant) || hasWitnessFact(facts, "type", variant) ||
			hasStructuredWitnessSelector(body, "variant", variant) || hasStructuredWitnessSelector(body, "type", variant)
		if !selected {
			return fmt.Errorf("%s witness does not select variant %q", module, variant)
		}
	}
	if assertion != "" && !strings.Contains(example, assertion) {
		return fmt.Errorf("%s witness does not contain assertion %q", module, assertion)
	}
	return nil
}

func hasStructuredWitnessSelector(body []string, name, value string) bool {
	for _, line := range body {
		key, got, ok := strings.Cut(strings.TrimSpace(strings.TrimRight(line, "\r")), ":")
		if ok && strings.TrimSpace(key) == name && strings.TrimSpace(got) == value {
			return true
		}
	}
	return false
}

func hasWitnessFact(facts bodyFacts, name, value string) bool {
	for _, got := range facts.fieldValues[name] {
		if got == value {
			return true
		}
	}
	return false
}

const baseLayoutYAML = `schema_version: "1"
name: demo
body_format: fields
version: "1.0.0"
category: opening
serves: [attention]
metadata:
  author: test
  provenance: custom
`

func baseVariantYAML(variants string) string {
	return baseLayoutYAML + "variants:\n" + variants
}
