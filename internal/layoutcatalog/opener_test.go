package layoutcatalog

import (
	"strings"
	"testing"
)

func TestParseBlockOpener(t *testing.T) {
	tests := []struct{ line, name, caption, raw string }{
		{":::toc[阅读导航]", "toc", "阅读导航", ""},
		{":::gallery-grid{columns=3 variant=card}", "gallery-grid", "", "{columns=3 variant=card}"},
		{":::matrix headers=能力,基础版,专业版", "matrix", "", "headers=能力,基础版,专业版"},
		{":::dialogue-pair left=读者 right=作者", "dialogue-pair", "", "left=读者 right=作者"},
		{":::split[3:2]", "split", "3:2", ""},
		{":::callout warning", "callout", "", "warning"},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got, err := parseBlockOpener(tt.line)
			if err != nil {
				t.Fatal(err)
			}
			if got.Name != tt.name || got.Caption != tt.caption || got.RawParams != tt.raw {
				t.Fatalf("parseBlockOpener() = %+v", got)
			}
		})
	}
}

func TestParseBlockOpenerRejectsMalformedForms(t *testing.T) {
	for _, line := range []string{
		":::block",
		":::block hero",
		":::block[hero]",
		":::block{type=hero}",
		":::gallery-grid{columns=3",
		":::future [标题]",
		":::future {columns=3}",
		":::toc[阅读导航] trailing",
		":::matrix =value",
		":::matrix headers=one headers=two",
		":::matrix bad.name=value",
		":::matrix 1header=value",
		":::matrix headers＝one",
	} {
		t.Run(line, func(t *testing.T) {
			if _, err := parseBlockOpener(line); err == nil {
				t.Fatalf("parseBlockOpener(%q) unexpectedly succeeded", line)
			}
		})
	}
}

