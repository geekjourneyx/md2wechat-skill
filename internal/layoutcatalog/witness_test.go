package layoutcatalog

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type recommendedScenarioMap struct {
	SourceCommit               string   `yaml:"source_commit"`
	SourceFile                 string   `yaml:"source_file"`
	SourceSHA256               string   `yaml:"source_sha256"`
	GuideOnlyRecommendedSyntax []string `yaml:"guide_only_recommended_syntax"`
	Scenarios                  []struct {
		ID      string `yaml:"id"`
		Module  string `yaml:"module"`
		Variant string `yaml:"variant,omitempty"`
	} `yaml:"scenarios"`
}

const pinnedRecommendedScenarioOracle = `
hero-editorial|hero|editorial
hero-briefing|hero|briefing
hero-story|hero|story
hero-masthead|hero|masthead
cards|cards|
part|part|
epilogue|epilogue|
section-title-marker|section-title|marker
section-title-divider|section-title|divider
section-title-numbered|section-title|numbered
section-title-frame|section-title|frame
section-title-focus|section-title|focus
section-title-vertical|section-title|vertical
toc|toc|
label-title|label-title|
audience-fit|audience-fit|
verdict|verdict|
myth-fact|myth-fact|
bridge|bridge|
manifesto|manifesto|
infographic-thesis|infographic|thesis
infographic-formula|infographic|formula
infographic-pullquote|infographic|pullquote
infographic-contrast|infographic|contrast
infographic-diagnosis|infographic|diagnosis
infographic-number|infographic|number
infographic-path|infographic|path
infographic-anatomy|infographic|anatomy
infographic-tradeoff|infographic|tradeoff
infographic-evidence-chain|infographic|evidence-chain
infographic-micro-case|infographic|micro-case
metrics|metrics|
compare|compare|
steps|steps|
timeline|timeline|
quote-light|quote|light
quote-brand|quote|brand
quote-proof|quote|proof
image-text|image-text|
image-compare|image-compare|
image-annotate|image-annotate|
image-steps|image-steps|
image-phone-shot|image-phone-shot|
author-card|author-card|
series|series|
subscribe|subscribe|
people|people|
cases|cases|
pricing|pricing|
faq|faq|
logos|logos|
checklist|checklist|
toolbox|toolbox|
specs|specs|
notice|notice|
summary-one-line|summary|
summary-three|summary|three
summary-decision|summary|decision
summary-save|summary|save
closing|closing|
cta-save-follow|cta|save-follow
cta-consult|cta|consult
cta-trial|cta|trial
callout|callout|
callout-warning|callout|
quote-card|quote-card|
stat-row|stat-row|
definition|definition|
tweet|tweet|
question|question|
comparison-table|comparison-table|
changelog|changelog|
resource-list|resource-list|
split|split|
flow|flow|
matrix|matrix|
dialogue-pair|dialogue-pair|
`

func TestRecommendedScenarioMappingMatchesPinnedSources(t *testing.T) {
	m := readRecommendedScenarioMap(t)
	if m.SourceCommit != "edcde64ae1be56f1a08a0617bb1862471e7e00b1" || m.SourceFile != "lib/advanced-module-groups.ts" {
		t.Fatalf("source = %q:%q", m.SourceCommit, m.SourceFile)
	}
	if m.SourceSHA256 != "e5443d6c7298bf592ec556395622f55e49721a418f2aa7578d8867555470e050" {
		t.Fatalf("source digest = %q", m.SourceSHA256)
	}
	if len(m.Scenarios) != 77 {
		t.Fatalf("scenario count = %d", len(m.Scenarios))
	}
	if err := comparePinnedScenarioTuples(m); err != nil {
		t.Fatal(err)
	}

	seenIDs, coveredModules := map[string]bool{}, map[string]bool{}
	for _, scenario := range m.Scenarios {
		if scenario.ID == "" || seenIDs[scenario.ID] {
			t.Fatalf("invalid duplicate id %q", scenario.ID)
		}
		seenIDs[scenario.ID] = true
		coveredModules[scenario.Module] = true
	}
	if len(coveredModules) != 51 {
		t.Fatalf("covered module count = %d, want 51", len(coveredModules))
	}
	wantGuideOnly := []string{
		"figure-caption", "gallery-grid", "gallery-story", "svg-reveal", "svg-swipe-gallery",
	}
	slices.Sort(m.GuideOnlyRecommendedSyntax)
	if !slices.Equal(m.GuideOnlyRecommendedSyntax, wantGuideOnly) {
		t.Fatalf("guide-only syntax = %v", m.GuideOnlyRecommendedSyntax)
	}
	diff := missingRecommendedNames(recommendedModuleNames, coveredModules)
	slices.Sort(diff)
	if !slices.Equal(diff, wantGuideOnly) {
		t.Fatalf("scenario-source gap = %v, want %v", diff, wantGuideOnly)
	}
}

