package main

import (
	"encoding/json"
	"strings"
	"testing"

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
			if provider.DefaultModel != "gpt-image-1.5" {
				t.Fatalf("openai default model = %q, want gpt-image-1.5", provider.DefaultModel)
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
		"ark":        "doubao-seedream-5.0-lite",
		"openrouter": "google/gemini-3-pro-image-preview",
		"gemini":     "gemini-3.1-flash-image-preview",
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