func TestValidateOpenerMapsFiniteStyles(t *testing.T) {
	tests := []struct {
		name string
		line string
		spec *OpenerSpec
		want ParsedOpener
	}{
		{
			name: "caption",
			line: ":::toc[阅读导航]",
			spec: &OpenerSpec{Caption: true},
			want: ParsedOpener{Name: "toc", Caption: "阅读导航", Params: map[string]string{}},
		},
		{
			name: "bracket parameter",
			line: ":::split[3:2]",
			spec: &OpenerSpec{ParamStyle: ParamStyleBracket, Params: []ParamSpec{{Name: "ratio"}}},
			want: ParsedOpener{Name: "split", Params: map[string]string{"ratio": "3:2"}},
		},
		{
			name: "raw token parameter",
			line: ":::callout warning",
			spec: &OpenerSpec{ParamStyle: ParamStyleToken, Params: []ParamSpec{{Name: "type", Enum: []string{"info", "warning"}}}},
			want: ParsedOpener{Name: "callout", RawParams: "warning", Params: map[string]string{"type": "warning"}},
		},
		{
			name: "braced parameters",
			line: ":::gallery-grid{columns=3 variant=card}",
			spec: &OpenerSpec{ParamStyle: ParamStyleBraces, Params: []ParamSpec{{Name: "columns"}, {Name: "variant"}}},
			want: ParsedOpener{Name: "gallery-grid", RawParams: "{columns=3 variant=card}", Params: map[string]string{"columns": "3", "variant": "card"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parseBlockOpener(tt.line)
			if err != nil {
				t.Fatal(err)
			}
			got, err := validateOpener(parsed, tt.spec)
			if err != nil {
				t.Fatal(err)
			}
			if got.Name != tt.want.Name || got.Caption != tt.want.Caption || got.RawParams != tt.want.RawParams || !equalStringMaps(got.Params, tt.want.Params) {
				t.Fatalf("validateOpener() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestValidateOpenerRejectsSpecViolations(t *testing.T) {
	tests := []struct {
		line string
		spec *OpenerSpec
	}{
		{":::toc[阅读导航]", &OpenerSpec{}},
		{":::toc[]", &OpenerSpec{}},
		{":::toc[]", &OpenerSpec{Caption: true}},
		{":::split[]", &OpenerSpec{ParamStyle: ParamStyleBracket, Params: []ParamSpec{{Name: "ratio"}}}},
		{":::matrix unknown=value", &OpenerSpec{ParamStyle: ParamStyleTokens, Params: []ParamSpec{{Name: "headers"}}}},
		{":::callout danger", &OpenerSpec{ParamStyle: ParamStyleToken, Params: []ParamSpec{{Name: "type", Enum: []string{"info", "warning"}}}}},
		{":::gallery-grid columns=3", &OpenerSpec{ParamStyle: ParamStyleBraces, Params: []ParamSpec{{Name: "columns"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			parsed, err := parseBlockOpener(tt.line)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := validateOpener(parsed, tt.spec); err == nil {
				t.Fatal("validateOpener() unexpectedly succeeded")
			}
		})
	}
}

func TestValidateOpenerPreservesLegacyCaptionCompatibility(t *testing.T) {
	parsed, err := parseBlockOpener(":::fixture-module[标题]")
	if err != nil {
		t.Fatal(err)
	}
	got, err := validateOpener(parsed, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Caption != "标题" {
		t.Fatalf("Caption = %q, want 标题", got.Caption)
	}
}

func TestRenderOpenerSortsParameterKeysIncludingDefaults(t *testing.T) {
	spec := &LayoutSpec{
		Name: "gallery-grid",
		Opener: &OpenerSpec{
			ParamStyle: ParamStyleBraces,
			Params: []ParamSpec{
				{Name: "zeta"},
				{Name: "middle", Default: "m"},
				{Name: "alpha"},
			},
		},
	}
	vars := map[string]any{}
	vars["zeta"] = "z"
	vars["alpha"] = "a"
	got, err := renderOpenerWithOrder(spec, vars, openerParamOrderLexical)
	if err != nil {
		t.Fatal(err)
	}
	if got != ":::gallery-grid{alpha=a middle=m zeta=z}" {
		t.Fatalf("renderOpener() = %q", got)
	}
}

func TestRenderOpenerRoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		spec       *LayoutSpec
		vars       map[string]any
		wantParams map[string]string
	}{
		{
			name: "braces",
			spec: &LayoutSpec{Name: "gallery-grid", Opener: &OpenerSpec{
				ParamStyle: ParamStyleBraces,
				Params:     []ParamSpec{{Name: "columns"}, {Name: "variant"}},
			}},
			vars:       map[string]any{"columns": 3, "variant": "card"},
			wantParams: map[string]string{"columns": "3", "variant": "card"},
		},
		{
			name: "tokens",
			spec: &LayoutSpec{Name: "dialogue-pair", Opener: &OpenerSpec{
				ParamStyle: ParamStyleTokens,
				Params:     []ParamSpec{{Name: "left"}, {Name: "right"}},
			}},
			vars:       map[string]any{"left": "reader", "right": "author"},
			wantParams: map[string]string{"left": "reader", "right": "author"},
		},
		{
			name: "token",
			spec: &LayoutSpec{Name: "callout", Opener: &OpenerSpec{
				ParamStyle: ParamStyleToken,
				Params:     []ParamSpec{{Name: "type", Enum: []string{"info", "warning"}}},
			}},
			vars:       map[string]any{"type": "warning"},
			wantParams: map[string]string{"type": "warning"},
		},
		{
			name: "bracket",
			spec: &LayoutSpec{Name: "split", Opener: &OpenerSpec{
				ParamStyle: ParamStyleBracket,
				Params:     []ParamSpec{{Name: "ratio"}},
			}},
			vars:       map[string]any{"ratio": "3 : 2"},
			wantParams: map[string]string{"ratio": "3 : 2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered, err := renderOpener(tt.spec, tt.vars)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := parseBlockOpener(rendered)
			if err != nil {
				t.Fatalf("parse rendered opener %q: %v", rendered, err)
			}
			validated, err := validateOpener(parsed, tt.spec.Opener)
			if err != nil {
				t.Fatalf("validate rendered opener %q: %v", rendered, err)
			}
			if !equalStringMaps(validated.Params, tt.wantParams) {
				t.Fatalf("round-trip params = %v, want %v", validated.Params, tt.wantParams)
			}
		})
	}
}

func TestRenderOpenerRejectsGrammarChangingValues(t *testing.T) {
	tests := []struct {
		name  string
		style string
		param string
		value string
	}{
		{"braces assignment", ParamStyleBraces, "variant", "card=wide"},
		{"braces opening brace", ParamStyleBraces, "variant", "{card"},
		{"braces closing brace", ParamStyleBraces, "variant", "card}"},
		{"braces full-width assignment", ParamStyleBraces, "variant", "card＝wide"},
		{"tokens assignment", ParamStyleTokens, "headers", "basic=pro"},
		{"tokens full-width assignment", ParamStyleTokens, "headers", "basic：pro"},
		{"raw token assignment", ParamStyleToken, "type", "warn=ing"},
		{"raw token full-width equals", ParamStyleToken, "type", "warn＝ing"},
		{"raw token full-width colon", ParamStyleToken, "type", "warn：ing"},
		{"invalid param name", ParamStyleTokens, "bad.name", "value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &LayoutSpec{Name: "future", Opener: &OpenerSpec{
				ParamStyle: tt.style,
				Params:     []ParamSpec{{Name: tt.param}},
			}}
			if _, err := renderOpener(spec, map[string]any{tt.param: tt.value}); err == nil {
				t.Fatalf("renderOpener() accepted grammar-changing value %q", tt.value)
			}
		})
	}
}

func TestRenderOpenerValidatesSpecBeforeCaptionShortcut(t *testing.T) {
	spec := &LayoutSpec{Name: "split", Opener: &OpenerSpec{
		Caption:    true,
		ParamStyle: ParamStyleBracket,
		Params:     []ParamSpec{{Name: "ratio"}},
	}}
	if _, err := renderOpener(spec, map[string]any{"caption": "标题"}); err == nil {
		t.Fatal("renderOpener() accepted caption/bracket ambiguity")
	}
}

func equalStringMaps(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}

func TestParseLayoutSpecValidatesOpenerSchema(t *testing.T) {
	base := `schema_version: "1"
name: fixture
version: "1.0.0"
category: body
serves: [readability]
%s
metadata:
  author: test
  provenance: test-fixture
`
	for _, tc := range []struct {
		name   string
		opener string
		want   string
	}{
		{"invalid style", "opener:\n  param_style: freeform", "invalid opener param_style"},
		{"duplicate params", "opener:\n  param_style: tokens\n  params:\n    - name: side\n    - name: side", "duplicate opener param"},
		{"bracket arity", "opener:\n  param_style: bracket\n  params:\n    - name: left\n    - name: right", "requires exactly one parameter"},
		{"token arity", "opener:\n  param_style: token", "requires exactly one parameter"},
		{"caption bracket ambiguity", "opener:\n  caption: true\n  param_style: bracket\n  params:\n    - name: ratio", "caption and bracket"},
		{"caption params ambiguity", "opener:\n  caption: true\n  param_style: braces\n  params:\n    - name: columns", "caption and opener params"},
		{"params without style", "opener:\n  params:\n    - name: columns", "params require param_style"},
		{"invalid param name", "opener:\n  param_style: tokens\n  params:\n    - name: bad.name", "invalid opener param name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseLayoutSpec([]byte(strings.Replace(base, "%s", tc.opener, 1)))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("parseLayoutSpec() error = %v, want substring %q", err, tc.want)
			}
		})
	}
}
