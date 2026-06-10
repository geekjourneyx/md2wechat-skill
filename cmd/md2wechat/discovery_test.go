package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"github.com/geekjourneyx/md2wechat-skill/internal/promptcatalog"
)

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
		"volcengine": "doubao-seedream-5-0-260128",
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

	want := map[string]string{
		"elegant-green": "elegant",
		"sspai-red":     "featured",
		"wechat-native": "featured",
	}
	for _, theme := range themes {
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
	if !contains(commands, "inspect") || !contains(commands, "preview") || !contains(commands, "convert") || !contains(commands, "skills") {
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
	if layout["module_count"] != 43 {
		t.Fatalf("layout module_count = %#v, want 43", layout["module_count"])
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