func TestRecommendedScenarioCatalogProjection(t *testing.T) {
	t.Skip("catalog projection is implemented in Tasks 3-4")

	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	for _, scenario := range readRecommendedScenarioMap(t).Scenarios {
		spec, ok := c.Get(scenario.Module)
		if !ok || spec.Lifecycle != LifecycleRecommended {
			t.Fatalf("invalid target %+v", scenario)
		}
		assertScenarioHasWitness(t, spec, scenario.Variant)
	}
}

func TestPinnedScenarioTupleOracleRejectsFabricationAndReassignment(t *testing.T) {
	for _, mutate := range []func(*recommendedScenarioMap){
		func(m *recommendedScenarioMap) { m.Scenarios[0].ID = "fabricated-id" },
		func(m *recommendedScenarioMap) { m.Scenarios[0].Module = "cards" },
		func(m *recommendedScenarioMap) { m.Scenarios[0].Variant = "briefing" },
	} {
		m := readRecommendedScenarioMap(t)
		mutate(&m)
		if err := comparePinnedScenarioTuples(m); err == nil {
			t.Fatal("mutated pinned scenario tuple must fail")
		}
	}
}

func comparePinnedScenarioTuples(mapping recommendedScenarioMap) error {
	want := strings.Split(strings.TrimSpace(pinnedRecommendedScenarioOracle), "\n")
	got := make([]string, 0, len(mapping.Scenarios))
	for _, scenario := range mapping.Scenarios {
		got = append(got, strings.Join([]string{scenario.ID, scenario.Module, scenario.Variant}, "|"))
	}
	slices.Sort(want)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		return fmt.Errorf("pinned scenario tuples differ\ngot:  %v\nwant: %v", got, want)
	}
	return nil
}

