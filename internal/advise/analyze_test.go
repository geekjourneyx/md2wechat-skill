package advise

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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
	source := filepath.Join(t.TempDir(), "article.md ")
	result, err := Analyze(Input{
		SourceFile: source,
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
	if action.CommandHint != "md2wechat title suggest "+shellQuote(source)+" --json" {
		t.Fatalf("command_hint = %q", action.CommandHint)
	}
	wantArgs := []string{"md2wechat", "title", "suggest", source, "--json"}
	if !reflect.DeepEqual(action.CommandArgs, wantArgs) {
		t.Fatalf("command_args = %#v, want %#v", action.CommandArgs, wantArgs)
	}
	if !action.RequiresConfirmation {
		t.Fatal("title action should require confirmation")
	}
	if len(action.Evidence) == 0 {
		t.Fatalf("title evidence = %#v", action.Evidence)
	}
	if result.Source != source {
		t.Fatalf("source = %q, want %q", result.Source, source)
	}
}

func TestAnalyzeRecommendsTitleWhenOnlyH2Exists(t *testing.T) {
	result, err := Analyze(Input{
		SourceFile: filepath.Join(t.TempDir(), "article.md"),
		Markdown:   "## 背景\n\n正文第一段。\n\n正文第二段。",
	})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	action := findAction(result.Actions, ToolTitle)
	if action == nil {
		t.Fatalf("missing title action in %#v", result.Actions)
	}
	if !hasSignal(result.Article.Uncertainties, "title_missing") {
		t.Fatalf("uncertainties = %#v", result.Article.Uncertainties)
	}
	if hasSignal(result.Article.Signals, "has_title") {
		t.Fatalf("signals = %#v", result.Article.Signals)
	}
}

func TestAnalyzeCoverCommandHintIncludesCLIName(t *testing.T) {
	source := filepath.Join(t.TempDir(), "drafts", "feature article.md")
	result, err := Analyze(Input{
		SourceFile: source,
		Markdown:   "# 标题\n\n## 背景\n\n正文。\n\n## 方案\n\n正文。",
	})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	action := findAction(result.Actions, ToolCover)
	if action == nil {
		t.Fatalf("missing cover action in %#v", result.Actions)
	}
	if action.CommandHint != "md2wechat generate_cover --article "+shellQuote(source)+" --plan --json" {
		t.Fatalf("command_hint = %q", action.CommandHint)
	}
	wantArgs := []string{"md2wechat", "generate_cover", "--article", source, "--plan", "--json"}
	if !reflect.DeepEqual(action.CommandArgs, wantArgs) {
		t.Fatalf("command_args = %#v, want %#v", action.CommandArgs, wantArgs)
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
	action := findAction(result.Actions, ToolLayout)
	if action == nil {
		t.Fatalf("missing layout action in %#v", result.Actions)
	}
	wantArgs := []string{"md2wechat", "layout", "list", "--json"}
	if !reflect.DeepEqual(action.CommandArgs, wantArgs) {
		t.Fatalf("layout command_args = %#v, want %#v", action.CommandArgs, wantArgs)
	}
}

func TestAnalyzeDoesNotTreatVersionsOrDatesAsMetrics(t *testing.T) {
	result, err := Analyze(Input{
		SourceFile: filepath.Join(t.TempDir(), "article.md"),
		Markdown:   "# 发布说明\n\n版本 v2.9.0 在 2026 年发布。\n\n这只是日期和版本号，不是业务指标。",
	})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if hasSignal(result.Article.Signals, "has_metrics") {
		t.Fatalf("signals = %#v", result.Article.Signals)
	}
	if findModule(result.Layout.Recommended, "metrics") != nil {
		t.Fatalf("unexpected metrics recommendation: %#v", result.Layout.Recommended)
	}
	if !hasRejectedModule(result.Layout.Rejected, "metrics") {
		t.Fatalf("expected metrics rejection: %#v", result.Layout.Rejected)
	}
	if result.Article.Type == ArticleTypeReport {
		t.Fatalf("article type = %q", result.Article.Type)
	}
}

func TestAnalyzeIgnoresFencedCodeBlocksForSemanticDetection(t *testing.T) {
	markdown := "# 标题\n\n```markdown\n## 代码里的标题\n1. 第一步\n2. 第二步\n> 代码里的引用\n点击下载\n增长 30%\n![cover](cover.png)\n```\n\n正文。"
	result, err := Analyze(Input{
		SourceFile: filepath.Join(t.TempDir(), "article.md"),
		Markdown:   markdown,
	})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if result.Article.Stats.H2Count != 0 || result.Article.Stats.OrderedListCount != 0 || result.Article.Stats.BlockquoteCount != 0 || result.Article.Stats.ImageCount != 0 {
		t.Fatalf("stats counted fenced content: %#v", result.Article.Stats)
	}
	for _, signal := range []string{"has_steps", "has_quotes", "has_cta", "has_metrics"} {
		if hasSignal(result.Article.Signals, signal) {
			t.Fatalf("unexpected signal %q in %#v", signal, result.Article.Signals)
		}
	}
	if len(result.Layout.Recommended) != 0 {
		t.Fatalf("unexpected layout recommendations from fenced content: %#v", result.Layout.Recommended)
	}
}

func TestAnalyzeCTAAllowsNoRegistrationCopy(t *testing.T) {
	result, err := Analyze(Input{
		SourceFile: filepath.Join(t.TempDir(), "article.md"),
		Markdown:   "# 标题\n\n无需注册，点击下载完整清单。",
	})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if !hasSignal(result.Article.Signals, "has_cta") {
		t.Fatalf("signals = %#v", result.Article.Signals)
	}
	module := findModule(result.Layout.Recommended, "cta")
	if module == nil {
		t.Fatalf("missing cta recommendation: %#v", result.Layout.Recommended)
	}
	if module.State != ActionStateRecommended {
		t.Fatalf("cta state = %q", module.State)
	}
}

func TestAnalyzeOmitsBodyRelativeEvidenceLineNumbersAfterFrontmatter(t *testing.T) {
	markdown := "---\ntitle: Frontmatter Title\n---\n\n## 背景\n\n正文。\n\n## 方案\n\n正文。\n\n## 结果\n\n正文。"
	result, err := Analyze(Input{
		SourceFile: filepath.Join(t.TempDir(), "article.md"),
		Markdown:   markdown,
	})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	module := findModule(result.Layout.Recommended, "toc")
	if module == nil {
		t.Fatalf("missing toc recommendation: %#v", result.Layout.Recommended)
	}
	if len(module.Evidence) == 0 {
		t.Fatalf("toc evidence = %#v", module.Evidence)
	}
	if module.Evidence[0].Location != nil && module.Evidence[0].Location.Line != 0 {
		t.Fatalf("misleading body-relative evidence location: %#v", module.Evidence[0].Location)
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

func TestAnalyzeKeepsMicroEditsSeparateFromCommandActions(t *testing.T) {
	result, err := Analyze(Input{
		SourceFile: filepath.Join(t.TempDir(), "article.md"),
		Markdown:   "# 标题\n\n第一段。\n\n第二段。\n\n第三段。\n\n第四段。",
	})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(result.MicroEdits) == 0 {
		t.Fatalf("expected micro edits")
	}
	if findAction(result.Actions, ToolMicroEdit) != nil {
		t.Fatalf("micro_edit should live in micro_edits, not command actions: %#v", result.Actions)
	}
	if !result.MicroEdits[0].RequiresConfirmation {
		t.Fatalf("micro edit should require confirmation: %#v", result.MicroEdits[0])
	}
}

func TestAnalyzeDoesNotRouteFormulaicPhrasesToHumanize(t *testing.T) {
	result, err := Analyze(Input{
		SourceFile: filepath.Join(t.TempDir(), "article.md"),
		Markdown: "# 标题\n\n值得注意的是，这里有常见过渡句。\n\n" +
			"从某种程度上，这些短语本身不足以证明文章需要去痕。",
	})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if findAction(result.Actions, "humanize") != nil {
		t.Fatalf("formulaic phrases should not create a humanize action: %#v", result.Actions)
	}
	for _, edit := range result.MicroEdits {
		if edit.Kind == "reduce_formulaic_language" {
			t.Fatalf("formulaic phrases should not create humanize micro edit: %#v", result.MicroEdits)
		}
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

func hasSignal(signals []string, name string) bool {
	for _, signal := range signals {
		if signal == name {
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
