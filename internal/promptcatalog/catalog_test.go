package promptcatalog

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestDefaultCatalogLoadsBuiltinPrompts(t *testing.T) {
	ResetDefaultCatalogForTests()
	t.Chdir(t.TempDir())

	cat, err := DefaultCatalog()
	if err != nil {
		t.Fatalf("DefaultCatalog() error = %v", err)
	}

	spec, err := cat.Get("humanizer", "base")
	if err != nil {
		t.Fatalf("Get(humanizer, base) error = %v", err)
	}
	if spec.Kind != "humanizer" || spec.Name != "base" {
		t.Fatalf("unexpected spec: %#v", spec)
	}
}

func TestCatalogRenderReplacesVariables(t *testing.T) {
	ResetDefaultCatalogForTests()
	t.Chdir(t.TempDir())

	cat, err := DefaultCatalog()
	if err != nil {
		t.Fatalf("DefaultCatalog() error = %v", err)
	}

	rendered, spec, err := cat.Render("image", "cover-default", map[string]string{
		"ARTICLE_TITLE":   "测试标题",
		"ARTICLE_SUMMARY": "测试摘要",
		"VISUAL_STYLE":    "极简",
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if spec.Name != "cover-default" {
		t.Fatalf("spec.Name = %q", spec.Name)
	}
	if !strings.Contains(rendered, "测试标题") || !strings.Contains(rendered, "极简") {
		t.Fatalf("rendered prompt = %q", rendered)
	}
}

func TestBuiltinTitlePromptIsDiscoverable(t *testing.T) {
	ResetDefaultCatalogForTests()
	t.Chdir(t.TempDir())

	cat, err := DefaultCatalog()
	if err != nil {
		t.Fatalf("DefaultCatalog() error = %v", err)
	}

	spec, err := cat.Get("title", "wechat-title-expert")
	if err != nil {
		t.Fatalf("Get(title, wechat-title-expert) error = %v", err)
	}
	if spec.Kind != "title" || spec.Name != "wechat-title-expert" {
		t.Fatalf("unexpected spec: %#v", spec)
	}

	prompts := cat.List("title")
	if len(prompts) == 0 {
		t.Fatal("expected title prompts in catalog list")
	}
	found := false
	for _, prompt := range prompts {
		if prompt.Name == "wechat-title-expert" {
			found = true
		}
		if prompt.Kind != "title" {
			t.Fatalf("unexpected prompt in title list: %#v", prompt)
		}
	}
	if !found {
		t.Fatalf("title list missing wechat-title-expert: %#v", prompts)
	}
}

func TestBuiltinTitlePromptRendersArticleContentAndJSONInstruction(t *testing.T) {
	ResetDefaultCatalogForTests()
	t.Chdir(t.TempDir())

	cat, err := DefaultCatalog()
	if err != nil {
		t.Fatalf("DefaultCatalog() error = %v", err)
	}

	rendered, spec, err := cat.Render("title", "wechat-title-expert", map[string]string{
		"ARTICLE_CONTENT":  "这篇文章复盘了一个公众号标题实验，重点是用真实价值降低标题党风险。",
		"TARGET_READER":    "内容创作者",
		"TITLE_COUNT":      "8",
		"MAX_TITLE_CHARS":  "22",
		"HOOK_LEVEL":       "3",
		"HOOK_LEVEL_LABEL": "high_tension",
	})
	if err != nil {
		t.Fatalf("Render(title, wechat-title-expert) error = %v", err)
	}
	if spec.Name != "wechat-title-expert" {
		t.Fatalf("spec.Name = %q", spec.Name)
	}
	if !strings.Contains(rendered, "这篇文章复盘了一个公众号标题实验") {
		t.Fatalf("rendered prompt missing article content: %q", rendered)
	}
	if !strings.Contains(rendered, "strict JSON only") || !strings.Contains(rendered, "不要输出 Markdown") {
		t.Fatalf("rendered prompt missing JSON-only instruction: %q", rendered)
	}
	if !strings.Contains(rendered, `"article_summary"`) || !strings.Contains(rendered, `"weighted_score"`) {
		t.Fatalf("rendered prompt missing expected JSON fields: %q", rendered)
	}
	if !containsString(spec.Variables, "HOOK_LEVEL") {
		t.Fatalf("title prompt variables missing HOOK_LEVEL: %#v", spec.Variables)
	}
	if !containsString(spec.Variables, "HOOK_LEVEL_LABEL") {
		t.Fatalf("title prompt variables missing HOOK_LEVEL_LABEL: %#v", spec.Variables)
	}
	for _, want := range []string{
		"标题钩子力度：3（high_tension",
		`"hook_level": 3`,
		`"hook_level_label"`,
		`"claim_strength"`,
		`"evidence_basis"`,
		`"risk_flags"`,
		"Level 3",
		"evidence_basis 必须非空",
		"risk_flags 必须存在",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered prompt missing %q: %q", want, rendered)
		}
	}
}

func TestCatalogPrefersExplicitPromptDirOverBuiltin(t *testing.T) {
	ResetDefaultCatalogForTests()
	tmpDir := t.TempDir()
	overrideDir := filepath.Join(tmpDir, "prompts", "humanizer")
	if err := os.MkdirAll(overrideDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	override := strings.Join([]string{
		"name: medium",
		"kind: humanizer",
		"description: override",
		"version: \"1.0\"",
		"template: |",
		"  override medium",
	}, "\n")
	if err := os.WriteFile(filepath.Join(overrideDir, "medium.yaml"), []byte(override), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv(promptsDirEnvVar, filepath.Join(tmpDir, "prompts"))

	cat := NewCatalog()
	if err := cat.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	spec, err := cat.Get("humanizer", "medium")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if spec.Source != filepath.Join(tmpDir, "prompts") {
		t.Fatalf("Source = %q", spec.Source)
	}
	if strings.TrimSpace(spec.Template) != "override medium" {
		t.Fatalf("Template = %q", spec.Template)
	}
}

func TestCatalogListFilteredByArchetypeAndTag(t *testing.T) {
	ResetDefaultCatalogForTests()
	t.Chdir(t.TempDir())

	cat, err := DefaultCatalog()
	if err != nil {
		t.Fatalf("DefaultCatalog() error = %v", err)
	}

	prompts := cat.ListFiltered(ListFilter{
		Kind:      "image",
		Archetype: "cover",
		Tag:       "editorial",
	})
	if len(prompts) == 0 {
		t.Fatal("expected filtered image prompts")
	}
	for _, prompt := range prompts {
		if prompt.Kind != "image" {
			t.Fatalf("unexpected kind: %#v", prompt)
		}
		if prompt.Archetype != "cover" {
			t.Fatalf("unexpected archetype: %#v", prompt)
		}
		if !containsTag(prompt.Tags, "editorial") {
			t.Fatalf("expected editorial tag: %#v", prompt)
		}
	}
}

func TestBuiltinImagePromptIncludesArchetypeMetadata(t *testing.T) {
	ResetDefaultCatalogForTests()
	t.Chdir(t.TempDir())

	cat, err := DefaultCatalog()
	if err != nil {
		t.Fatalf("DefaultCatalog() error = %v", err)
	}

	spec, err := cat.Get("image", "cover-hero")
	if err != nil {
		t.Fatalf("Get(image, cover-hero) error = %v", err)
	}
	if spec.Archetype != "cover" {
		t.Fatalf("Archetype = %q", spec.Archetype)
	}
	if !containsTag(spec.Tags, "hero") {
		t.Fatalf("Tags = %#v", spec.Tags)
	}
	if len(spec.Examples) == 0 {
		t.Fatalf("Examples = %#v", spec.Examples)
	}
	if spec.Metadata["author"] != "geekjourneyx" {
		t.Fatalf("Metadata = %#v", spec.Metadata)
	}
}

func TestBuiltinImagePromptSupportsAttributionMetadata(t *testing.T) {
	ResetDefaultCatalogForTests()
	t.Chdir(t.TempDir())

	cat, err := DefaultCatalog()
	if err != nil {
		t.Fatalf("DefaultCatalog() error = %v", err)
	}

	spec, err := cat.Get("image", "infographic-flat-vector-panorama")
	if err != nil {
		t.Fatalf("Get(image, infographic-flat-vector-panorama) error = %v", err)
	}
	if spec.Archetype != "infographic" {
		t.Fatalf("Archetype = %q", spec.Archetype)
	}
	if spec.Metadata["author"] != "geekjourneyx" {
		t.Fatalf("author metadata = %#v", spec.Metadata)
	}
	if spec.Metadata["inspired_by"] != "op7418" {
		t.Fatalf("inspired_by metadata = %#v", spec.Metadata)
	}
	if spec.Metadata["provenance"] != "adapted" {
		t.Fatalf("provenance metadata = %#v", spec.Metadata)
	}
}

func TestBuiltinDarkTicketInfographicPromptIsDiscoverable(t *testing.T) {
	ResetDefaultCatalogForTests()
	t.Chdir(t.TempDir())

	cat, err := DefaultCatalog()
	if err != nil {
		t.Fatalf("DefaultCatalog() error = %v", err)
	}

	spec, err := cat.Get("image", "infographic-dark-ticket-cn")
	if err != nil {
		t.Fatalf("Get(image, infographic-dark-ticket-cn) error = %v", err)
	}
	if spec.Archetype != "infographic" {
		t.Fatalf("Archetype = %q", spec.Archetype)
	}
	if !containsTag(spec.Tags, "ticket") || !containsTag(spec.Tags, "dark") {
		t.Fatalf("Tags = %#v", spec.Tags)
	}
	if spec.Metadata["author"] != "geekjourneyx" || spec.Metadata["provenance"] != "original" {
		t.Fatalf("Metadata = %#v", spec.Metadata)
	}
	if !strings.Contains(spec.Template, "21:9") {
		t.Fatalf("Template = %q", spec.Template)
	}
}

func TestBuiltinHanddrawnSketchnotePromptIsDiscoverable(t *testing.T) {
	ResetDefaultCatalogForTests()
	t.Chdir(t.TempDir())

	cat, err := DefaultCatalog()
	if err != nil {
		t.Fatalf("DefaultCatalog() error = %v", err)
	}

	spec, err := cat.Get("image", "infographic-handdrawn-sketchnote")
	if err != nil {
		t.Fatalf("Get(image, infographic-handdrawn-sketchnote) error = %v", err)
	}
	if spec.Archetype != "infographic" {
		t.Fatalf("Archetype = %q", spec.Archetype)
	}
	if !containsTag(spec.Tags, "sketchnote") || !containsTag(spec.Tags, "handdrawn") {
		t.Fatalf("Tags = %#v", spec.Tags)
	}
	if spec.Metadata["author"] != "geekjourneyx" || spec.Metadata["provenance"] != "original" {
		t.Fatalf("Metadata = %#v", spec.Metadata)
	}
	if !strings.Contains(spec.Template, "图文并茂") {
		t.Fatalf("Template = %q", spec.Template)
	}
}

func TestBuiltinVictorianEngravingBannerPromptIsDiscoverable(t *testing.T) {
	ResetDefaultCatalogForTests()
	t.Chdir(t.TempDir())

	cat, err := DefaultCatalog()
	if err != nil {
		t.Fatalf("DefaultCatalog() error = %v", err)
	}

	spec, err := cat.Get("image", "infographic-victorian-engraving-banner")
	if err != nil {
		t.Fatalf("Get(image, infographic-victorian-engraving-banner) error = %v", err)
	}
	if spec.Archetype != "infographic" {
		t.Fatalf("Archetype = %q", spec.Archetype)
	}
	if !containsTag(spec.Tags, "cover") || !containsTag(spec.Tags, "victorian") {
		t.Fatalf("Tags = %#v", spec.Tags)
	}
	if spec.Metadata["author"] != "geekjourneyx" || spec.Metadata["provenance"] != "original" {
		t.Fatalf("Metadata = %#v", spec.Metadata)
	}
	if !strings.Contains(spec.Template, "21:9") || !strings.Contains(spec.Template, "Gustave Doré") {
		t.Fatalf("Template = %q", spec.Template)
	}
}

func TestBuiltinImagePromptsHaveConsistentUseCaseAndAspectMetadata(t *testing.T) {
	ResetDefaultCatalogForTests()
	t.Chdir(t.TempDir())

	cat, err := DefaultCatalog()
	if err != nil {
		t.Fatalf("DefaultCatalog() error = %v", err)
	}

	prompts := cat.List("image")
	if len(prompts) == 0 {
		t.Fatal("expected builtin image prompts")
	}

	for _, spec := range prompts {
		if spec.PrimaryUseCase == "" {
			t.Fatalf("%s missing primary_use_case", spec.Name)
		}
		if spec.DefaultAspectRatio == "" {
			t.Fatalf("%s missing default_aspect_ratio", spec.Name)
		}
		if len(spec.RecommendedAspectRatios) == 0 {
			t.Fatalf("%s missing recommended_aspect_ratios", spec.Name)
		}
		if !containsAspectRatio(spec.RecommendedAspectRatios, spec.DefaultAspectRatio) {
			t.Fatalf("%s default_aspect_ratio %q not found in recommended_aspect_ratios %#v", spec.Name, spec.DefaultAspectRatio, spec.RecommendedAspectRatios)
		}
		if spec.Metadata["author"] == "" {
			t.Fatalf("%s missing metadata.author", spec.Name)
		}
		if spec.Metadata["provenance"] == "" {
			t.Fatalf("%s missing metadata.provenance", spec.Name)
		}
		if !SupportsUseCase(&spec, spec.PrimaryUseCase) {
			t.Fatalf("%s primary_use_case %q not supported by SupportsUseCase", spec.Name, spec.PrimaryUseCase)
		}
	}
}

func TestBuiltinPlanVisualAssetPromptsAreDiscoverable(t *testing.T) {
	ResetDefaultCatalogForTests()
	t.Chdir(t.TempDir())

	cat, err := DefaultCatalog()
	if err != nil {
		t.Fatalf("DefaultCatalog() error = %v", err)
	}

	tests := []struct {
		name          string
		archetype     string
		defaultAspect string
		tag           string
		marker        string
	}{
		{name: "cover-claude-warm", archetype: "cover", defaultAspect: "21:9", tag: "warm-editorial", marker: "羊皮纸"},
		{name: "cover-semantic-concept", archetype: "cover", defaultAspect: "21:9", tag: "semantic", marker: "文字嵌入构图"},
		{name: "cover-editorial-collage", archetype: "cover", defaultAspect: "21:9", tag: "collage", marker: "手撕拼贴"},
		{name: "cover-suspense-black-gold", archetype: "cover", defaultAspect: "2.35:1", tag: "suspense", marker: "#D4A84B"},
		{name: "infographic-claude-warm", archetype: "infographic", defaultAspect: "21:9", tag: "warm-editorial", marker: "最多 3 个"},
		{name: "infographic-ticket-process", archetype: "infographic", defaultAspect: "21:9", tag: "ticket-process", marker: "票券本身就是整张图片"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := cat.Get("image", tt.name)
			if err != nil {
				t.Fatalf("Get(image, %s) error = %v", tt.name, err)
			}
			if spec.Archetype != tt.archetype || spec.PrimaryUseCase != tt.archetype {
				t.Fatalf("archetype/use case = %q/%q, want %q", spec.Archetype, spec.PrimaryUseCase, tt.archetype)
			}
			if spec.DefaultAspectRatio != tt.defaultAspect {
				t.Fatalf("DefaultAspectRatio = %q, want %q", spec.DefaultAspectRatio, tt.defaultAspect)
			}
			if !containsAspectRatio(spec.RecommendedAspectRatios, tt.defaultAspect) {
				t.Fatalf("recommended ratios %#v missing default %q", spec.RecommendedAspectRatios, tt.defaultAspect)
			}
			if !containsTag(spec.Tags, tt.tag) {
				t.Fatalf("Tags = %#v, want %q", spec.Tags, tt.tag)
			}
			if spec.Metadata["author"] != "geekjourneyx" || spec.Metadata["provenance"] != "adapted" || spec.Metadata["inspired_by"] != "plan-visual-assets" {
				t.Fatalf("Metadata = %#v", spec.Metadata)
			}
			if !strings.Contains(spec.Template, tt.marker) {
				t.Fatalf("Template missing marker %q", tt.marker)
			}
		})
	}
}

func TestBuiltinImagePromptVariablesCloseOverGenerationContract(t *testing.T) {
	ResetDefaultCatalogForTests()
	t.Chdir(t.TempDir())

	cat, err := DefaultCatalog()
	if err != nil {
		t.Fatalf("DefaultCatalog() error = %v", err)
	}

	renderVars := map[string]string{
		"ARTICLE_TITLE":   "测试标题",
		"ARTICLE_SUMMARY": "测试摘要",
		"KEYWORDS":        "测试,提示词",
		"KEY_POINTS":      "第一点；第二点；第三点",
		"VISUAL_STYLE":    "克制、清晰",
		"ASPECT_RATIO":    "21:9",
	}
	placeholderPattern := regexp.MustCompile(`\{\{([A-Z][A-Z0-9_]*)\}\}`)

	for _, spec := range cat.List("image") {
		declared := make(map[string]bool, len(spec.Variables))
		for _, variable := range spec.Variables {
			declared[variable] = true
			if _, ok := renderVars[variable]; !ok {
				t.Errorf("%s declares unsupported generate_image variable %q", spec.Name, variable)
			}
		}

		for _, match := range placeholderPattern.FindAllStringSubmatch(spec.Template, -1) {
			if !declared[match[1]] {
				t.Errorf("%s uses undeclared template variable %q", spec.Name, match[1])
			}
		}

		rendered, _, err := cat.Render("image", spec.Name, renderVars)
		if err != nil {
			t.Errorf("Render(image, %s) error = %v", spec.Name, err)
			continue
		}
		if unresolved := placeholderPattern.FindString(rendered); unresolved != "" {
			t.Errorf("Render(image, %s) left unresolved placeholder %q", spec.Name, unresolved)
		}
	}
}

func containsAspectRatio(ratios []string, target string) bool {
	for _, ratio := range ratios {
		if strings.EqualFold(strings.TrimSpace(ratio), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func TestBuiltinAppleKeynotePremiumPromptIsDiscoverable(t *testing.T) {
	ResetDefaultCatalogForTests()
	t.Chdir(t.TempDir())

	cat, err := DefaultCatalog()
	if err != nil {
		t.Fatalf("DefaultCatalog() error = %v", err)
	}

	spec, err := cat.Get("image", "infographic-apple-keynote-premium")
	if err != nil {
		t.Fatalf("Get(image, infographic-apple-keynote-premium) error = %v", err)
	}
	if spec.Archetype != "infographic" {
		t.Fatalf("Archetype = %q", spec.Archetype)
	}
	if spec.PrimaryUseCase != "infographic" {
		t.Fatalf("PrimaryUseCase = %q", spec.PrimaryUseCase)
	}
	if spec.DefaultAspectRatio != "3:4" {
		t.Fatalf("DefaultAspectRatio = %q", spec.DefaultAspectRatio)
	}
	if !containsTag(spec.Tags, "apple") || !containsTag(spec.Tags, "keynote") {
		t.Fatalf("Tags = %#v", spec.Tags)
	}
	if !SupportsUseCase(spec, "cover") {
		t.Fatalf("expected preset to support cover use case: %#v", spec.CompatibleUseCases)
	}
	if spec.Metadata["author"] != "geekjourneyx" || spec.Metadata["provenance"] != "original" {
		t.Fatalf("Metadata = %#v", spec.Metadata)
	}
	if !strings.Contains(spec.Template, "Apple keynote aesthetic") || !strings.Contains(spec.Template, "3:4") {
		t.Fatalf("Template = %q", spec.Template)
	}
}
