package layoutcatalog

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidServesContainsFourValues(t *testing.T) {
	want := []string{"attention", "readability", "memorability", "conversion"}
	for _, v := range want {
		if !ValidServes[v] {
			t.Errorf("expected %q to be a valid serve, missing", v)
		}
	}
	if len(ValidServes) != 4 {
		t.Errorf("ValidServes should contain exactly 4 values, got %d", len(ValidServes))
	}
}

func TestSchemaVersionConstant(t *testing.T) {
	if SchemaVersion != "1" {
		t.Errorf("SchemaVersion = %q, want %q", SchemaVersion, "1")
	}
}

func TestValidBodyFormats(t *testing.T) {
	want := []string{
		BodyFormatFields,
		BodyFormatRows,
		BodyFormatJSONObject,
		BodyFormatJSONArray,
		BodyFormatMarkdownImages,
		BodyFormatMarkdownFields,
		BodyFormatSplit,
		BodyFormatLines,
		BodyFormatDialogue,
	}
	for _, v := range want {
		if !ValidBodyFormats[v] {
			t.Errorf("expected %q to be a valid body_format, missing", v)
		}
	}
	if len(ValidBodyFormats) != len(want) {
		t.Errorf("ValidBodyFormats should contain exactly %d values, got %d", len(want), len(ValidBodyFormats))
	}
}

func TestFieldShapeSpecJSONOmitsUnsetExtensionConstraints(t *testing.T) {
	encoded, err := json.Marshal(FieldShapeSpec{Field: "items", Separator: "|", MinParts: 2})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"MaxParts", "MaxOccurrences", "PartRules", "ItemMaxParts"} {
		if strings.Contains(string(encoded), field) {
			t.Fatalf("unset extension constraint %s leaked into JSON: %s", field, encoded)
		}
	}
}

func TestFieldShapeSpecJSONKeepsPartRulesInternal(t *testing.T) {
	encoded, err := json.Marshal(FieldShapeSpec{
		Field:     "point",
		Separator: "|",
		MinParts:  2,
		PartRules: []FieldPartRuleSpec{{MinParts: 4, RequiredPositions: []int{1, 4}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "PartRules") || strings.Contains(string(encoded), "RequiredPositions") {
		t.Fatalf("internal compatibility rules leaked into discovery JSON: %s", encoded)
	}
}

func TestSchemaExtensionsExposeAgentInputAndFieldDefaults(t *testing.T) {
	spec := LayoutSpec{
		InputPositions: []AgentInputPosition{InputBodyKV, InputHeaderAttrs},
		Fields: &FieldsSpec{Optional: []FieldSpec{{
			Name: "variant", Default: "marker", MinRunes: 1, MaxRunes: 4, AppliesTo: []string{"marker"},
		}}},
		Variants: []VariantSpec{{Name: "marker", Defaults: map[string]string{"symbol": "diamond-outline"}}},
	}
	if got := spec.Fields.Optional[0].Default; got != "marker" {
		t.Fatalf("field default = %q, want marker", got)
	}
	encoded, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"input_positions":["body-kv","header-attrs"]`) {
		t.Fatalf("input positions missing from JSON: %s", encoded)
	}
	if !strings.Contains(string(encoded), `"defaults":{"symbol":"diamond-outline"}`) {
		t.Fatalf("variant defaults missing from JSON: %s", encoded)
	}
}
