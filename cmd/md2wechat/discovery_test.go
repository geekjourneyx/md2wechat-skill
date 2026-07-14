package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"github.com/geekjourneyx/md2wechat-skill/internal/converter"
	"github.com/geekjourneyx/md2wechat-skill/internal/image"
	"github.com/geekjourneyx/md2wechat-skill/internal/layoutcatalog"
	"github.com/geekjourneyx/md2wechat-skill/internal/promptcatalog"
	titlebuilder "github.com/geekjourneyx/md2wechat-skill/internal/title"
	"github.com/spf13/cobra"
)

const (
	providerListBudget = 4 * 1024
	themeListBudget    = 12 * 1024
	promptListBudget   = 20 * 1024
)

func TestDiscoveryListItemsExcludeHeavyDetail(t *testing.T) {
	tests := []struct {
		name      string
		value     any
		forbidden []string
	}{
		{
			name:      "provider",
			value:     providerToListItem(providerView{Name: "openai", DefaultBaseURL: "https://example.com", SupportedModels: []image.ProviderModelMeta{{Name: "model"}}}),
			forbidden: []string{"default_base_url", "supported_models", "required_config", "optional_config"},
		},
		{
			name:      "theme",
			value:     themeToListItem(themeView{Name: "default", Style: converter.ThemeStyle{}}),
			forbidden: []string{"style"},
		},
		{
			name:      "prompt",
			value:     promptToListItem(promptcatalog.PromptSpec{Name: "cover", Kind: "image", Template: "large template", Metadata: map[string]string{"author": "x"}}),
			forbidden: []string{"template", "metadata", "examples", "source"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatal(err)
			}
			for _, field := range tt.forbidden {
				if bytes.Contains(encoded, []byte(`"`+field+`"`)) {
					t.Fatalf("list item leaked %q: %s", field, encoded)
				}
			}
		})
	}
}