func readRecommendedScenarioMap(t *testing.T) recommendedScenarioMap {
	t.Helper()
	data, err := os.ReadFile("testdata/recommended_scenarios.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var mapping recommendedScenarioMap
	if err := yaml.Unmarshal(data, &mapping); err != nil {
		t.Fatal(err)
	}
	return mapping
}

func assertScenarioHasWitness(t *testing.T, spec *LayoutSpec, variant string) {
	t.Helper()
	if variant == "" {
		if err := checkExecutableWitness(NewCatalogWithSpec(spec), spec.Name, "", spec.Example, spec.ExampleAssertContains); err != nil {
			t.Fatal(err)
		}
		return
	}
	for _, candidate := range spec.Variants {
		if candidate.Name != variant && !slices.Contains(candidate.Aliases, variant) {
			continue
		}
		c := NewCatalogWithSpec(spec)
		if err := c.ValidateWitness(WitnessContract{
			Module: spec.Name, Variant: candidate.Name, VariantAliases: candidate.Aliases,
			Example: candidate.Example, AssertContains: candidate.AssertContains,
		}); err != nil {
			t.Fatal(err)
		}
		return
	}
	t.Fatalf("%s has no declared witness for variant %q", spec.Name, variant)
}

func NewCatalogWithSpec(spec *LayoutSpec) *Catalog {
	c := NewCatalog()
	c.modules[spec.Name] = spec
	return c
}

func missingRecommendedNames(names []string, covered map[string]bool) []string {
	var missing []string
	for _, name := range names {
		if !covered[name] {
			missing = append(missing, name)
		}
	}
	return missing
}

func TestExecutableWitnessContract(t *testing.T) {
	c := NewCatalog()
	c.modules["demo"] = &LayoutSpec{
		Name: "demo", Lifecycle: LifecycleRecommended, BodyFormat: BodyFormatFields,
		Fields:   &FieldsSpec{Required: []FieldSpec{{Name: "title"}}, Optional: []FieldSpec{{Name: "variant"}}},
		Variants: []VariantSpec{{Name: "compact"}},
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

func TestExecutableWitnessAcceptsDeclaredVariantAlias(t *testing.T) {
	c := NewCatalog()
	c.modules["demo"] = &LayoutSpec{
		Name: "demo", BodyFormat: BodyFormatFields,
		Fields:   &FieldsSpec{Optional: []FieldSpec{{Name: "variant"}}},
		Variants: []VariantSpec{{Name: "compact", Aliases: []string{"dense"}}},
	}
	err := c.ValidateWitness(WitnessContract{
		Module: "demo", Variant: "compact", VariantAliases: []string{"dense"},
		Example: ":::demo\nvariant: dense\n:::\n",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestExecutableWitnessUsesEffectiveLastWriteSelector(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, variant, example string
		wantErr                bool
	}{
		{name: "type overrides variant", variant: "formula", example: ":::infographic\nvariant: formula\ntype: thesis\ntitle: 判断\n:::\n", wantErr: true},
		{name: "last type wins", variant: "formula", example: ":::infographic\ntype: thesis\ntype: formula\ntitle: 公式\nformula: 判断 + 行动\n:::\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.ValidateWitness(WitnessContract{Module: "infographic", Variant: tt.variant, Example: tt.example})
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateWitness() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
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
				if err := c.ValidateWitness(WitnessContract{
					Module: spec.Name, Variant: variant.Name, VariantAliases: variant.Aliases,
					Example: variant.Example, AssertContains: variant.AssertContains,
				}); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestBuiltinExecutableWitnessesAreComplete(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	for _, spec := range c.ListFiltered(ListFilter{}) {
		t.Run(spec.Name+"/canonical", func(t *testing.T) {
			if err := checkExecutableWitness(c, spec.Name, "", spec.Example, spec.ExampleAssertContains); err != nil {
				t.Fatal(err)
			}
		})
		for _, variant := range spec.Variants {
			variant := variant
			t.Run(spec.Name+"/"+variant.Name, func(t *testing.T) {
				if strings.TrimSpace(variant.UseWhen) == "" {
					t.Fatal("variant use_when is required")
				}
				if err := c.ValidateWitness(WitnessContract{
					Module: spec.Name, Variant: variant.Name, VariantAliases: variant.Aliases,
					Example: variant.Example, AssertContains: variant.AssertContains,
				}); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestSchemaParsingDefersExecutableWitnessRequirementToCatalogLoad(t *testing.T) {
	spec, err := parseLayoutSpec([]byte(`schema_version: "1"
name: schema-fixture
body_format: fields
version: "1.0.0"
category: opening
serves: [attention]
metadata:
  author: test
  provenance: test-fixture
`))
	if err != nil {
		t.Fatalf("schema fixture without examples failed to parse: %v", err)
	}
	if spec.Example != "" || len(spec.Variants) != 0 {
		t.Fatalf("unexpected witness metadata: example=%q variants=%v", spec.Example, spec.Variants)
	}
}

func TestSchemaParsingDefersVariantWitnessMetadataToCatalogLoad(t *testing.T) {
	spec, err := parseLayoutSpec([]byte(baseLayoutYAML + `fields:
  optional:
    - name: variant
    - name: title
variants:
  - name: compact
    required: [title]
`))
	if err != nil {
		t.Fatalf("variant without witness metadata failed to parse: %v", err)
	}
	if len(spec.Variants) != 1 || spec.Variants[0].Example != "" || spec.Variants[0].UseWhen != "" {
		t.Fatalf("unexpected variant: %+v", spec.Variants)
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
		{name: "spaced name", variants: "  - name: ' compact'\n", want: "surrounding whitespace"},
		{name: "spaced alias", variants: "  - name: compact\n    aliases: ['dense ']\n", want: "surrounding whitespace"},
		{name: "spaced collision fails closed", variants: "  - name: compact\n  - name: ' compact '\n", want: "surrounding whitespace"},
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

func TestParseLayoutSpecValidatesVariantRules(t *testing.T) {
	tests := []struct {
		name, variant, want string
	}{
		{name: "undeclared required", variant: "  - name: compact\n    use_when: 需要紧凑结构\n    required: [missing]\n    example: |\n      :::demo\n      variant: compact\n      title: Value\n      :::\n", want: "not declared"},
		{name: "empty required any", variant: "  - name: compact\n    use_when: 需要紧凑结构\n    required_any: [[]]\n    example: |\n      :::demo\n      variant: compact\n      title: Value\n      :::\n", want: "must not be empty"},
		{name: "undeclared shape field", variant: "  - name: compact\n    use_when: 需要紧凑结构\n    shapes:\n      - {field: missing, separator: '|', min_parts: 2}\n    example: |\n      :::demo\n      variant: compact\n      title: Value\n      :::\n", want: "shape field"},
		{name: "invalid shape minimum", variant: "  - name: compact\n    use_when: 需要紧凑结构\n    shapes:\n      - {field: title, separator: '|', min_parts: 1}\n    example: |\n      :::demo\n      variant: compact\n      title: Value\n      :::\n", want: "greater than 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := baseLayoutYAML + "fields:\n  optional:\n    - name: title\n    - name: variant\nvariants:\n" + tt.variant
			_, err := parseLayoutSpec([]byte(yaml))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("parseLayoutSpec() error = %v, want category %q", err, tt.want)
			}
		})
	}
}

func TestParseLayoutSpecValidatesFieldValueTypesAndDefaultRules(t *testing.T) {
	tests := []struct {
		name, field, extra, want string
	}{
		{name: "string type", field: "    - {name: title, value_type: string}\n"},
		{name: "legacy any type", field: "    - {name: title}\n"},
		{name: "unknown value type", field: "    - {name: title, value_type: number}\n", want: "value_type"},
		{name: "unknown default required field", field: "    - {name: title}\n", extra: "  required_when_no_variant: [missing]\n", want: "required_when_no_variant"},
		{name: "unknown default required-any field", field: "    - {name: title}\n", extra: "  required_any_when_no_variant:\n    - [title, missing]\n", want: "required_any_when_no_variant"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := baseLayoutYAML + "fields:\n  optional:\n" + tt.field + tt.extra
			_, err := parseLayoutSpec([]byte(yaml))
			if tt.want == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want category %q", err, tt.want)
			}
		})
	}
}

func TestValidatorIndependentNegativeBoundaries(t *testing.T) {
	tests := []struct {
		name, markdown, category string
		spec                     *LayoutSpec
	}{
		{name: "missing required field", markdown: ":::demo\n:::\n", category: "required field", spec: &LayoutSpec{Name: "demo", BodyFormat: BodyFormatFields, Fields: &FieldsSpec{Required: []FieldSpec{{Name: "title"}}}}},
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
	return c.ValidateWitness(WitnessContract{
		Module: module, Variant: variant, Example: example, AssertContains: assertion,
	})
}

const baseLayoutYAML = `schema_version: "1"
name: demo
body_format: fields
version: "1.0.0"
category: opening
serves: [attention]
metadata:
  author: test
  provenance: test-fixture
`

func baseVariantYAML(variants string) string {
	return baseLayoutYAML + "variants:\n" + variants
}
