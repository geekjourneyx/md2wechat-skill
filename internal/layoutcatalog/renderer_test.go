package layoutcatalog

import (
	"errors"
	"strings"
	"testing"
)

func TestRenderBlockIncludesBraceParamsAndRawBody(t *testing.T) {
	c := NewCatalog()
	c.modules["gallery-grid"] = &LayoutSpec{
		Name:       "gallery-grid",
		BodyFormat: BodyFormatMarkdownImages,
		Lifecycle:  LifecycleRecommended,
		Opener: &OpenerSpec{
			ParamStyle: ParamStyleBraces,
			Params: []ParamSpec{
				{Name: "columns"},
				{Name: "variant"},
			},
		},
		Body: &BodySpec{MinImages: 1},
	}
	out, err := c.RenderBlock("gallery-grid", RenderInput{
		Params: map[string]string{"variant": "card", "columns": "3"},
		Body:   "![产品界面](https://example.com/a.jpg) | 移动端首页",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := ":::gallery-grid{columns=3 variant=card}\n![产品界面](https://example.com/a.jpg) | 移动端首页\n:::\n"
	if out != want {
		t.Fatalf("%q != %q", out, want)
	}
}

func TestRenderBlockIncludesCaption(t *testing.T) {
	c := NewCatalog()
	c.modules["gallery"] = &LayoutSpec{
		Name:       "gallery",
		BodyFormat: BodyFormatMarkdownImages,
		Opener:     &OpenerSpec{Caption: true},
		Body:       &BodySpec{MinImages: 1},
	}
	out, err := c.RenderBlock("gallery", RenderInput{
		Caption: "产品截图",
		Body:    "![首页](https://example.com/home.jpg)",
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := ":::gallery[产品截图]\n![首页](https://example.com/home.jpg)\n:::\n"; out != want {
		t.Fatalf("%q != %q", out, want)
	}
}

func TestRenderBlockIncludesTokenParam(t *testing.T) {
	c := NewCatalog()
	c.modules["callout"] = &LayoutSpec{
		Name:       "callout",
		BodyFormat: BodyFormatLines,
		Opener: &OpenerSpec{
			ParamStyle: ParamStyleToken,
			Params:     []ParamSpec{{Name: "tone", Enum: []string{"info", "warning"}}},
		},
		Body: &BodySpec{MinItems: 1},
	}
	out, err := c.RenderBlock("callout", RenderInput{
		Params: map[string]string{"tone": "warning"},
		Body:   "发布前请复核。",
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := ":::callout warning\n发布前请复核。\n:::\n"; out != want {
		t.Fatalf("%q != %q", out, want)
	}
}

func TestRenderBlockRequiresComplexBody(t *testing.T) {
	c := NewCatalog()
	c.modules["gallery"] = &LayoutSpec{
		Name:       "gallery",
		BodyFormat: BodyFormatMarkdownImages,
		Body:       &BodySpec{MinImages: 1},
	}
	_, err := c.RenderBlock("gallery", RenderInput{})
	if !errors.Is(err, ErrInvalidFieldValue) {
		t.Fatalf("RenderBlock() error = %v, want ErrInvalidFieldValue", err)
	}
}

func TestRenderBlockRejectsInvalidOpenerEnum(t *testing.T) {
	c := NewCatalog()
	c.modules["callout"] = &LayoutSpec{
		Name:       "callout",
		BodyFormat: BodyFormatLines,
		Opener: &OpenerSpec{
			ParamStyle: ParamStyleToken,
			Params:     []ParamSpec{{Name: "tone", Enum: []string{"info", "warning"}}},
		},
		Body: &BodySpec{MinItems: 1},
	}
	_, err := c.RenderBlock("callout", RenderInput{
		Params: map[string]string{"tone": "urgent"},
		Body:   "发布前请复核。",
	})
	if !errors.Is(err, ErrInvalidFieldValue) {
		t.Fatalf("RenderBlock() error = %v, want ErrInvalidFieldValue", err)
	}
}

func TestRenderBlockPreservesLegacyHeroRender(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	vars := map[string]any{
		"eyebrow": "深度观察",
		"title":   "公众号排版的真问题",
	}
	legacy, err := c.Render("hero", vars)
	if err != nil {
		t.Fatal(err)
	}
	block, err := c.RenderBlock("hero", RenderInput{Fields: vars})
	if err != nil {
		t.Fatal(err)
	}
	if legacy != block {
		t.Fatalf("Render() = %q, RenderBlock() = %q", legacy, block)
	}
	want := ":::hero\neyebrow: 深度观察\ntitle: 公众号排版的真问题\n:::\n"
	if legacy != want {
		t.Fatalf("Render() = %q, want unchanged hero output %q", legacy, want)
	}
}

func TestRenderBlockKeepsCanonicalSummarySyntaxSeparateFromExplicitCompatibilityInput(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	canonical, err := c.RenderBlock("summary", RenderInput{Fields: map[string]any{
		"variant": "three",
		"title":   "Three conclusions",
		"items":   "one | two | three",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(canonical, "variant: three") || strings.Contains(canonical, "variant: points") {
		t.Fatalf("canonical summary render = %q", canonical)
	}

	legacy := ":::summary\nvariant: points\ntitle: Legacy\nitems: one | two | three\n:::\n"
	if report := c.Validate(legacy); len(report.Errors) != 0 {
		t.Fatalf("explicit compatibility input must validate: %+v", report.Errors)
	}
	raw, err := c.RenderBlock("summary", RenderInput{Body: "variant: points\ntitle: Legacy\nitems: one | two | three"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(raw, "variant: points") {
		t.Fatalf("explicit legacy raw input was not preserved: %q", raw)
	}
}

func TestRenderPreservesDeclaredOpenerOrderWhileRenderBlockSortsParams(t *testing.T) {
	c := NewCatalog()
	c.modules["ordered"] = &LayoutSpec{
		Name:       "ordered",
		BodyFormat: BodyFormatFields,
		Opener: &OpenerSpec{
			ParamStyle: ParamStyleBraces,
			Params: []ParamSpec{
				{Name: "zeta", Default: "z-default"},
				{Name: "alpha"},
			},
		},
		Fields: &FieldsSpec{Required: []FieldSpec{{Name: "title"}}},
	}

	legacy, err := c.Render("ordered", map[string]any{"alpha": "a", "title": "Title"})
	if err != nil {
		t.Fatal(err)
	}
	if want := ":::ordered{zeta=z-default alpha=a}\ntitle: Title\n:::\n"; legacy != want {
		t.Fatalf("Render() = %q, want %q", legacy, want)
	}

	block, err := c.RenderBlock("ordered", RenderInput{
		Fields: map[string]any{"title": "Title"},
		Params: map[string]string{"alpha": "a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := ":::ordered{alpha=a zeta=z-default}\ntitle: Title\n:::\n"; block != want {
		t.Fatalf("RenderBlock() = %q, want %q", block, want)
	}
}

func TestRenderBlockExplicitBodyTakesPrecedenceAcrossPrimaryFormats(t *testing.T) {
	tests := []struct {
		name   string
		format string
		spec   *LayoutSpec
		fields map[string]any
		body   string
	}{
		{
			name: "fields", format: BodyFormatFields,
			spec:   &LayoutSpec{Fields: &FieldsSpec{Required: []FieldSpec{{Name: "title"}}}},
			fields: map[string]any{"title": "Structured"}, body: "title: Raw",
		},
		{
			name: "rows", format: BodyFormatRows,
			spec:   &LayoutSpec{Rows: &RowsSpec{Delimiter: "|", MinColumns: 2}},
			fields: map[string]any{"rows": []any{[]any{"Structured", "Row"}}}, body: "Raw|Row",
		},
		{
			name: "json object", format: BodyFormatJSONObject,
			spec:   &LayoutSpec{Fields: &FieldsSpec{Required: []FieldSpec{{Name: "title"}}}},
			fields: map[string]any{"title": "Structured"}, body: `{"title":"Raw"}`,
		},
		{
			name: "json array", format: BodyFormatJSONArray,
			spec:   &LayoutSpec{Fields: &FieldsSpec{Required: []FieldSpec{{Name: "title"}}}},
			fields: map[string]any{"title": "Structured"}, body: `[{"title":"Raw"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCatalog()
			tt.spec.Name = "primary"
			tt.spec.BodyFormat = tt.format
			c.modules[tt.spec.Name] = tt.spec

			out, err := c.RenderBlock(tt.spec.Name, RenderInput{Fields: tt.fields, Body: tt.body})
			if err != nil {
				t.Fatal(err)
			}
			want := ":::primary\n" + tt.body + "\n:::\n"
			if out != want {
				t.Fatalf("RenderBlock() = %q, want %q", out, want)
			}
		})
	}
}

func TestRenderBlockRawBodyAcceptsCompatibleFormat(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	body := `[{"q":"支持吗？","a":"支持。"}]`
	out, err := c.RenderBlock("question", RenderInput{Body: body})
	if err != nil {
		t.Fatal(err)
	}
	if want := ":::question\n" + body + "\n:::\n"; out != want {
		t.Fatalf("RenderBlock() = %q, want %q", out, want)
	}
	if report := c.Validate(out); len(report.Errors) != 0 {
		t.Fatalf("rendered compatible body failed validation: %+v", report.Errors)
	}
}

func TestRenderLegacyBodyUsesRawPathWhenBodySpecExists(t *testing.T) {
	c := NewCatalog()
	c.modules["legacy-fields"] = &LayoutSpec{
		Name:       "legacy-fields",
		BodyFormat: BodyFormatFields,
		Fields:     &FieldsSpec{Required: []FieldSpec{{Name: "title"}}},
		Body:       &BodySpec{},
	}
	out, err := c.Render("legacy-fields", map[string]any{
		"title": "Structured",
		"body":  "title: Raw",
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := ":::legacy-fields\ntitle: Raw\n:::\n"; out != want {
		t.Fatalf("Render() = %q, want %q", out, want)
	}
}

func TestRenderBlockMarkdownFieldsSupportsStructuredFields(t *testing.T) {
	c := NewCatalog()
	c.modules["markdown-card"] = &LayoutSpec{
		Name:       "markdown-card",
		BodyFormat: BodyFormatMarkdownFields,
		Fields:     &FieldsSpec{Required: []FieldSpec{{Name: "title"}}},
	}
	out, err := c.RenderBlock("markdown-card", RenderInput{Fields: map[string]any{"title": "Structured"}})
	if err != nil {
		t.Fatal(err)
	}
	if want := ":::markdown-card\ntitle: Structured\n:::\n"; out != want {
		t.Fatalf("RenderBlock() = %q, want %q", out, want)
	}
}

func TestRenderRejectsFieldOutsideEffectiveVariantWithoutInjectingDefault(t *testing.T) {
	c := NewCatalog()
	c.modules["schema-fixture"] = &LayoutSpec{
		Name: "schema-fixture", BodyFormat: BodyFormatFields,
		Fields: &FieldsSpec{Optional: []FieldSpec{
			{Name: "variant", Default: "numbered"},
			{Name: "index", AppliesTo: []string{"numbered"}},
		}},
		Variants: []VariantSpec{{Name: "numbered"}, {Name: "plain"}},
	}
	if _, err := c.Render("schema-fixture", map[string]any{"variant": "plain", "index": "1"}); !errors.Is(err, ErrInvalidFieldValue) {
		t.Fatalf("Render() error = %v, want ErrInvalidFieldValue", err)
	}
	out, err := c.Render("schema-fixture", map[string]any{"index": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "variant:") {
		t.Fatalf("Render() injected field default: %q", out)
	}
}

func TestRowsSchemaAppliesToKeepsRenderAndValidateAligned(t *testing.T) {
	c := NewCatalog()
	c.modules["row-schema-fixture"] = &LayoutSpec{
		Name: "row-schema-fixture", BodyFormat: BodyFormatRows,
		Fields: &FieldsSpec{Optional: []FieldSpec{{Name: "variant", Default: "numbered"}}},
		Rows: &RowsSpec{Delimiter: "|", MinColumns: 2, MaxColumns: 2, Schema: []FieldSpec{
			{Name: "label"}, {Name: "index", AppliesTo: []string{"numbered"}},
		}},
		Variants: []VariantSpec{{Name: "numbered"}, {Name: "plain"}},
	}
	tests := []struct {
		name, variant string
		accepted      bool
	}{
		{name: "selected variant accepts row field", variant: "numbered", accepted: true},
		{name: "other variant rejects row field", variant: "plain"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := map[string]any{"variant": tt.variant, "rows": []any{[]any{"Chapter", "1"}}}
			out, err := c.Render("row-schema-fixture", fields)
			if got := err == nil; got != tt.accepted {
				t.Fatalf("Render() error = %v, accepted = %v, want %v", err, got, tt.accepted)
			}
			markdown := ":::row-schema-fixture\nvariant: " + tt.variant + "\nChapter|1\n:::\n"
			report := c.Validate(markdown)
			if got := len(report.Errors) == 0; got != tt.accepted {
				t.Fatalf("Validate() errors = %+v, accepted = %v, want %v", report.Errors, got, tt.accepted)
			}
			if tt.accepted && len(c.Validate(out).Errors) != 0 {
				t.Fatalf("rendered output failed validation: %q", out)
			}
		})
	}
}

func TestRenderBlockRejectsCaptionWithExplicitOrDefaultParams(t *testing.T) {
	tests := []struct {
		name   string
		param  ParamSpec
		params map[string]string
	}{
		{name: "explicit", param: ParamSpec{Name: "columns"}, params: map[string]string{"columns": "3"}},
		{name: "default", param: ParamSpec{Name: "columns", Default: "3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCatalog()
			c.modules["gallery"] = &LayoutSpec{
				Name:       "gallery",
				BodyFormat: BodyFormatLines,
				Opener: &OpenerSpec{
					Caption:    true,
					ParamStyle: ParamStyleBraces,
					Params:     []ParamSpec{tt.param},
				},
				Body: &BodySpec{MinItems: 1},
			}
			_, err := c.RenderBlock("gallery", RenderInput{
				Params:  tt.params,
				Caption: "Screenshots",
				Body:    "One item",
			})
			if !errors.Is(err, ErrInvalidFieldValue) {
				t.Fatalf("RenderBlock() error = %v, want ErrInvalidFieldValue", err)
			}
		})
	}
}

func TestRenderHeroFields(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	out, err := c.Render("hero", map[string]any{
		"eyebrow":  "深度观察",
		"title":    "公众号排版的真问题不是好不好看",
		"subtitle": "是读者愿不愿意读完",
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.HasPrefix(out, ":::hero") || !strings.HasSuffix(strings.TrimRight(out, "\n"), ":::") {
		t.Errorf("output missing :::hero block fence:\n%s", out)
	}
	for _, want := range []string{"eyebrow: 深度观察", "title: 公众号排版的真问题不是好不好看", "subtitle: 是读者愿不愿意读完"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
}

func TestRenderMissingRequiredFieldFails(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	_, err := c.Render("hero", map[string]any{"eyebrow": "x"})
	if err == nil {
		t.Fatal("expected error for missing title")
	}
	if !strings.Contains(err.Error(), "title") {
		t.Errorf("error should mention missing field name, got: %v", err)
	}
}

func TestRenderRejectsUnknownStructuredField(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	_, err := c.Render("hero", map[string]any{
		"title":    "T",
		"subtitel": "typo",
	})
	if !errors.Is(err, ErrInvalidFieldValue) || !strings.Contains(err.Error(), "subtitel") {
		t.Fatalf("Render() error = %v, want unknown-field ErrInvalidFieldValue", err)
	}
}

func TestRenderUnknownModuleFails(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	_, err := c.Render("nonexistent", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRenderBlockDistinguishesReservedAndUnknownModuleNames(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	if _, err := c.RenderBlock(reservedModuleName, RenderInput{}); !errors.Is(err, ErrInvalidFieldValue) || errors.Is(err, ErrUnknownModule) {
		t.Fatalf("RenderBlock(%q) error = %v, want only ErrInvalidFieldValue", reservedModuleName, err)
	}
	if _, err := c.RenderBlock("well-formed-unknown", RenderInput{}); !errors.Is(err, ErrUnknownModule) || errors.Is(err, ErrInvalidFieldValue) {
		t.Fatalf("RenderBlock(well-formed-unknown) error = %v, want only ErrUnknownModule", err)
	}
}

// rowsSpecWithEnum returns an in-memory LayoutSpec that has rows mode,
// a required field with an enum constraint, and an optional field with an enum constraint.
func rowsSpecWithEnum() *LayoutSpec {
	return &LayoutSpec{
		SchemaVersion: SchemaVersion,
		Name:          "test-rows-enum",
		BodyFormat:    BodyFormatRows,
		Category:      "body",
		Serves:        []string{"readability"},
		Fields: &FieldsSpec{
			Required: []FieldSpec{
				{Name: "align", Enum: []string{"left", "center", "right"}},
			},
			Optional: []FieldSpec{
				{Name: "style", Enum: []string{"plain", "bordered"}},
			},
		},
		Rows: &RowsSpec{Delimiter: "|", MinColumns: 1},
		Metadata: LayoutMetadata{
			Author:     "test",
			Provenance: "test",
		},
	}
}

func TestRenderRowsEnumRequiredInvalidFails(t *testing.T) {
	c := NewCatalog()
	c.modules["test-rows-enum"] = rowsSpecWithEnum()

	_, err := c.Render("test-rows-enum", map[string]any{
		"align": "invalid-align",
		"rows":  []any{[]any{"cell1"}},
	})
	if err == nil {
		t.Fatal("expected error for invalid enum value on required field in rows mode")
	}
	if !strings.Contains(err.Error(), "align") {
		t.Errorf("error should mention field name 'align', got: %v", err)
	}
}

func TestRenderRowsEnumOptionalInvalidFails(t *testing.T) {
	c := NewCatalog()
	c.modules["test-rows-enum"] = rowsSpecWithEnum()

	_, err := c.Render("test-rows-enum", map[string]any{
		"align": "left",
		"style": "fancy", // not in enum
		"rows":  []any{[]any{"cell1"}},
	})
	if err == nil {
		t.Fatal("expected error for invalid enum value on optional field in rows mode")
	}
	if !strings.Contains(err.Error(), "style") {
		t.Errorf("error should mention field name 'style', got: %v", err)
	}
}

func TestRenderRowsEnumValidSucceeds(t *testing.T) {
	c := NewCatalog()
	c.modules["test-rows-enum"] = rowsSpecWithEnum()

	out, err := c.Render("test-rows-enum", map[string]any{
		"align": "center",
		"style": "bordered",
		"rows":  []any{[]any{"a", "b"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "align: center") || !strings.Contains(out, "style: bordered") || !strings.Contains(out, "a|b") {
		t.Errorf("output missing expected fields:\n%s", out)
	}
}

func TestRenderJSONArrayRequiresFields(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	_, err := c.Render("question", map[string]any{})
	if err == nil {
		t.Fatal("expected missing q/a error")
	}
}

func TestRenderJSONFieldsUsesExampleBodyShape(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	out, err := c.Render("definition", map[string]any{
		"term":      "OKR",
		"def":       "目标与关键结果",
		"termLabel": "术语",
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.Contains(out, `{"def":"目标与关键结果","term":"OKR","termLabel":"术语"}`) {
		t.Fatalf("definition should render JSON body, got:\n%s", out)
	}
}

func TestRenderJSONArrayFieldsUsesExampleBodyShape(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	out, err := c.Render("question", map[string]any{
		"q": "模块是否支持所有主题？",
		"a": "支持。",
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.Contains(out, `[{"a":"支持。","q":"模块是否支持所有主题？"}]`) {
		t.Fatalf("question should render JSON array body, got:\n%s", out)
	}
}