func TestDiscoveryShowViewsRetainFullDetail(t *testing.T) {
	providerJSON, err := json.Marshal(providerView{DefaultBaseURL: "https://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(providerJSON, []byte(`"default_base_url"`)) {
		t.Fatal("provider show view lost full detail")
	}

	themeJSON, err := json.Marshal(themeView{Style: converter.ThemeStyle{}})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(themeJSON, []byte(`"style"`)) {
		t.Fatal("theme show view lost style detail")
	}

	promptJSON, err := json.Marshal(promptcatalog.PromptSpec{
		Template: "large template",
		Metadata: map[string]string{"author": "x"},
		Examples: []string{"example"},
		Source:   "builtin",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"template", "metadata", "examples", "source"} {
		if !bytes.Contains(promptJSON, []byte(`"`+field+`"`)) {
			t.Fatalf("prompt show view lost full detail field %q: %s", field, promptJSON)
		}
	}
}

func TestDiscoveryListCommandsExcludeHeavyDetailAndStayWithinBudget(t *testing.T) {
	oldCfg := cfg
	oldJSON := jsonOutput
	oldPromptKind := promptKind
	oldPromptArchetype := promptArchetype
	oldPromptTag := promptTag
	t.Cleanup(func() {
		cfg = oldCfg
		jsonOutput = oldJSON
		promptKind = oldPromptKind
		promptArchetype = oldPromptArchetype
		promptTag = oldPromptTag
		promptcatalog.ResetDefaultCatalogForTests()
	})

	cfg = &config.Config{DefaultTheme: "default", ImageProvider: "openai"}
	jsonOutput = true
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MD2WECHAT_THEMES_DIR", "")
	t.Setenv("MD2WECHAT_PROMPTS_DIR", "")
	promptKind = ""
	promptArchetype = ""
	promptTag = ""
	promptcatalog.ResetDefaultCatalogForTests()

	tests := []struct {
		name       string
		command    *cobra.Command
		payloadKey string
		budget     int
		forbidden  []string
	}{
		{
			name:       "providers",
			command:    providersListCmd,
			payloadKey: "providers",
			budget:     providerListBudget,
			forbidden:  []string{"default_base_url", "supported_models", "required_config", "optional_config"},
		},
		{
			name:       "themes",
			command:    themesListCmd,
			payloadKey: "themes",
			budget:     themeListBudget,
			forbidden:  []string{"style"},
		},
		{
			name:       "prompts",
			command:    promptsListCmd,
			payloadKey: "prompts",
			budget:     promptListBudget,
			forbidden:  []string{"template", "metadata", "examples", "source"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout := captureStdout(t, func() {
				if err := tt.command.RunE(tt.command, nil); err != nil {
					t.Fatalf("RunE() error = %v", err)
				}
			})

			var response map[string]any
			if err := json.Unmarshal(stdout, &response); err != nil {
				t.Fatalf("unmarshal response: %v\n%s", err, stdout)
			}
			data, ok := response["data"].(map[string]any)
			if !ok {
				t.Fatalf("data type = %T", response["data"])
			}
			items, ok := data[tt.payloadKey].([]any)
			if !ok || len(items) == 0 {
				t.Fatalf("%s payload = %#v", tt.payloadKey, data[tt.payloadKey])
			}
			encoded, err := json.Marshal(items)
			if err != nil {
				t.Fatal(err)
			}
			for _, field := range tt.forbidden {
				if bytes.Contains(encoded, []byte(`"`+field+`"`)) {
					t.Fatalf("list payload leaked %q: %s", field, encoded)
				}
			}
			if len(encoded) > tt.budget {
				t.Fatalf("list payload size = %d bytes, budget = %d", len(encoded), tt.budget)
			}
		})
	}
}

func TestBuildProviderViewsIncludesBuiltinProviders(t *testing.T) {
	oldCfg := cfg
	t.Cleanup(func() { cfg = oldCfg })

	cfg = nil
	providers, err := buildProviderViews()
	if err != nil {
		t.Fatalf("buildProviderViews() error = %v", err)
	}
	if len(providers) == 0 {
		t.Fatal("expected providers")
	}
	found := false
	for _, provider := range providers {
		if provider.Name == "agent" || contains(provider.Aliases, "agent") {
			t.Fatalf("agent must not be exposed as an image provider: %#v", provider)
		}
		if provider.Name == "openai" {
			found = true
			if !provider.SupportsSize {
				t.Fatalf("expected openai SupportsSize")
			}
			if provider.DefaultModel != "gpt-image-2" {
				t.Fatalf("openai default model = %q, want gpt-image-2", provider.DefaultModel)
			}
			if len(provider.SupportedModels) == 0 {
				t.Fatal("expected openai supported models")
			}
			if provider.SupportedModels[0].Name != "gpt-image-2" || !provider.SupportedModels[0].Default {
				t.Fatalf("unexpected openai supported models: %#v", provider.SupportedModels)
			}
		}
	}
	if !found {
		t.Fatal("expected openai provider")
	}
}

func TestCapabilitiesIncludeImagePlanMode(t *testing.T) {
	oldCfg := cfg
	t.Cleanup(func() { cfg = oldCfg })

	cfg = &config.Config{DefaultTheme: "default"}
	data, err := buildCapabilitiesData()
	if err != nil {
		t.Fatalf("buildCapabilitiesData() error = %v", err)
	}

	imageGeneration, ok := data["image_generation"].(map[string]any)
	if !ok {
		t.Fatalf("image_generation type = %T", data["image_generation"])
	}

	wantCommands := []string{"generate_image", "generate_cover", "generate_infographic"}

	directProvider, ok := imageGeneration["direct_provider"].(map[string]any)
	if !ok {
		t.Fatalf("direct_provider type = %T", imageGeneration["direct_provider"])
	}
	if directProvider["available"] != true {
		t.Fatalf("direct_provider available = %#v", directProvider["available"])
	}
	if directProvider["requires_provider"] != true {
		t.Fatalf("direct_provider requires_provider = %#v", directProvider["requires_provider"])
	}
	if directProvider["requires_image_api_key"] != true {
		t.Fatalf("direct_provider requires_image_api_key = %#v", directProvider["requires_image_api_key"])
	}
	if directProvider["side_effects"] != true {
		t.Fatalf("direct_provider side_effects = %#v", directProvider["side_effects"])
	}
	directCommands, ok := directProvider["commands"].([]string)
	if !ok {
		t.Fatalf("direct_provider commands type = %T", directProvider["commands"])
	}
	for _, command := range wantCommands {
		if !contains(directCommands, command) {
			t.Fatalf("direct_provider commands missing %s: %#v", command, directCommands)
		}
	}

	planMode, ok := imageGeneration["plan_mode"].(map[string]any)
	if !ok {
		t.Fatalf("plan_mode type = %T", imageGeneration["plan_mode"])
	}
	if planMode["available"] != true {
		t.Fatalf("plan_mode available = %#v", planMode["available"])
	}
	if planMode["requires_provider"] != false {
		t.Fatalf("plan_mode requires_provider = %#v", planMode["requires_provider"])
	}
	if planMode["requires_image_api_key"] != false {
		t.Fatalf("plan_mode requires_image_api_key = %#v", planMode["requires_image_api_key"])
	}
	if planMode["side_effects"] != false {
		t.Fatalf("plan_mode side_effects = %#v", planMode["side_effects"])
	}
	if planMode["execution_owner"] != "host_agent" {
		t.Fatalf("plan_mode execution_owner = %#v", planMode["execution_owner"])
	}
	if planMode["requires_json"] != true {
		t.Fatalf("plan_mode requires_json = %#v", planMode["requires_json"])
	}
	if planMode["response_code"] != codeImagePlanReady {
		t.Fatalf("plan_mode response_code = %#v, want %s", planMode["response_code"], codeImagePlanReady)
	}
	planCommands, ok := planMode["commands"].([]string)
	if !ok {
		t.Fatalf("plan_mode commands type = %T", planMode["commands"])
	}
	for _, command := range wantCommands {
		if !contains(planCommands, command) {
			t.Fatalf("plan_mode commands missing %s: %#v", command, planCommands)
		}
	}
}

func TestBuildProviderViewsUsesCurrentRuntimeDefaults(t *testing.T) {
	oldCfg := cfg
	t.Cleanup(func() { cfg = oldCfg })

	cfg = nil
	providers, err := buildProviderViews()
	if err != nil {
		t.Fatalf("buildProviderViews() error = %v", err)
	}

	defaults := map[string]string{
		"openrouter": "google/gemini-3-pro-image-preview",
		"gemini":     "gemini-3.1-flash-image-preview",
		"volcengine": "doubao-seedream-5-0-pro-260628",
	}

	for name, wantModel := range defaults {
		found := false
		for _, provider := range providers {
			if provider.Name != name {
				continue
			}
			found = true
			if provider.DefaultModel != wantModel {
				t.Fatalf("%s default model = %q, want %q", name, provider.DefaultModel, wantModel)
			}
			if len(provider.SupportedModels) == 0 {
				t.Fatalf("expected %s supported models", name)
			}
		}
		if !found {
			t.Fatalf("expected %s provider", name)
		}
	}
}

func TestListThemesIncludesBuiltinTheme(t *testing.T) {
	themes, err := listThemes()
	if err != nil {
		t.Fatalf("listThemes() error = %v", err)
	}
	found := false
	for _, theme := range themes {
		if theme.Name == "default" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected builtin default theme")
	}
}

func TestListThemeViewsExposeSelectionMetadata(t *testing.T) {
	themes, err := listThemeViews()
	if err != nil {
		t.Fatalf("listThemeViews() error = %v", err)
	}

	found := false
	for _, theme := range themes {
		if theme.Name != "minimal-blue" {
			continue
		}
		found = true
		if theme.Type != "api" {
			t.Fatalf("Type = %q, want api", theme.Type)
		}
		if !theme.Selectable {
			t.Fatal("expected minimal-blue selectable")
		}
		if theme.Style.Series != "minimal" {
			t.Fatalf("Style.Series = %q, want minimal", theme.Style.Series)
		}
		if theme.Style.Color != "blue" {
			t.Fatalf("Style.Color = %q, want blue", theme.Style.Color)
		}
	}
	if !found {
		t.Fatal("expected minimal-blue theme view")
	}
}

func TestListThemeViewsExposeExpandedAPICollectionThemes(t *testing.T) {
	themes, err := listThemeViews()
	if err != nil {
		t.Fatalf("listThemeViews() error = %v", err)
	}

	selectableAPIThemes := 0
	selectableFeaturedThemes := 0
	want := map[string]string{
		"elegant-green":   "elegant",
		"sspai-red":       "featured",
		"wechat-native":   "featured",
		"nyt-classic":     "featured",
		"github-readme":   "featured",
		"mint-fresh":      "featured",
		"sunset-amber":    "featured",
		"ink-minimal":     "featured",
		"lavender-dream":  "featured",
		"coffee-house":    "featured",
		"bauhaus-primary": "featured",
	}
	for _, theme := range themes {
		if theme.Type == "api" && theme.Selectable {
			selectableAPIThemes++
			if theme.Style.Series == "featured" {
				selectableFeaturedThemes++
			}
		}

		series, ok := want[theme.Name]
		if !ok {
			continue
		}
		if theme.Type != "api" || !theme.Selectable {
			t.Fatalf("unexpected expanded theme metadata for %s: %#v", theme.Name, theme)
		}
		if theme.APITheme != theme.Name {
			t.Fatalf("APITheme = %q, want %q", theme.APITheme, theme.Name)
		}
		if theme.Style.Series != series {
			t.Fatalf("%s Style.Series = %q, want %q", theme.Name, theme.Style.Series, series)
		}
		delete(want, theme.Name)
	}
	if len(want) != 0 {
		t.Fatalf("missing expanded API collection themes: %#v", want)
	}
	if selectableAPIThemes != 48 {
		t.Fatalf("selectable API theme count = %d, want 48", selectableAPIThemes)
	}
	if selectableFeaturedThemes != 10 {
		t.Fatalf("selectable featured theme count = %d, want 10", selectableFeaturedThemes)
	}
}

func TestListThemeViewsMarksAPICollectionNotSelectable(t *testing.T) {
	themes, err := listThemeViews()
	if err != nil {
		t.Fatalf("listThemeViews() error = %v", err)
	}

	found := false
	for _, theme := range themes {
		if theme.Name != "api-collection" {
			continue
		}
		found = true
		if theme.Selectable {
			t.Fatal("expected api-collection not selectable")
		}
	}
	if !found {
		t.Fatal("expected api-collection theme view")
	}
}

func TestBuildCapabilitiesDataIncludesPromptCatalog(t *testing.T) {
	oldCfg := cfg
	t.Cleanup(func() {
		cfg = oldCfg
		promptcatalog.ResetDefaultCatalogForTests()
	})

	cfg = nil
	promptcatalog.ResetDefaultCatalogForTests()

	data, err := buildCapabilitiesData()
	if err != nil {
		t.Fatalf("buildCapabilitiesData() error = %v", err)
	}
	prompts, ok := data["prompts"].([]promptcatalog.PromptSpec)
	if !ok || len(prompts) == 0 {
		t.Fatalf("expected prompt catalog in capabilities: %#v", data["prompts"])
	}
	archetypes, ok := data["prompt_archetypes"].([]string)
	if !ok || len(archetypes) == 0 {
		t.Fatalf("expected prompt archetypes in capabilities: %#v", data["prompt_archetypes"])
	}
}

func TestBuildCapabilitiesDataKeepsConvertContractStableWithInspectAndPreview(t *testing.T) {
	oldCfg := cfg
	t.Cleanup(func() { cfg = oldCfg })

	cfg = &config.Config{DefaultTheme: "default"}
	data, err := buildCapabilitiesData()
	if err != nil {
		t.Fatalf("buildCapabilitiesData() error = %v", err)
	}

	commands, ok := data["commands"].([]string)
	if !ok {
		t.Fatalf("commands type = %T", data["commands"])
	}
	if !contains(commands, "inspect") || !contains(commands, "preview") || !contains(commands, "convert") {
		t.Fatalf("commands = %#v", commands)
	}

	convertData, ok := data["convert"].(map[string]any)
	if !ok {
		t.Fatalf("convert type = %T", data["convert"])
	}
	if convertData["default_mode"] != "api" {
		t.Fatalf("default_mode = %#v", convertData["default_mode"])
	}
	if convertData["default_theme"] != "default" {
		t.Fatalf("default_theme = %#v", convertData["default_theme"])
	}
	backgroundTypes, ok := convertData["background_types"].([]string)
	if !ok {
		t.Fatalf("background_types type = %T", convertData["background_types"])
	}
	if len(backgroundTypes) != 3 || backgroundTypes[0] != "default" || backgroundTypes[1] != "grid" || backgroundTypes[2] != "none" {
		t.Fatalf("background_types = %#v", backgroundTypes)
	}
}

func TestBuildCapabilitiesDataDerivesCommandsFromRootManifest(t *testing.T) {
	oldCfg := cfg
	t.Cleanup(func() { cfg = oldCfg })

	cfg = &config.Config{DefaultTheme: "default"}

	data, err := buildCapabilitiesData()
	if err != nil {
		t.Fatalf("buildCapabilitiesData() error = %v", err)
	}

	commands, ok := data["commands"].([]string)
	if !ok {
		t.Fatalf("commands type = %T", data["commands"])
	}

	for _, want := range []string{
		"convert",
		"inspect",
		"advise",
		"preview",
		"config",
		"write",
		"humanize",
		"title",
		"upload_image",
		"download_and_upload",
		"generate_image",
		"generate_cover",
		"generate_infographic",
		"create_draft",
		"create_image_post",
		"test-draft",
		"providers",
		"themes",
		"prompts",
		"layout",
		"brand",
		"doctor",
		"skills",
		"capabilities",
		"version",
	} {
		if !contains(commands, want) {
			t.Fatalf("commands missing %q: %#v", want, commands)
		}
	}
}

func TestBuildCapabilitiesDataIncludesTitleGenerationCapability(t *testing.T) {
	oldCfg := cfg
	t.Cleanup(func() {
		cfg = oldCfg
		promptcatalog.ResetDefaultCatalogForTests()
	})

	cfg = &config.Config{DefaultTheme: "default"}
	promptcatalog.ResetDefaultCatalogForTests()

	data, err := buildCapabilitiesData()
	if err != nil {
		t.Fatalf("buildCapabilitiesData() error = %v", err)
	}

	commands, ok := data["commands"].([]string)
	if !ok {
		t.Fatalf("commands type = %T", data["commands"])
	}
	if !contains(commands, "title") {
		t.Fatalf("commands missing title: %#v", commands)
	}

	promptKinds, ok := data["prompt_kinds"].([]string)
	if !ok {
		t.Fatalf("prompt_kinds type = %T", data["prompt_kinds"])
	}
	if !contains(promptKinds, titlebuilder.PromptKind) {
		t.Fatalf("prompt_kinds missing %q: %#v", titlebuilder.PromptKind, promptKinds)
	}

	titleGeneration, ok := data["title_generation"].(map[string]any)
	if !ok {
		t.Fatalf("title_generation type = %T", data["title_generation"])
	}

	wantValues := map[string]any{
		"available":                       true,
		"command":                         "title suggest",
		"prompt_kind":                     titlebuilder.PromptKind,
		"default_prompt":                  titlebuilder.DefaultPromptName,
		"action":                          "ai_title_suggestion_request",
		"mode":                            "ai_request_host_agent_handoff",
		"execution_owner":                 "host_agent",
		"side_effects":                    false,
		"requires_external_model":         true,
		"requires_json":                   true,
		"requires_provider":               false,
		"requires_image_api_key":          false,
		"requires_wechat_credentials":     false,
		"response_code":                   codeTitleSuggestRequestReady,
		"default_max_title_chars":         titlebuilder.DefaultMaxTitleChars,
		"metadata_title_max_chars":        titlebuilder.MetadataTitleMaxChars,
		"default_hook_level":              titlebuilder.DefaultHookLevel,
		"max_recommended_hook_level":      2,
		"level_3_requires_evidence_basis": true,
		"recommendation_only":             true,
	}
	for key, want := range wantValues {
		if titleGeneration[key] != want {
			t.Fatalf("title_generation[%s] = %#v, want %#v", key, titleGeneration[key], want)
		}
	}

	candidateCount, ok := titleGeneration["candidate_count"].(map[string]any)
	if !ok {
		t.Fatalf("candidate_count type = %T", titleGeneration["candidate_count"])
	}
	for key, want := range map[string]int{
		"min":     titlebuilder.MinCount,
		"max":     titlebuilder.MaxCount,
		"default": titlebuilder.DefaultCount,
	} {
		if candidateCount[key] != want {
			t.Fatalf("candidate_count[%s] = %#v, want %d", key, candidateCount[key], want)
		}
	}

	hookLevels, ok := titleGeneration["hook_levels"].([]map[string]any)
	if !ok {
		t.Fatalf("hook_levels type = %T", titleGeneration["hook_levels"])
	}
	if len(hookLevels) != 3 {
		t.Fatalf("hook_levels length = %d, want 3: %#v", len(hookLevels), hookLevels)
	}
	for i, want := range []struct {
		level int
		label string
	}{
		{level: 1, label: "restrained"},
		{level: 2, label: "punchy"},
		{level: 3, label: "high_tension"},
	} {
		if hookLevels[i]["level"] != want.level || hookLevels[i]["label"] != want.label {
			t.Fatalf("hook_levels[%d] = %#v, want level=%d label=%s", i, hookLevels[i], want.level, want.label)
		}
		if hookLevels[i]["description"] == "" {
			t.Fatalf("hook_levels[%d] missing description: %#v", i, hookLevels[i])
		}
	}
}

func TestCapabilitiesIncludesArticleAdvice(t *testing.T) {
	oldCfg := cfg
	t.Cleanup(func() { cfg = oldCfg })

	cfg = &config.Config{DefaultTheme: "default"}
	data, err := buildCapabilitiesData()
	if err != nil {
		t.Fatalf("buildCapabilitiesData() error = %v", err)
	}

	commands, ok := data["commands"].([]string)
	if !ok {
		t.Fatalf("commands type = %T", data["commands"])
	}
	if !contains(commands, "advise") {
		t.Fatalf("commands missing advise: %#v", commands)
	}

	articleAdvice, ok := data["article_advice"].(map[string]any)
	if !ok {
		t.Fatalf("article_advice type = %T", data["article_advice"])
	}
	wantValues := map[string]any{
		"available":          true,
		"command":            "advise",
		"requires_json":      true,
		"side_effects":       false,
		"deterministic":      true,
		"max_actions":        3,
		"max_layout_modules": 3,
		"response_code":      codeAdviseCompleted,
	}
	for key, want := range wantValues {
		if articleAdvice[key] != want {
			t.Fatalf("article_advice[%s] = %#v, want %#v", key, articleAdvice[key], want)
		}
	}
	tools, ok := articleAdvice["tools"].([]string)
	if !ok {
		t.Fatalf("tools type = %T", articleAdvice["tools"])
	}
	wantTools := []string{"title", "cover", "layout", "micro_edit"}
	if len(tools) != len(wantTools) {
		t.Fatalf("tools = %#v, want %#v", tools, wantTools)
	}
	for i := range wantTools {
		if tools[i] != wantTools[i] {
			t.Fatalf("tools = %#v, want %#v", tools, wantTools)
		}
	}
}

func TestRootCommandManifestUsesUniquePositiveDiscoveryOrders(t *testing.T) {
	seen := map[int]string{}
	for _, entry := range rootCommandManifest() {
		if entry.Command == nil || entry.DiscoveryOrder <= 0 {
			continue
		}
		fields := strings.Fields(entry.Command.Use)
		if len(fields) == 0 {
			t.Fatalf("manifest entry with order %d has empty Use", entry.DiscoveryOrder)
		}
		if previous, ok := seen[entry.DiscoveryOrder]; ok {
			t.Fatalf("duplicate DiscoveryOrder %d for %s and %s", entry.DiscoveryOrder, previous, fields[0])
		}
		seen[entry.DiscoveryOrder] = fields[0]
	}
}

func TestTopLevelCommandNamesPlaceAdviseAfterInspect(t *testing.T) {
	names := topLevelCommandNames()

	inspectIndex := -1
	for i, name := range names {
		if name == "inspect" {
			inspectIndex = i
			break
		}
	}
	if inspectIndex < 0 {
		t.Fatalf("inspect missing from top-level commands: %#v", names)
	}
	if inspectIndex+1 >= len(names) || names[inspectIndex+1] != "advise" {
		t.Fatalf("commands around inspect = %#v, want advise immediately after inspect", names)
	}
}

func TestBuildCapabilitiesDataKeepsStableCommandOrderFromRootManifest(t *testing.T) {
	oldCfg := cfg
	t.Cleanup(func() { cfg = oldCfg })

	cfg = &config.Config{DefaultTheme: "default"}

	data, err := buildCapabilitiesData()
	if err != nil {
		t.Fatalf("buildCapabilitiesData() error = %v", err)
	}

	commands, ok := data["commands"].([]string)
	if !ok {
		t.Fatalf("commands type = %T", data["commands"])
	}

	want := []string{
		"convert",
		"inspect",
		"advise",
		"preview",
		"config",
		"write",
		"humanize",
		"title",
		"upload_image",
		"download_and_upload",
		"generate_image",
		"generate_cover",
		"generate_infographic",
		"create_draft",
		"create_image_post",
		"test-draft",
		"providers",
		"themes",
		"prompts",
		"layout",
		"brand",
		"doctor",
		"skills",
		"capabilities",
		"version",
	}
	if len(commands) != len(want) {
		t.Fatalf("commands length = %d, want %d: %#v", len(commands), len(want), commands)
	}
	for i := range want {
		if commands[i] != want[i] {
			t.Fatalf("commands[%d] = %q, want %q; commands=%#v", i, commands[i], want[i], commands)
		}
	}

	manifestNames := map[string]bool{}
	for _, entry := range rootCommandManifest() {
		if entry.Command == nil {
			continue
		}
		fields := strings.Fields(entry.Command.Use)
		if len(fields) == 0 {
			continue
		}
		manifestNames[fields[0]] = true
	}
	for _, command := range commands {
		if !manifestNames[command] {
			t.Fatalf("capabilities command %q is not backed by root manifest: %#v", command, commands)
		}
	}
}

func TestAddWechatAccountFlagCanBeCalledRepeatedly(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("addWechatAccountFlag panicked on repeated calls: %v", r)
		}
	}()

	addWechatAccountFlag(cmd)
	addWechatAccountFlag(cmd)

	if cmd.Flags().Lookup("wechat-account") == nil {
		t.Fatal("expected wechat-account flag")
	}
}

