package layoutcatalog

import "testing"

func TestValidateValidHero(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	md := ":::hero\neyebrow: 深度观察\ntitle: 真问题\n:::\n"
	r := c.Validate(md)
	if len(r.Errors) != 0 {
		t.Errorf("expected no errors, got %v", r.Errors)
	}
}

func TestValidateMissingRequiredField(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	md := ":::hero\neyebrow: x\n:::\n"
	r := c.Validate(md)
	if len(r.Errors) == 0 {
		t.Fatalf("expected error for missing title")
	}
	if r.Errors[0].Module != "hero" || r.Errors[0].Field != "title" {
		t.Errorf("unexpected error: %+v", r.Errors[0])
	}
}

func TestValidateBracketTitleKnownModule(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	// toc is a rows-mode module; bracket title must not cause an unknown-module warning.
	md := ":::toc[阅读导航]\n01 | 先看模块 | 概述\n:::\n"
	r := c.Validate(md)
	if len(r.Warnings) != 0 {
		t.Errorf("expected no warnings for known module with bracket title, got %v", r.Warnings)
	}
}

func TestValidateMetricsBracketMissingRequired(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	// metrics is rows-mode; empty block should produce an error.
	md := ":::metrics[核心数据]\n:::\n"
	r := c.Validate(md)
	if len(r.Errors) == 0 {
		t.Error("expected at least one error for empty rows-mode block, got none")
	}
}

func TestValidateUnknownModuleWarns(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	md := ":::futuristic-block\nfoo: bar\n:::\n"
	r := c.Validate(md)
	if len(r.Errors) != 0 {
		t.Errorf("unknown module must NOT error, got %v", r.Errors)
	}
	if len(r.Warnings) != 1 || r.Warnings[0].Module != "futuristic-block" {
		t.Errorf("expected one warning for futuristic-block, got %+v", r.Warnings)
	}
}

func TestValidateUnknownModuleWithWellFormedParamsWarns(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	md := ":::future-grid{columns=3}\nfoo: bar\n:::\n"
	r := c.Validate(md)
	if len(r.Errors) != 0 {
		t.Fatalf("unknown well-formed module must not error, got %v", r.Errors)
	}
	if len(r.Warnings) != 1 || r.Warnings[0].Module != "future-grid" {
		t.Fatalf("expected future-grid warning, got %+v", r.Warnings)
	}
}

func TestValidateWhitespacePrefixedStructuredSuffixErrors(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	for _, opener := range []string{":::future [标题]", ":::future {columns=3}"} {
		t.Run(opener, func(t *testing.T) {
			r := c.Validate(opener + "\nfoo: bar\n:::\n")
			if len(r.Errors) == 0 || r.Errors[0].Message != "invalid layout block opener" {
				t.Fatalf("expected structural opener error, got %+v", r)
			}
			if len(r.Warnings) != 0 {
				t.Fatalf("malformed opener must not warn as unknown, got %+v", r.Warnings)
			}
		})
	}
}

func TestValidateKnownModuleOpenerAgainstSpec(t *testing.T) {
	c := NewCatalog()
	c.modules["callout"] = &LayoutSpec{
		Name:       "callout",
		BodyFormat: BodyFormatFields,
		Opener: &OpenerSpec{
			ParamStyle: ParamStyleToken,
			Params:     []ParamSpec{{Name: "type", Enum: []string{"info", "warning"}}},
		},
	}

	valid := c.Validate(":::callout warning\n:::\n")
	if len(valid.Errors) != 0 {
		t.Fatalf("valid token opener errors = %v", valid.Errors)
	}
	invalid := c.Validate(":::callout danger\n:::\n")
	if len(invalid.Errors) != 1 || invalid.Errors[0].Module != "callout" {
		t.Fatalf("invalid token opener errors = %+v", invalid.Errors)
	}
}

func TestValidateInvalidBlockOpenerErrors(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	md := ":::block hero\ntitle: Wrong syntax\n:::\n"
	r := c.Validate(md)
	if len(r.Errors) == 0 {
		t.Fatal("expected invalid opener error")
	}
	if r.Errors[0].Message != "invalid layout block opener" {
		t.Fatalf("unexpected error: %+v", r.Errors[0])
	}
}

func TestValidateReservedBlockNameAlwaysErrors(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	for _, opener := range []string{":::block", ":::block[hero]", ":::block{type=hero}"} {
		t.Run(opener, func(t *testing.T) {
			r := c.Validate(opener + "\n:::\n")
			if len(r.Errors) == 0 || r.Errors[0].Message != "invalid layout block opener" {
				t.Fatalf("reserved block opener report = %+v", r)
			}
			if len(r.Warnings) != 0 {
				t.Fatalf("reserved block name must not degrade to unknown warning: %+v", r.Warnings)
			}
		})
	}
}

