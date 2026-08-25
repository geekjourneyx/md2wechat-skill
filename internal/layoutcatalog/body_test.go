package layoutcatalog

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestValidateRowsRejectsTooFewColumns(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	r := c.Validate(":::steps\n01 | only-two\n:::\n")
	if len(r.Errors) == 0 || !strings.Contains(r.Errors[0].Message, "at least 3 columns") {
		t.Fatalf("got %+v", r.Errors)
	}
}

func TestValidateBlockBodyMatrix(t *testing.T) {
	field := func(name string) FieldSpec { return FieldSpec{Name: name} }
	tests := []struct {
		name    string
		spec    *LayoutSpec
		body    []string
		wantErr string
	}{
		{
			name: "markdown images accepts image",
			spec: &LayoutSpec{BodyFormat: BodyFormatMarkdownImages, Body: &BodySpec{MinImages: 1, MaxImages: 2}},
			body: []string{"![cover](https://example.com/cover.png)"},
		},
		{
			name: "markdown images rejects zero images",
			spec: &LayoutSpec{BodyFormat: BodyFormatMarkdownImages, Body: &BodySpec{MinImages: 1}},
			body: []string{"plain text"}, wantErr: "at least 1 image",
		},
		{
			name: "markdown images rejects overflow",
			spec: &LayoutSpec{BodyFormat: BodyFormatMarkdownImages, Body: &BodySpec{MaxImages: 1}},
			body: []string{"![one](https://example.com/1.png)", "![two](<https://example.com/2.png>)"}, wantErr: "at most 1 image",
		},
		{
			name: "markdown fields accepts paired image and caption",
			spec: &LayoutSpec{
				BodyFormat: BodyFormatMarkdownFields,
				Fields:     &FieldsSpec{Optional: []FieldSpec{field("image"), field("caption")}},
				Body:       &BodySpec{RequiredPairs: [][2]string{{"image", "caption"}}},
			},
			body: []string{"image: ![alt](https://example.com/a.png)", "caption: A caption"},
		},
		{
			name: "markdown fields rejects missing image",
			spec: &LayoutSpec{
				BodyFormat: BodyFormatMarkdownFields,
				Fields:     &FieldsSpec{Optional: []FieldSpec{field("image"), field("caption")}},
				Body:       &BodySpec{RequiredPairs: [][2]string{{"image", "caption"}}},
			},
			body: []string{"caption: A caption"}, wantErr: "image",
		},
		{
			name: "markdown fields rejects missing caption",
			spec: &LayoutSpec{
				BodyFormat: BodyFormatMarkdownFields,
				Fields:     &FieldsSpec{Optional: []FieldSpec{field("image"), field("caption")}},
				Body:       &BodySpec{RequiredPairs: [][2]string{{"image", "caption"}}},
			},
			body: []string{"image: ![alt](https://example.com/a.png)"}, wantErr: "caption",
		},
		{
			name: "grouped markdown fields accepts complete group",
			spec: &LayoutSpec{
				BodyFormat: BodyFormatMarkdownFields,
				Fields:     &FieldsSpec{Optional: []FieldSpec{field("step"), field("desc")}},
				Body:       &BodySpec{Group: &FieldGroupSpec{Start: "step", Required: []string{"step", "desc"}, Min: 1}},
			},
			body: []string{"step: One", "desc: Do it"},
		},
		{
			name: "grouped markdown fields rejects incomplete group",
			spec: &LayoutSpec{
				BodyFormat: BodyFormatMarkdownFields,
				Fields:     &FieldsSpec{Optional: []FieldSpec{field("step"), field("desc")}},
				Body:       &BodySpec{Group: &FieldGroupSpec{Start: "step", Required: []string{"step", "desc"}, Min: 1}},
			},
			body: []string{"step: One"}, wantErr: "desc",
		},
		{
			name: "grouped markdown fields rejects no complete groups",
			spec: &LayoutSpec{
				BodyFormat: BodyFormatMarkdownFields,
				Fields:     &FieldsSpec{Optional: []FieldSpec{field("step"), field("desc")}},
				Body:       &BodySpec{Group: &FieldGroupSpec{Start: "step", Required: []string{"step", "desc"}, Min: 1}},
			},
			body: []string{"unrelated"}, wantErr: "at least 1 complete group",
		},
		{
			name: "split accepts two nonempty sides",
			spec: &LayoutSpec{BodyFormat: BodyFormatSplit, Body: &BodySpec{Separator: "---"}},
			body: []string{"Left", "---", "Right"},
		},
		{
			name: "split rejects missing divider",
			spec: &LayoutSpec{BodyFormat: BodyFormatSplit, Body: &BodySpec{Separator: "---"}},
			body: []string{"Left", "Right"}, wantErr: "standalone separator",
		},
		{
			name: "split rejects edge divider",
			spec: &LayoutSpec{BodyFormat: BodyFormatSplit, Body: &BodySpec{Separator: "---"}},
			body: []string{"---", "Right"}, wantErr: "two non-empty sides",
		},
		{
			name: "lines accepts arrow separated items",
			spec: &LayoutSpec{BodyFormat: BodyFormatLines, Body: &BodySpec{MinItems: 3, Separator: "→"}},
			body: []string{"Plan → Build → Ship"},
		},
		{
			name: "lines accepts bullet items",
			spec: &LayoutSpec{BodyFormat: BodyFormatLines, Body: &BodySpec{MinItems: 2, AllowedPrefixes: []string{"-"}}},
			body: []string{"- Plan", "- Build"},
		},
		{
			name: "lines accepts newline items",
			spec: &LayoutSpec{BodyFormat: BodyFormatLines, Body: &BodySpec{MinItems: 2}},
			body: []string{"Plan", "Build"},
		},
		{
			name: "lines rejects too few items",
			spec: &LayoutSpec{BodyFormat: BodyFormatLines, Body: &BodySpec{MinItems: 2}},
			body: []string{"Only one"}, wantErr: "at least 2 items",
		},
		{
			name: "dialogue accepts configured pair",
			spec: &LayoutSpec{BodyFormat: BodyFormatDialogue, Body: &BodySpec{AllowedPrefixes: []string{"Q", "A"}, RequiredPairs: [][2]string{{"Q", "A"}}}},
			body: []string{"Q: Why?", "A: Because."},
		},
		{
			name: "dialogue rejects full-width delimiter for configured pair",
			spec: &LayoutSpec{BodyFormat: BodyFormatDialogue, Body: &BodySpec{AllowedPrefixes: []string{"Q:", "A:"}, RequiredPairs: [][2]string{{"Q", "A"}}}},
			body: []string{"Q：Why?", "A：Because."}, wantErr: "ASCII colon",
		},
		{
			name: "dialogue accepts configured alternate pair with ascii colon",
			spec: &LayoutSpec{BodyFormat: BodyFormatDialogue, Body: &BodySpec{AllowedPrefixes: []string{"U", "E"}, RequiredPairs: [][2]string{{"U", "E"}}}},
			body: []string{"U: Question", "E: Answer"},
		},
		{
			name: "dialogue rejects configured single side",
			spec: &LayoutSpec{BodyFormat: BodyFormatDialogue, Body: &BodySpec{AllowedPrefixes: []string{"U", "E"}, RequiredPairs: [][2]string{{"U", "E"}}}},
			body: []string{"U: Question"}, wantErr: "E",
		},
		{
			name: "dialogue accepts named speakers when enabled",
			spec: &LayoutSpec{BodyFormat: BodyFormatDialogue, Body: &BodySpec{AllowNamedSpeakers: true, MinItems: 2}},
			body: []string{"读者：为什么？", "作者：因为。"},
		},
		{
			name: "dialogue rejects named speakers with ascii colon",
			spec: &LayoutSpec{BodyFormat: BodyFormatDialogue, Body: &BodySpec{AllowNamedSpeakers: true, MinItems: 2}},
			body: []string{"Reader: Why?", "Author: Because."}, wantErr: "full-width colon",
		},
		{
			name: "dialogue rejects https urls as named speakers",
			spec: &LayoutSpec{BodyFormat: BodyFormatDialogue, Body: &BodySpec{AllowNamedSpeakers: true, MinItems: 2}},
			body: []string{"https://example.com/one", "https://example.com/two"}, wantErr: "full-width colon",
		},
		{
			name: "dialogue rejects empty prefix",
			spec: &LayoutSpec{BodyFormat: BodyFormatDialogue, Body: &BodySpec{AllowNamedSpeakers: true}},
			body: []string{"：没有说话人"}, wantErr: "speaker prefix",
		},
		{
			name: "rows accepts URL colon as row content",
			spec: &LayoutSpec{BodyFormat: BodyFormatRows, Rows: &RowsSpec{Delimiter: "|", MinColumns: 3}},
			body: []string{"https://example.com | Title | Description"},
		},
		{
			name: "rows rejects empty required cell",
			spec: &LayoutSpec{BodyFormat: BodyFormatRows, Rows: &RowsSpec{Delimiter: "|", MinColumns: 3}},
			body: []string{"01 | | Description"}, wantErr: "must not be empty",
		},
		{
			name: "required any accepts one field per group",
			spec: &LayoutSpec{
				BodyFormat: BodyFormatFields,
				Fields:     &FieldsSpec{Optional: []FieldSpec{field("title"), field("body"), field("note")}, RequiredAny: [][]string{{"title", "body"}, {"note"}}},
			},
			body: []string{"body: Main", "note: Detail"},
		},
		{
			name: "required any rejects missing group",
			spec: &LayoutSpec{
				BodyFormat: BodyFormatFields,
				Fields:     &FieldsSpec{Optional: []FieldSpec{field("title"), field("body"), field("note")}, RequiredAny: [][]string{{"title", "body"}, {"note"}}},
			},
			body: []string{"body: Main"}, wantErr: "one of [note]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := validateBlockBody(tt.spec, tt.body)
			if tt.wantErr == "" {
				if len(issues) != 0 {
					t.Fatalf("validateBlockBody() issues = %+v", issues)
				}
				return
			}
			if len(issues) == 0 || !strings.Contains(issues[0].message, tt.wantErr) {
				t.Fatalf("validateBlockBody() issues = %+v, want error containing %q", issues, tt.wantErr)
			}
		})
	}
}