func TestBuildCapabilitiesDataIncludesLayoutWithoutUnreleasedFormat(t *testing.T) {
	oldCfg := cfg
	t.Cleanup(func() { cfg = oldCfg })

	cfg = &config.Config{DefaultTheme: "default"}
	data, err := buildCapabilitiesData()
	if err != nil {
		t.Fatalf("buildCapabilitiesData() error = %v", err)
	}

	commands, ok := data["commands"].([]string)
	if !ok {
		t.Fatalf("commands type = %T", data["commands"])
	}
	if !contains(commands, "layout") {
		t.Fatalf("commands missing layout: %#v", commands)
	}
	if !contains(commands, "brand") {
		t.Fatalf("commands missing brand: %#v", commands)
	}
	if !contains(commands, "doctor") {
		t.Fatalf("commands missing doctor: %#v", commands)
	}
	if !contains(commands, "skills") {
		t.Fatalf("commands missing skills: %#v", commands)
	}
	if contains(commands, "format") {
		t.Fatalf("commands should not include format in Capability Truth phase: %#v", commands)
	}

	layout, ok := data["layout"].(map[string]any)
	if !ok {
		t.Fatalf("layout type = %T", data["layout"])
	}
	if layout["available"] != true {
		t.Fatalf("layout available = %#v", layout["available"])
	}
	cat, err := layoutcatalog.DefaultCatalog()
	if err != nil {
		t.Fatal(err)
	}
	wantModuleCount := len(cat.ListFiltered(layoutcatalog.ListFilter{}))
	if layout["module_count"] != wantModuleCount {
		t.Fatalf("layout module_count = %#v, want %d", layout["module_count"], wantModuleCount)
	}
	if layout["module_count"] != 53 {
		t.Fatalf("layout module_count = %#v, want 53", layout["module_count"])
	}
	if layout["recommended_syntax_count"] != 53 {
		t.Fatalf("recommended_syntax_count = %#v", layout)
	}
	if layout["recommended_scenario_count"] != 68 {
		t.Fatalf("recommended_scenario_count = %#v", layout)
	}
	if layout["compatibility_module_count"] != 3 {
		t.Fatalf("compatibility_module_count = %#v", layout)
	}
	if layout["base_enhancement_count"] != 4 || layout["render_syntax_count"] != 60 {
		t.Fatalf("render count contract = %#v", layout)
	}
	wantCategories := []string{"brand", "conversion", "evidence", "free-layout", "infographic", "interactive", "judgment", "opening", "sprint4"}
	if got := layout["categories"]; !reflect.DeepEqual(got, wantCategories) {
		t.Fatalf("layout categories = %#v, want %#v", got, wantCategories)
	}
	if layout["api_mode_only"] != true {
		t.Fatalf("layout api_mode_only = %#v", layout["api_mode_only"])
	}
	if layout["supports_validate"] != true {
		t.Fatalf("layout supports_validate = %#v", layout["supports_validate"])
	}
	if _, ok := data["format"]; ok {
		t.Fatalf("capabilities should not expose unreleased format workflow: %#v", data["format"])
	}
}

