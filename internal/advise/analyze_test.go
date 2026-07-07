package advise

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzeShortArticleNeedsNoEnhancement(t *testing.T) {
	result, err := Analyze(Input{
		SourceFile: filepath.Join(t.TempDir(), "article.md"),
		Markdown:   "# 标题\n\n这是一段短文。",
	})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if result.Decision != DecisionNoEnhancementNeeded {
		t.Fatalf("decision = %q", result.Decision)
	}
	if len(result.Actions) != 0 {
		t.Fatalf("actions = %#v", result.Actions)
	}
	if len(result.Layout.Recommended) != 0 {
		t.Fatalf("layout.recommended = %#v", result.Layout.Recommended)
	}
}

func TestAnalyzeRecommendsTitleWhenTitleMissing(t *testing.T) {
	result, err := Analyze(Input{
		SourceFile: filepath.Join(t.TempDir(), "article.md"),
		Markdown:   "正文第一段。\n\n正文第二段。",
	})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	action := findAction(result.Actions, ToolTitle)
	if action == nil {
		t.Fatalf("missing title action in %#v", result.Actions)
	}
	if action.State != ActionStateRecommended {
		t.Fatalf("title state = %q", action.State)
	}
	if action.CommandHint != "md2wechat title suggest article.md --json" {
		t.Fatalf("command_hint = %q", action.CommandHint)
	}
	if !action.RequiresConfirmation {
		t.Fatal("title action should require confirmation")
	}
	if len(action.Evidence) == 0 {
		t.Fatalf("title evidence = %#v", action.Evidence)
	}
}

func TestAnalyzeRecommendsBoundedLayoutModules(t *testing.T) {
	markdown := "# 教程\n\n## 背景\n\n正文。\n\n## 步骤\n\n1. 第一步\n2. 第二步\n\n## 结果\n\n> 这是一句可以引用的金句。\n\n## 总结\n\n正文。"
	result, err := Analyze(Input{
		SourceFile: filepath.Join(t.TempDir(), "article.md"),
		Markdown:   markdown,
	})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(result.Layout.Recommended) == 0 {
		t.Fatalf("expected layout recommendations")
	}
	if len(result.Layout.Recommended) > 3 {
		t.Fatalf("layout recommendations = %d, want <= 3", len(result.Layout.Recommended))
	}
	for _, name := range []string{"toc", "steps", "quote"} {
		module := findModule(result.Layout.Recommended, name)
		if module == nil {
			t.Fatalf("missing %s recommendation: %#v", name, result.Layout.Recommended)
		}
		if module.State != ActionStateRecommended {
			t.Fatalf("%s state = %q", name, module.State)
		}
		if len(module.Evidence) == 0 {
			t.Fatalf("%s evidence = %#v", name, module.Evidence)
		}
	}
}

func TestAnalyzeRejectsUnsupportedModulesWithoutEvidence(t *testing.T) {
	result, err := Analyze(Input{
		SourceFile: filepath.Join(t.TempDir(), "article.md"),
		Markdown:   "# 标题\n\n正文没有数字指标，也没有行动号召。",
	})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if !hasRejectedModule(result.Layout.Rejected, "metrics") {
		t.Fatalf("expected metrics rejection: %#v", result.Layout.Rejected)
	}
	if !hasRejectedModule(result.Layout.Rejected, "cta") {
		t.Fatalf("expected cta rejection: %#v", result.Layout.Rejected)
	}
}

func TestAnalyzeDoesNotWriteFiles(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "article.md")
	if err := os.WriteFile(source, []byte("# 标题\n\n正文"), 0600); err != nil {
		t.Fatalf("write source: %v", err)
	}
	before, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read before: %v", err)
	}
	if _, err := Analyze(Input{SourceFile: source, Markdown: string(before)}); err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	after, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read after: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("Analyze mutated source file")
	}
}

func TestAnalyzeJSONContract(t *testing.T) {
	result, err := Analyze(Input{
		SourceFile: filepath.Join(t.TempDir(), "article.md"),
		Markdown:   "# 标题\n\n正文",
	})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	var contract map[string]json.RawMessage
	if err := json.Unmarshal(payload, &contract); err != nil {
		t.Fatalf("unmarshal result contract: %v", err)
	}
	for _, key := range []string{
		"schema_version",
		"command",
		"source",
		"decision",
		"article",
		"actions",
		"layout",
		"micro_edits",
		"do_not_do",
	} {
		if _, ok := contract[key]; !ok {
			t.Fatalf("missing JSON key %q in %s", key, payload)
		}
	}
	assertJSONArray(t, contract["actions"], "actions")
	assertJSONArray(t, contract["micro_edits"], "micro_edits")

	var layout map[string]json.RawMessage
	if err := json.Unmarshal(contract["layout"], &layout); err != nil {
		t.Fatalf("unmarshal layout contract: %v", err)
	}
	for _, key := range []string{"recommended", "rejected"} {
		raw, ok := layout[key]
		if !ok {
			t.Fatalf("missing JSON key %q in layout: %s", key, contract["layout"])
		}
		assertJSONArray(t, raw, "layout."+key)
	}

	var article map[string]json.RawMessage
	if err := json.Unmarshal(contract["article"], &article); err != nil {
		t.Fatalf("unmarshal article contract: %v", err)
	}
	for _, key := range []string{"signals", "uncertainties"} {
		raw, ok := article[key]
		if !ok {
			t.Fatalf("missing JSON key %q in article: %s", key, contract["article"])
		}
		assertJSONArray(t, raw, "article."+key)
	}
	if !json.Valid(payload) {
		t.Fatalf("invalid json: %s", payload)
	}
	if result.SchemaVersion != SchemaVersion {
		t.Fatalf("schema_version = %q", result.SchemaVersion)
	}
	if result.Command != "advise" {
		t.Fatalf("command = %q", result.Command)
	}
	if result.Layout.MaxModules != 3 {
		t.Fatalf("layout.max_modules = %d", result.Layout.MaxModules)
	}
}

func findAction(actions []Action, tool string) *Action {
	for i := range actions {
		if actions[i].Tool == tool {
			return &actions[i]
		}
	}
	return nil
}

func findModule(modules []ModuleAdvice, name string) *ModuleAdvice {
	for i := range modules {
		if modules[i].Name == name {
			return &modules[i]
		}
	}
	return nil
}

func hasRejectedModule(modules []ModuleAdvice, name string) bool {
	for _, module := range modules {
		if module.Name == name && module.State == ActionStateSkip {
			return true
		}
	}
	return false
}

func assertJSONArray(t *testing.T, raw json.RawMessage, path string) {
	t.Helper()

	if len(raw) == 0 {
		t.Fatalf("%s is missing", path)
	}
	if raw[0] != '[' {
		t.Fatalf("%s should marshal as JSON array, got %s", path, raw)
	}
}