func TestFieldShapeValidationMatrix(t *testing.T) {
	field := func(name string) FieldSpec { return FieldSpec{Name: name} }
	tests := []struct {
		name    string
		values  []string
		shapes  []FieldShapeSpec
		wantErr bool
	}{
		{name: "pipe accepts two", values: []string{"one | two"}, shapes: []FieldShapeSpec{{Field: "items", Separator: "|", MinParts: 2}}},
		{name: "pipe rejects one", values: []string{"one"}, shapes: []FieldShapeSpec{{Field: "items", Separator: "|", MinParts: 2}}, wantErr: true},
		{name: "plus ignores empty parts", values: []string{"one + "}, shapes: []FieldShapeSpec{{Field: "items", Separator: "+", MinParts: 2}}, wantErr: true},
		{name: "validates every repeated value", values: []string{"01 | 20 | 30 | title", "broken"}, shapes: []FieldShapeSpec{{Field: "point", Separator: "|", MinParts: 4}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldName := "items"
			if tt.shapes[0].Field == "point" {
				fieldName = "point"
			}
			body := make([]string, 0, len(tt.values))
			for _, value := range tt.values {
				body = append(body, fieldName+": "+value)
			}
			spec := &LayoutSpec{BodyFormat: BodyFormatFields, Fields: &FieldsSpec{Optional: []FieldSpec{field(fieldName)}, Shapes: tt.shapes}}
			issues := validateBlockBody(spec, body)
			if got := len(issues) > 0; got != tt.wantErr {
				t.Fatalf("issues = %+v, wantErr %v", issues, tt.wantErr)
			}
		})
	}
}