func TestLayoutCapabilitiesExposeSingleCatalogCounts(t *testing.T) {
	layout := buildLayoutCapabilityData()
	if layout["recommended_syntax_count"] != 53 || layout["render_syntax_count"] != 60 {
		t.Fatalf("layout capability counts drifted: %#v", layout)
	}
	for _, obsolete := range []string{
		"effective_recommended_syntax_count",
		"effective_compatibility_module_count",
		"local_override_module_count",
	} {
		if _, ok := layout[obsolete]; ok {
			t.Fatalf("layout capability still exposes obsolete %q: %#v", obsolete, layout)
		}
	}
}

func TestCapabilitiesJSONSuppressesConfigBannerOnStderr(t *testing.T) {
	oldCfg := cfg
	oldJSON := jsonOutput
	t.Cleanup(func() {
		cfg = oldCfg
		jsonOutput = oldJSON
	})

	home := t.TempDir()
	t.Setenv("HOME", home)
	configDir := filepath.Join(home, ".config", "md2wechat")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	configContent := strings.Join([]string{
		"wechat:",
		"  appid: appid",
		"  secret: secret",
		"api:",
		"  md2wechat_key: api-key",
	}, "\n")
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg = nil
	jsonOutput = true

	stderr := captureStderr(t, func() {
		stdout := captureStdout(t, func() {
			if err := capabilitiesCmd.RunE(capabilitiesCmd, nil); err != nil {
				t.Fatalf("RunE() error = %v", err)
			}
		})
		var response map[string]any
		if err := json.Unmarshal(stdout, &response); err != nil {
			t.Fatalf("unmarshal response: %v\n%s", err, stdout)
		}
	})
	if strings.TrimSpace(string(stderr)) != "" {
		t.Fatalf("expected no stderr in json mode, got %q", string(stderr))
	}
}

