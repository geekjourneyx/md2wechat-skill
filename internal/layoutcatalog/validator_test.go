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