func TestQuestionPairAndJSONArrayItemBoundaries(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, markdown string
		accepted       bool
	}{
		{name: "ordered two pairs", markdown: ":::question\nQ: One?\nA: One.\nQ: Two?\nA: Two.\n:::\n", accepted: true},
		{name: "answer before question", markdown: ":::question\nA: Answer.\nQ: Question?\n:::\n"},
		{name: "two questions before answer", markdown: ":::question\nQ: One?\nQ: Two?\nA: Answer.\n:::\n"},
		{name: "trailing answer", markdown: ":::question\nQ: One?\nA: One.\nA: Extra.\n:::\n"},
		{name: "full-width configured prefixes", markdown: ":::question\nQ：One?\nA：One.\n:::\n"},
		{name: "complete JSON items", markdown: ":::question\n[{\"q\":\"One?\",\"a\":\"One.\"},{\"q\":\"Two?\",\"a\":\"Two.\"}]\n:::\n", accepted: true},
		{name: "split JSON fields", markdown: ":::question\n[{\"q\":\"One?\"},{\"a\":\"One.\"}]\n:::\n"},
		{name: "second JSON item incomplete", markdown: ":::question\n[{\"q\":\"One?\",\"a\":\"One.\"},{\"q\":\"Two?\"}]\n:::\n"},
		{name: "numeric JSON question", markdown: ":::question\n[{\"q\":42,\"a\":\"Answer.\"}]\n:::\n"},
		{name: "boolean JSON answer", markdown: ":::question\n[{\"q\":\"Question?\",\"a\":true}]\n:::\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := c.Validate(tt.markdown)
			if got := len(report.Errors) == 0; got != tt.accepted {
				t.Fatalf("accepted = %v, want %v; errors=%+v", got, tt.accepted, report.Errors)
			}
		})
	}
	report := c.Validate(":::question\n[{\"q\":\"One?\"},{\"a\":\"One.\"}]\n:::\n")
	if len(report.Errors) == 0 || !strings.Contains(report.Errors[0].Message, "item 1") {
		t.Fatalf("split JSON error = %+v, want per-item error", report.Errors)
	}
	comparison := `:::comparison-table
{"left":{"title":"A","items":["one","two"]},"right":{"title":"B","items":["three","four"]}}
:::
`
	if report := c.Validate(comparison); len(report.Errors) != 0 {
		t.Fatalf("array-valued compatibility fields failed: %+v", report.Errors)
	}
}