func TestPromptsRenderCommandUsesStableJSONEnvelope(t *testing.T) {
	oldJSON := jsonOutput
	oldPromptKind := promptKind
	oldPromptArchetype := promptArchetype
	oldPromptTag := promptTag
	oldPromptVars := append([]string(nil), promptVars...)
	t.Cleanup(func() {
		jsonOutput = oldJSON
		promptKind = oldPromptKind
		promptArchetype = oldPromptArchetype
		promptTag = oldPromptTag
		promptVars = oldPromptVars
		promptcatalog.ResetDefaultCatalogForTests()
	})

	jsonOutput = true
	promptcatalog.ResetDefaultCatalogForTests()
	promptKind = "image"
	promptVars = []string{"ARTICLE_TITLE=测试标题", "ARTICLE_SUMMARY=测试摘要", "VISUAL_STYLE=极简"}

	stdout := captureStdout(t, func() {
		if err := promptsRenderCmd.RunE(promptsRenderCmd, []string{"cover-default"}); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["success"] != true || response["code"] != "PROMPTS_SHOWN" {
		t.Fatalf("unexpected response: %#v", response)
	}
	data, _ := response["data"].(map[string]any)
	rendered, _ := data["rendered"].(string)
	if !strings.Contains(rendered, "测试标题") {
		t.Fatalf("rendered = %q", rendered)
	}
	prompt, ok := data["prompt"].(map[string]any)
	if !ok {
		t.Fatalf("prompt summary type = %T", data["prompt"])
	}
	for _, field := range []string{"template", "metadata", "examples", "source"} {
		if _, leaked := prompt[field]; leaked {
			t.Fatalf("render response leaked prompt definition field %q: %#v", field, prompt)
		}
	}
}

func TestPromptsListCommandSupportsArchetypeAndTagFilters(t *testing.T) {
	oldJSON := jsonOutput
	oldPromptKind := promptKind
	oldPromptArchetype := promptArchetype
	oldPromptTag := promptTag
	t.Cleanup(func() {
		jsonOutput = oldJSON
		promptKind = oldPromptKind
		promptArchetype = oldPromptArchetype
		promptTag = oldPromptTag
		promptcatalog.ResetDefaultCatalogForTests()
	})

	jsonOutput = true
	promptcatalog.ResetDefaultCatalogForTests()
	promptKind = "image"
	promptArchetype = "cover"
	promptTag = "hero"

	stdout := captureStdout(t, func() {
		if err := promptsListCmd.RunE(promptsListCmd, nil); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	data, _ := response["data"].(map[string]any)
	prompts, _ := data["prompts"].([]any)
	if len(prompts) == 0 {
		t.Fatalf("expected filtered prompts in response: %#v", response)
	}
	first, _ := prompts[0].(map[string]any)
	if first["archetype"] != "cover" {
		t.Fatalf("unexpected prompt archetype: %#v", first)
	}
}

func TestPromptsListIncludesFlatVectorPanoramaInfographic(t *testing.T) {
	oldJSON := jsonOutput
	oldPromptKind := promptKind
	oldPromptArchetype := promptArchetype
	oldPromptTag := promptTag
	t.Cleanup(func() {
		jsonOutput = oldJSON
		promptKind = oldPromptKind
		promptArchetype = oldPromptArchetype
		promptTag = oldPromptTag
		promptcatalog.ResetDefaultCatalogForTests()
	})

	jsonOutput = true
	promptcatalog.ResetDefaultCatalogForTests()
	promptKind = "image"
	promptArchetype = "infographic"
	promptTag = "flat-vector"

	stdout := captureStdout(t, func() {
		if err := promptsListCmd.RunE(promptsListCmd, nil); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	data, _ := response["data"].(map[string]any)
	prompts, _ := data["prompts"].([]any)
	if len(prompts) == 0 {
		t.Fatalf("expected filtered prompts in response: %#v", response)
	}

	found := false
	for _, item := range prompts {
		prompt, _ := item.(map[string]any)
		if prompt["name"] == "infographic-flat-vector-panorama" {
			found = true
			if prompt["archetype"] != "infographic" {
				t.Fatalf("unexpected prompt archetype: %#v", prompt)
			}
		}
	}
	if !found {
		t.Fatalf("expected infographic-flat-vector-panorama in response: %#v", prompts)
	}
}

func TestPromptsListIncludesDarkTicketInfographicByTag(t *testing.T) {
	oldJSON := jsonOutput
	oldPromptKind := promptKind
	oldPromptArchetype := promptArchetype
	oldPromptTag := promptTag
	t.Cleanup(func() {
		jsonOutput = oldJSON
		promptKind = oldPromptKind
		promptArchetype = oldPromptArchetype
		promptTag = oldPromptTag
		promptcatalog.ResetDefaultCatalogForTests()
	})

	jsonOutput = true
	promptcatalog.ResetDefaultCatalogForTests()
	promptKind = "image"
	promptArchetype = "infographic"
	promptTag = "ticket"

	stdout := captureStdout(t, func() {
		if err := promptsListCmd.RunE(promptsListCmd, nil); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	data, _ := response["data"].(map[string]any)
	prompts, _ := data["prompts"].([]any)
	if len(prompts) == 0 {
		t.Fatalf("expected filtered prompts in response: %#v", response)
	}

	found := false
	for _, item := range prompts {
		prompt, _ := item.(map[string]any)
		if prompt["name"] == "infographic-dark-ticket-cn" {
			found = true
			if prompt["archetype"] != "infographic" {
				t.Fatalf("unexpected prompt archetype: %#v", prompt)
			}
		}
	}
	if !found {
		t.Fatalf("expected infographic-dark-ticket-cn in response: %#v", prompts)
	}
}

func TestPromptsListIncludesHanddrawnSketchnoteByTag(t *testing.T) {
	oldJSON := jsonOutput
	oldPromptKind := promptKind
	oldPromptArchetype := promptArchetype
	oldPromptTag := promptTag
	t.Cleanup(func() {
		jsonOutput = oldJSON
		promptKind = oldPromptKind
		promptArchetype = oldPromptArchetype
		promptTag = oldPromptTag
		promptcatalog.ResetDefaultCatalogForTests()
	})

	jsonOutput = true
	promptcatalog.ResetDefaultCatalogForTests()
	promptKind = "image"
	promptArchetype = "infographic"
	promptTag = "sketchnote"

	stdout := captureStdout(t, func() {
		if err := promptsListCmd.RunE(promptsListCmd, nil); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	data, _ := response["data"].(map[string]any)
	prompts, _ := data["prompts"].([]any)
	if len(prompts) == 0 {
		t.Fatalf("expected filtered prompts in response: %#v", response)
	}

	found := false
	for _, item := range prompts {
		prompt, _ := item.(map[string]any)
		if prompt["name"] == "infographic-handdrawn-sketchnote" {
			found = true
			if prompt["archetype"] != "infographic" {
				t.Fatalf("unexpected prompt archetype: %#v", prompt)
			}
		}
	}
	if !found {
		t.Fatalf("expected infographic-handdrawn-sketchnote in response: %#v", prompts)
	}
}

func TestPromptsListIncludesAppleKeynotePremiumByTag(t *testing.T) {
	oldJSON := jsonOutput
	oldPromptKind := promptKind
	oldPromptArchetype := promptArchetype
	oldPromptTag := promptTag
	t.Cleanup(func() {
		jsonOutput = oldJSON
		promptKind = oldPromptKind
		promptArchetype = oldPromptArchetype
		promptTag = oldPromptTag
		promptcatalog.ResetDefaultCatalogForTests()
	})

	jsonOutput = true
	promptcatalog.ResetDefaultCatalogForTests()
	promptKind = "image"
	promptArchetype = "infographic"
	promptTag = "apple"

	stdout := captureStdout(t, func() {
		if err := promptsListCmd.RunE(promptsListCmd, nil); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	data, _ := response["data"].(map[string]any)
	prompts, _ := data["prompts"].([]any)
	if len(prompts) == 0 {
		t.Fatalf("expected filtered prompts in response: %#v", response)
	}

	found := false
	for _, item := range prompts {
		prompt, _ := item.(map[string]any)
		if prompt["name"] == "infographic-apple-keynote-premium" {
			found = true
			if prompt["archetype"] != "infographic" {
				t.Fatalf("unexpected prompt archetype: %#v", prompt)
			}
		}
	}
	if !found {
		t.Fatalf("expected infographic-apple-keynote-premium in response: %#v", prompts)
	}
}

func TestPromptsListIncludesVictorianBannerByTag(t *testing.T) {
	oldJSON := jsonOutput
	oldPromptKind := promptKind
	oldPromptArchetype := promptArchetype
	oldPromptTag := promptTag
	t.Cleanup(func() {
		jsonOutput = oldJSON
		promptKind = oldPromptKind
		promptArchetype = oldPromptArchetype
		promptTag = oldPromptTag
		promptcatalog.ResetDefaultCatalogForTests()
	})

	jsonOutput = true
	promptcatalog.ResetDefaultCatalogForTests()
	promptKind = "image"
	promptArchetype = "infographic"
	promptTag = "victorian"

	stdout := captureStdout(t, func() {
		if err := promptsListCmd.RunE(promptsListCmd, nil); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	data, _ := response["data"].(map[string]any)
	prompts, _ := data["prompts"].([]any)
	if len(prompts) == 0 {
		t.Fatalf("expected filtered prompts in response: %#v", response)
	}

	found := false
	for _, item := range prompts {
		prompt, _ := item.(map[string]any)
		if prompt["name"] == "infographic-victorian-engraving-banner" {
			found = true
			if prompt["archetype"] != "infographic" {
				t.Fatalf("unexpected prompt archetype: %#v", prompt)
			}
		}
	}
	if !found {
		t.Fatalf("expected infographic-victorian-engraving-banner in response: %#v", prompts)
	}
}