func TestValidateJSONFields(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	md := ":::definition\n{\"term\":\"OKR\",\"def\":\"目标与关键结果\",\"termLabel\":\"术语\"}\n:::\n"
	r := c.Validate(md)
	if len(r.Errors) != 0 {
		t.Fatalf("expected JSON definition to validate, got %v", r.Errors)
	}
}

func TestValidateJSONFieldsRejectsMissingRequired(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	md := ":::definition\n{\"term\":\"OKR\"}\n:::\n"
	r := c.Validate(md)
	if len(r.Errors) == 0 {
		t.Fatal("expected missing def error")
	}
	if r.Errors[0].Module != "definition" || r.Errors[0].Field != "def" {
		t.Fatalf("unexpected error: %+v", r.Errors[0])
	}
}

func TestValidateJSONArrayRejectsEmptyObject(t *testing.T) {
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	md := ":::question\n[{}]\n:::\n"
	r := c.Validate(md)
	if len(r.Errors) == 0 {
		t.Fatal("expected missing q/a errors")
	}
}

func TestCalibratedVariantAndBodyMatrix(t *testing.T) {
	tests := []struct {
		name, markdown string
		accepted       bool
	}{
		{name: "hero title", markdown: ":::hero\ntitle: 标题\n:::\n", accepted: true},
		{name: "hero missing title", markdown: ":::hero\nsubtitle: 副标题\n:::\n"},
		{name: "quote proof source", markdown: ":::quote\nvariant: proof\nquote: 证据\nsource: 研究报告\n:::\n", accepted: true},
		{name: "quote proof missing source", markdown: ":::quote\nvariant: proof\nquote: 证据\n:::\n"},
		{name: "quote proof alias", markdown: ":::quote\nvariant: evidence\nquote: 证据\nsource: 研究报告\n:::\n", accepted: true},
		{name: "quote unknown variant", markdown: ":::quote\nvariant: loud\nquote: 证据\n:::\n"},
		{name: "summary legacy", markdown: ":::summary\nhighlight: 一句话\n:::\n", accepted: true},
		{name: "summary legacy missing highlight", markdown: ":::summary\nvariant: legacy\ntitle: 标题\n:::\n"},
		{name: "summary three", markdown: ":::summary\nvariant: three\nitems: 一 | 二 | 三\n:::\n", accepted: true},
		{name: "summary three alias", markdown: ":::summary\nvariant: points\nitems: 一 | 二 | 三\n:::\n", accepted: true},
		{name: "summary three missing items", markdown: ":::summary\nvariant: three\ntitle: 三点\n:::\n"},
		{name: "summary save", markdown: ":::summary\nvariant: save\nitems: 一 | 二\n:::\n", accepted: true},
		{name: "summary decision fit", markdown: ":::summary\nvariant: decision\nfit: A\n:::\n", accepted: true},
		{name: "summary decision recommendation", markdown: ":::summary\nvariant: decision\nrecommendation: 选择 A\n:::\n", accepted: true},
		{name: "summary decision empty", markdown: ":::summary\nvariant: decision\ntitle: 决策\n:::\n"},
		{name: "question dialogue", markdown: ":::question\nQ: 支持吗？\nA: 支持。\n:::\n", accepted: true},
		{name: "question legacy JSON", markdown: ":::question\n[{\"q\":\"支持吗？\",\"a\":\"支持。\"}]\n:::\n", accepted: true},
		{name: "callout warning", markdown: ":::callout warning\n注意备份。\n:::\n", accepted: true},
		{name: "callout invalid variant", markdown: ":::callout tip\n提示。\n:::\n"},
		{name: "callout empty", markdown: ":::callout info\n:::\n"},
		{name: "image steps complete", markdown: ":::image-steps{columns=2}\nstep: 打开编辑器\ndesc: 选择主题\n![界面](https://example.com/a.png)\n:::\n", accepted: true},
		{name: "image steps without image", markdown: ":::image-steps\nstep: 打开编辑器\ndesc: 选择主题\n:::\n", accepted: true},
		{name: "image steps incomplete", markdown: ":::image-steps\nstep: 打开编辑器\n:::\n"},
		{name: "tweet complete", markdown: ":::tweet\n{\"name\":\"作者\",\"text\":\"观点\"}\n:::\n", accepted: true},
		{name: "tweet missing name", markdown: ":::tweet\n{\"text\":\"观点\"}\n:::\n"},
		{name: "resource without URL", markdown: ":::resource-list\n[{\"name\":\"指南\"}]\n:::\n", accepted: true},
	}
	c := NewCatalog()
	if err := c.Load(); err != nil {
		t.Fatal(err)
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