func TestParseLayoutSpecBodySchemaMatrix(t *testing.T) {
	base := `
schema_version: "1"
name: fixture
body_format: %s
%s
version: "1.0.0"
category: body
serves: [readability]
%s
metadata:
  author: test
  provenance: test
`
	tests := []struct {
		name       string
		format     string
		extra      string
		definition string
		wantErr    string
	}{
		{
			name: "accepts finite body schema", format: BodyFormatLines,
			extra: "body:\n  min_items: 2\n  separator: \"→\"",
		},
		{
			name: "rejects unknown compatible format", format: BodyFormatDialogue,
			extra: "compatible_body_formats: [made_up]", wantErr: "compatible body_format",
		},
		{
			name: "rows may declare metadata fields", format: BodyFormatRows,
			definition: "fields:\n  optional:\n    - name: title\nrows:\n  delimiter: \"|\"\n  min_columns: 2",
		},
		{
			name: "compatible rows requires row schema", format: BodyFormatFields,
			extra: "compatible_body_formats: [rows]", wantErr: "requires rows",
		},
		{
			name: "rows schema requires accepted rows format", format: BodyFormatFields,
			definition: "rows:\n  delimiter: \"|\"\n  min_columns: 2", wantErr: "rows requires",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := fmt.Sprintf(base, tt.format, tt.extra, tt.definition)
			_, err := parseLayoutSpec([]byte(yaml))
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("parseLayoutSpec() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("parseLayoutSpec() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestFieldsBodyRejectsUnknownAndMalformedLines(t *testing.T) {
	spec := &LayoutSpec{
		Name: "demo", BodyFormat: BodyFormatFields,
		Fields: &FieldsSpec{Required: []FieldSpec{{Name: "title"}}},
	}
	for _, body := range [][]string{
		{"title: Valid", "titlle: typo"},
		{"title: Valid", "arbitrary prose"},
	} {
		if issues := validateBlockBody(spec, body); len(issues) == 0 {
			t.Fatalf("validateBlockBody(%q) accepted malformed fields body", body)
		}
	}
}

func TestRowsBodyEnforcesDeclaredColumnSchema(t *testing.T) {
	spec := &LayoutSpec{
		Name: "metrics", BodyFormat: BodyFormatRows,
		Rows: &RowsSpec{
			Delimiter: "|", MinColumns: 2,
			Schema: []FieldSpec{
				{Name: "label"}, {Name: "value"},
				{Name: "style", Enum: []string{"accent", "default"}},
			},
		},
	}
	tests := []struct {
		name string
		body []string
	}{
		{name: "invalid enum", body: []string{"Readers | 38 | bogus"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if issues := validateBlockBody(spec, tt.body); len(issues) == 0 {
				t.Fatalf("validateBlockBody(%q) accepted invalid row", tt.body)
			}
		})
	}
}

func TestFieldRuneBoundsAndAppliesToAreGeneric(t *testing.T) {
	spec := &LayoutSpec{
		Name: "schema-fixture", BodyFormat: BodyFormatFields,
		Fields: &FieldsSpec{Optional: []FieldSpec{
			{Name: "variant", Default: "numbered"},
			{Name: "index", MinRunes: 1, MaxRunes: 4, AppliesTo: []string{"numbered"}},
		}},
		Variants: []VariantSpec{{Name: "numbered"}, {Name: "plain"}},
	}
	tests := []struct {
		name, body string
		accepted   bool
	}{
		{name: "one unicode rune", body: "index: 壹", accepted: true},
		{name: "four unicode runes", body: "index: 一二三四", accepted: true},
		{name: "blank index", body: "index:   "},
		{name: "five unicode runes", body: "index: 一二三四五"},
		{name: "field on another variant", body: "variant: plain\nindex: 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := validateBlockBody(spec, strings.Split(tt.body, "\n"))
			if got := len(issues) == 0; got != tt.accepted {
				t.Fatalf("issues = %+v, accepted = %v, want %v", issues, got, tt.accepted)
			}
		})
	}
}

func TestRowsBodyRejectsMaxColumns(t *testing.T) {
	spec := &LayoutSpec{Name: "row-fixture", BodyFormat: BodyFormatRows, Rows: &RowsSpec{
		Delimiter: "|", MinColumns: 2, MaxColumns: 2,
	}}
	if issues := validateBlockBody(spec, []string{"one|two|three"}); len(issues) == 0 {
		t.Fatal("row with an extra cell was accepted")
	}
}

func TestSeparatorListRejectsItemMaxParts(t *testing.T) {
	spec := &LayoutSpec{Name: "list-fixture", BodyFormat: BodyFormatFields, Fields: &FieldsSpec{
		Optional: []FieldSpec{{Name: "items"}},
		Shapes:   []FieldShapeSpec{{Field: "items", Separator: ",", MinParts: 2, ItemSeparator: "|", ItemMinParts: 2, ItemMaxParts: 2}},
	}}
	if issues := validateBlockBody(spec, []string{"items: a|b|overflow,c|d"}); len(issues) == 0 {
		t.Fatal("separator list with too many item parts was accepted")
	}
}

func TestParseLayoutSpecFieldShapeAndOutputOrderSchema(t *testing.T) {
	tests := []struct {
		name, fields, want string
	}{
		{name: "valid", fields: "  output_order: [title, items]\n  shapes:\n    - {field: items, separator: '|', min_parts: 2}\n"},
		{name: "valid max occurrences and part rules", fields: "  shapes:\n    - field: items\n      separator: '|'\n      min_parts: 2\n      max_occurrences: 3\n      part_rules:\n        - {max_parts: 3, required_positions: [1, 2]}\n        - {min_parts: 4, required_positions: [1, 4]}\n"},
		{name: "unknown output field", fields: "  output_order: [missing]\n", want: "output_order field"},
		{name: "duplicate output field", fields: "  output_order: [title, title]\n", want: "duplicate output_order"},
		{name: "unknown shape field", fields: "  shapes:\n    - {field: missing, separator: '|', min_parts: 2}\n", want: "shape field"},
		{name: "empty separator", fields: "  shapes:\n    - {field: items, separator: '', min_parts: 2}\n", want: "separator"},
		{name: "small minimum", fields: "  shapes:\n    - {field: items, separator: '|', min_parts: 1}\n", want: "greater than 1"},
		{name: "negative max occurrences", fields: "  shapes:\n    - {field: items, separator: '|', min_parts: 2, max_occurrences: -1}\n", want: "max_occurrences"},
		{name: "part rule missing bounds", fields: "  shapes:\n    - field: items\n      separator: '|'\n      min_parts: 2\n      part_rules:\n        - {required_positions: [1]}\n", want: "requires min_parts or max_parts"},
		{name: "part rule negative bound", fields: "  shapes:\n    - field: items\n      separator: '|'\n      min_parts: 2\n      part_rules:\n        - {min_parts: -1, required_positions: [1]}\n", want: "bounds must be nonnegative"},
		{name: "part rule inverted bounds", fields: "  shapes:\n    - field: items\n      separator: '|'\n      min_parts: 2\n      part_rules:\n        - {min_parts: 4, max_parts: 3, required_positions: [1]}\n", want: "must not exceed"},
		{name: "part rule missing positions", fields: "  shapes:\n    - field: items\n      separator: '|'\n      min_parts: 2\n      part_rules:\n        - {max_parts: 3}\n", want: "requires required_positions"},
		{name: "part rule nonpositive position", fields: "  shapes:\n    - field: items\n      separator: '|'\n      min_parts: 2\n      part_rules:\n        - {max_parts: 3, required_positions: [0]}\n", want: "positions must be positive"},
		{name: "part rule position exceeds maximum", fields: "  shapes:\n    - field: items\n      separator: '|'\n      min_parts: 2\n      part_rules:\n        - {max_parts: 3, required_positions: [4]}\n", want: "exceeds max_parts"},
		{name: "part rule duplicate position", fields: "  shapes:\n    - field: items\n      separator: '|'\n      min_parts: 2\n      part_rules:\n        - {max_parts: 3, required_positions: [1, 1]}\n", want: "is duplicated"},
		{name: "item shape missing separator", fields: "  shapes:\n    - {field: items, separator: '|', min_parts: 2, item_min_parts: 2}\n", want: "item_separator"},
		{name: "item shape small minimum", fields: "  shapes:\n    - {field: items, separator: '|', min_parts: 2, item_separator: ':', item_min_parts: 1}\n", want: "item_min_parts"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := baseLayoutYAML + "fields:\n  optional:\n    - name: title\n    - name: items\n" + tt.fields
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

func TestDefaultOnlyRequirementsComposeWithVariants(t *testing.T) {
	spec := &LayoutSpec{
		Name: "demo", BodyFormat: BodyFormatFields,
		Fields: &FieldsSpec{
			Required:              []FieldSpec{{Name: "title"}},
			Optional:              []FieldSpec{{Name: "variant"}, {Name: "density"}},
			RequiredWhenNoVariant: []string{"density"},
		},
		Variants: []VariantSpec{{Name: "compact", Required: []string{"density"}}},
	}
	c := NewCatalog()
	c.modules[spec.Name] = spec
	tests := []struct {
		name, markdown string
		accepted       bool
	}{
		{name: "default complete", markdown: ":::demo\ntitle: T\ndensity: normal\n:::\n", accepted: true},
		{name: "default missing density", markdown: ":::demo\ntitle: T\n:::\n"},
		{name: "variant complete", markdown: ":::demo\nvariant: compact\ntitle: T\ndensity: compact\n:::\n", accepted: true},
		{name: "variant cannot skip base title", markdown: ":::demo\nvariant: compact\ndensity: compact\n:::\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := c.Validate(tt.markdown)
			if got := len(report.Errors) == 0; got != tt.accepted {
				t.Fatalf("accepted = %v, want %v; errors=%+v", got, tt.accepted, report.Errors)
			}
		})
	}
}

func TestSelectorEnumWithoutVariantsStillValidates(t *testing.T) {
	c := NewCatalog()
	c.modules["demo"] = &LayoutSpec{Name: "demo", BodyFormat: BodyFormatFields, Fields: &FieldsSpec{Optional: []FieldSpec{{Name: "type", Enum: []string{"known"}}}}}
	if report := c.Validate(":::demo\ntype: unknown\n:::\n"); len(report.Errors) == 0 || report.Errors[0].Field != "type" {
		t.Fatalf("report = %+v, want type enum error", report)
	}
}

func conditionalSchemaYAML(fields, body string) []byte {
	return []byte(fmt.Sprintf(`
schema_version: "1"
name: conditional
body_format: markdown_fields
version: "1.0.0"
category: body
serves: [readability]
%s
%s
metadata:
  author: test
  provenance: test
`, fields, body))
}

func TestParseLayoutSpecRejectsInvalidConditionalBodySchema(t *testing.T) {
	declaredFields := `fields:
  optional:
    - name: image
    - name: caption
    - name: step
    - name: desc`
	tests := []struct {
		name    string
		fields  string
		body    string
		wantErr string
	}{
		{name: "valid conditional schema", fields: declaredFields, body: `body:
  min_images: 1
  max_images: 2
  min_items: 0
  required_pairs:
    - [image, caption]
  group:
    start: step
    required: [step, desc]
    min: 1`},
		{name: "zero max is unbounded", fields: declaredFields, body: "body:\n  min_images: 2\n  max_images: 0"},
		{name: "negative min images", fields: declaredFields, body: "body:\n  min_images: -1", wantErr: "min_images must be nonnegative"},
		{name: "negative max images", fields: declaredFields, body: "body:\n  max_images: -1", wantErr: "max_images must be nonnegative"},
		{name: "negative min items", fields: declaredFields, body: "body:\n  min_items: -1", wantErr: "min_items must be nonnegative"},
		{name: "max images below min", fields: declaredFields, body: "body:\n  min_images: 2\n  max_images: 1", wantErr: "max_images must be at least min_images"},
		{name: "empty required any group", fields: declaredFields + "\n  required_any:\n    - []", body: "", wantErr: "required_any group must not be empty"},
		{name: "unknown required any field", fields: declaredFields + "\n  required_any:\n    - [image, missing]", body: "", wantErr: "required_any field \"missing\" is not declared"},
		{name: "duplicate required any field", fields: declaredFields + "\n  required_any:\n    - [image, image]", body: "", wantErr: "duplicate required_any field \"image\""},
		{name: "duplicate required any group", fields: declaredFields + "\n  required_any:\n    - [image, caption]\n    - [caption, image]", body: "", wantErr: "duplicate required_any group"},
		{name: "duplicate field declaration", fields: declaredFields + "\n    - name: image", body: "", wantErr: "duplicate field \"image\""},
		{name: "empty required pair field", fields: declaredFields, body: "body:\n  required_pairs:\n    - [image, \"\"]", wantErr: "required_pairs fields must not be empty"},
		{name: "unknown required pair field", fields: declaredFields, body: "body:\n  required_pairs:\n    - [image, missing]", wantErr: "required_pairs field \"missing\" is not declared"},
		{name: "same required pair field", fields: declaredFields, body: "body:\n  required_pairs:\n    - [image, image]", wantErr: "required_pairs fields must be distinct"},
		{name: "duplicate required pair", fields: declaredFields, body: "body:\n  required_pairs:\n    - [image, caption]\n    - [image, caption]", wantErr: "duplicate required_pairs pair"},
		{name: "reverse duplicate required pair", fields: declaredFields, body: "body:\n  required_pairs:\n    - [image, caption]\n    - [caption, image]", wantErr: "duplicate required_pairs pair"},
		{name: "empty group start", fields: declaredFields, body: "body:\n  group:\n    start: \"\"\n    required: [step]\n    min: 1", wantErr: "group.start must not be empty"},
		{name: "unknown group start", fields: declaredFields, body: "body:\n  group:\n    start: missing\n    required: [step]\n    min: 1", wantErr: "group.start field \"missing\" is not declared"},
		{name: "empty group required", fields: declaredFields, body: "body:\n  group:\n    start: step\n    required: []\n    min: 1", wantErr: "group.required must not be empty"},
		{name: "empty group required field", fields: declaredFields, body: "body:\n  group:\n    start: step\n    required: [\"\"]\n    min: 1", wantErr: "group.required fields must not be empty"},
		{name: "unknown group required field", fields: declaredFields, body: "body:\n  group:\n    start: step\n    required: [missing]\n    min: 1", wantErr: "group.required field \"missing\" is not declared"},
		{name: "duplicate group required field", fields: declaredFields, body: "body:\n  group:\n    start: step\n    required: [desc, desc]\n    min: 1", wantErr: "duplicate group.required field \"desc\""},
		{name: "negative group min", fields: declaredFields, body: "body:\n  group:\n    start: step\n    required: [step]\n    min: -1", wantErr: "group.min must be nonnegative"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseLayoutSpec(conditionalSchemaYAML(tt.fields, tt.body))
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("parseLayoutSpec() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("parseLayoutSpec() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseLayoutSpecDialoguePairsUseAllowedPrefixNamespaceRoundTrip(t *testing.T) {
	input := []byte(`
schema_version: "1"
name: dialogue-question
body_format: dialogue
compatible_body_formats: [json_array]
version: "1.0.0"
category: body
serves: [readability]
fields:
  required:
    - name: q
    - name: a
body:
  min_items: 2
  allowed_prefixes: ["Q:", "A:"]
  required_pairs:
    - [Q, A]
metadata:
  author: test
  provenance: test
`)

	spec, err := parseLayoutSpec(input)
	if err != nil {
		t.Fatalf("parseLayoutSpec() error = %v", err)
	}
	encoded, err := yaml.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	roundTripped, err := parseLayoutSpec(encoded)
	if err != nil {
		t.Fatalf("parseLayoutSpec(round trip) error = %v", err)
	}
	if issues := validateBlockBody(roundTripped, []string{"Q: Why?", "A: Because."}); len(issues) != 0 {
		t.Fatalf("dialogue body issues = %+v", issues)
	}
}

func TestParseLayoutSpecDialoguePairsRejectUndeclaredPrefix(t *testing.T) {
	input := []byte(`
schema_version: "1"
name: invalid-dialogue
body_format: dialogue
compatible_body_formats: [json_array]
version: "1.0.0"
category: body
serves: [readability]
fields:
  required:
    - name: q
    - name: a
body:
  allowed_prefixes: ["Q:", "A:"]
  required_pairs:
    - [Q, E]
metadata:
  author: test
  provenance: test
`)

	_, err := parseLayoutSpec(input)
	if err == nil || !strings.Contains(err.Error(), `required_pairs dialogue prefix "E" is not configured`) {
		t.Fatalf("parseLayoutSpec() error = %v", err)
	}
}

func TestParseLayoutSpecPairsValidateEveryApplicableNamespace(t *testing.T) {
	input := []byte(`
schema_version: "1"
name: mixed-dialogue
body_format: dialogue
compatible_body_formats: [markdown_fields]
version: "1.0.0"
category: body
serves: [readability]
fields:
  required:
    - name: q
    - name: a
body:
  allowed_prefixes: ["Q:", "A:"]
  required_pairs:
    - [Q, A]
metadata:
  author: test
  provenance: test
`)

	_, err := parseLayoutSpec(input)
	if err == nil || !strings.Contains(err.Error(), `required_pairs field "Q" is not declared`) {
		t.Fatalf("parseLayoutSpec() error = %v", err)
	}
}

func dialogueJSONSpec() *LayoutSpec {
	return &LayoutSpec{
		Name:                  "question-next",
		BodyFormat:            BodyFormatDialogue,
		CompatibleBodyFormats: []string{BodyFormatJSONArray},
		Fields: &FieldsSpec{Required: []FieldSpec{
			{Name: "q"},
			{Name: "a"},
		}},
		Body: &BodySpec{
			AllowedPrefixes: []string{"Q", "A"},
			RequiredPairs:   [][2]string{{"Q", "A"}},
			MinItems:        2,
		},
	}
}

func TestRenderPreservesMissingRequiredFieldSentinel(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	_, err := c.Render("hero", map[string]any{"eyebrow": "Context"})
	if !errors.Is(err, ErrMissingRequiredField) {
		t.Fatalf("Render() error = %v, want ErrMissingRequiredField", err)
	}
}

func TestValidateBlockBodyPreservesEmptyFormatAsFields(t *testing.T) {
	spec := &LayoutSpec{Fields: &FieldsSpec{Required: []FieldSpec{{Name: "title"}}}}
	if issues := validateBlockBody(spec, []string{"title: Legacy"}); len(issues) != 0 {
		t.Fatalf("validateBlockBody() issues = %+v", issues)
	}
}

func TestValidateCompatibilityFormats(t *testing.T) {
	c := NewCatalog()
	c.modules["question-next"] = dialogueJSONSpec()

	tests := []struct {
		name        string
		body        string
		wantErr     bool
		wantMessage []string
	}{
		{name: "primary dialogue", body: "Q: Why?\nA: Because."},
		{name: "legacy JSON", body: `[{"q":"Why?","a":"Because."}]`},
		{name: "unrelated text", body: "This is not a question body", wantErr: true, wantMessage: []string{BodyFormatDialogue, BodyFormatJSONArray}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := c.Validate(":::question-next\n" + tt.body + "\n:::\n")
			if !tt.wantErr {
				if len(report.Errors) != 0 {
					t.Fatalf("Validate() errors = %+v", report.Errors)
				}
				return
			}
			if len(report.Errors) == 0 {
				t.Fatal("Validate() unexpectedly accepted body")
			}
			for _, want := range tt.wantMessage {
				if !strings.Contains(report.Errors[0].Message, want) {
					t.Fatalf("error %q does not mention accepted format %q", report.Errors[0].Message, want)
				}
			}
		})
	}
}

func TestRenderSelectsCompatibleStructuredFormat(t *testing.T) {
	c := NewCatalog()
	c.modules["question-next"] = dialogueJSONSpec()

	out, err := c.Render("question-next", map[string]any{"q": "Why?", "a": "Because."})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `[{"a":"Because.","q":"Why?"}]`) {
		t.Fatalf("Render() = %q, want JSON fallback", out)
	}
	if report := c.Validate(out); len(report.Errors) != 0 {
		t.Fatalf("rendered output does not pass shared validation: %+v", report.Errors)
	}
}

func TestRenderSelectedFormatCannotPassViaLooseCompatibilityFormat(t *testing.T) {
	c := NewCatalog()
	c.modules["strict-json"] = &LayoutSpec{
		Name:                  "strict-json",
		BodyFormat:            BodyFormatJSONObject,
		CompatibleBodyFormats: []string{BodyFormatLines},
		Fields:                &FieldsSpec{Required: []FieldSpec{{Name: "title"}}},
		Body:                  &BodySpec{MinItems: 1},
	}

	_, err := c.Render("strict-json", map[string]any{})
	if !errors.Is(err, ErrMissingRequiredField) {
		t.Fatalf("Render() error = %v, want selected JSON missing-field error", err)
	}
}

func TestRenderCompatibleJSONMissingFieldPreservesSentinel(t *testing.T) {
	c := NewCatalog()
	c.modules["question-next"] = dialogueJSONSpec()

	_, err := c.Render("question-next", map[string]any{"q": "Why?"})
	if !errors.Is(err, ErrMissingRequiredField) {
		t.Fatalf("Render() error = %v, want ErrMissingRequiredField", err)
	}
}

func TestRenderRawBodyUsesPrimaryValidation(t *testing.T) {
	c := NewCatalog()
	c.modules["question-next"] = dialogueJSONSpec()

	out, err := c.Render("question-next", map[string]any{"body": "Q: Why?\nA: Because."})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Q: Why?\nA: Because.") {
		t.Fatalf("Render() = %q, want primary dialogue body", out)
	}
	if _, err := c.Render("question-next", map[string]any{"body": "Q: Why?"}); err == nil || !strings.Contains(err.Error(), "A") {
		t.Fatalf("Render() error = %v, want missing dialogue pair", err)
	}
}
