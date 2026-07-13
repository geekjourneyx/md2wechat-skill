package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"github.com/geekjourneyx/md2wechat-skill/internal/image"
	"github.com/geekjourneyx/md2wechat-skill/internal/promptcatalog"
)

func TestResolveGenerateImagePromptRendersPlanVisualAssetPresets(t *testing.T) {
	promptcatalog.ResetDefaultCatalogForTests()
	t.Cleanup(promptcatalog.ResetDefaultCatalogForTests)
	t.Chdir(t.TempDir())

	tests := []struct {
		name       string
		useCase    string
		wantAspect string
	}{
		{name: "cover-claude-warm", useCase: "cover", wantAspect: "21:9"},
		{name: "cover-semantic-concept", useCase: "cover", wantAspect: "21:9"},
		{name: "cover-editorial-collage", useCase: "cover", wantAspect: "21:9"},
		{name: "cover-suspense-black-gold", useCase: "cover", wantAspect: "2.35:1"},
		{name: "infographic-claude-warm", useCase: "infographic", wantAspect: "21:9"},
		{name: "infographic-ticket-process", useCase: "infographic", wantAspect: "21:9"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := resolveGenerateImagePrompt(generateImageInput{
				Preset:            tt.name,
				Title:             "AI 时代的内容生产系统",
				Summary:           "从选题、写作到发布，建立可重复执行的内容工作流。",
				Keywords:          "AI,内容生产,工作流",
				Style:             "高级、克制、编辑感",
				RequiredArchetype: tt.useCase,
			})
			if err != nil {
				t.Fatalf("resolveGenerateImagePrompt() error = %v", err)
			}
			if !strings.Contains(prompt, "AI 时代的内容生产系统") {
				t.Fatalf("prompt missing article title: %q", prompt)
			}
			if !strings.Contains(prompt, tt.wantAspect) {
				t.Fatalf("prompt missing default aspect %q: %q", tt.wantAspect, prompt)
			}
			if strings.Contains(prompt, "{{") || strings.Contains(prompt, "}}") {
				t.Fatalf("prompt contains unresolved placeholder: %q", prompt)
			}
		})
	}
}

func TestResolveGenerateImagePromptWithPresetAndArticle(t *testing.T) {
	oldPreset, oldArticle := generateImageCmdPreset, generateImageCmdArticle
	oldTitle, oldSummary := generateImageCmdTitle, generateImageCmdSummary
	oldKeywords, oldStyle := generateImageCmdKeywords, generateImageCmdStyle
	oldAspect := generateImageCmdAspect
	t.Cleanup(func() {
		generateImageCmdPreset = oldPreset
		generateImageCmdArticle = oldArticle
		generateImageCmdTitle = oldTitle
		generateImageCmdSummary = oldSummary
		generateImageCmdKeywords = oldKeywords
		generateImageCmdStyle = oldStyle
		generateImageCmdAspect = oldAspect
	})

	article := strings.Join([]string{
		"---",
		"title: AI 时代的写作系统",
		"digest: 一篇关于写作工作流、提示词和信息组织的总结。",
		"---",
		"",
		"# 忽略这个标题",
		"",
		"正文第一段。",
	}, "\n")
	articlePath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(articlePath, []byte(article), 0600); err != nil {
		t.Fatalf("write article: %v", err)
	}

	generateImageCmdPreset = "cover-hero"
	generateImageCmdArticle = articlePath

	prompt, err := resolveGenerateImagePrompt(generateImageInput{
		Preset:  generateImageCmdPreset,
		Article: generateImageCmdArticle,
	})
	if err != nil {
		t.Fatalf("resolveGenerateImagePrompt() error = %v", err)
	}
	if !strings.Contains(prompt, "AI 时代的写作系统") {
		t.Fatalf("prompt missing title: %q", prompt)
	}
	if !strings.Contains(prompt, "一篇关于写作工作流") {
		t.Fatalf("prompt missing summary: %q", prompt)
	}
	if !strings.Contains(prompt, "16:9") {
		t.Fatalf("prompt missing default aspect ratio: %q", prompt)
	}
}

func TestRunGenerateImageUsesPresetPrompt(t *testing.T) {
	oldCfg := cfg
	oldPreset, oldArticle := generateImageCmdPreset, generateImageCmdArticle
	oldTitle, oldSummary := generateImageCmdTitle, generateImageCmdSummary
	oldKeywords, oldStyle := generateImageCmdKeywords, generateImageCmdStyle
	oldAspect, oldSize, oldModel := generateImageCmdAspect, generateImageCmdSize, generateImageCmdModel
	oldNewImageProcessor, oldNewImageProcessorWithConfig := newImageProcessor, newImageProcessorWithConfig
	t.Cleanup(func() {
		cfg = oldCfg
		generateImageCmdPreset = oldPreset
		generateImageCmdArticle = oldArticle
		generateImageCmdTitle = oldTitle
		generateImageCmdSummary = oldSummary
		generateImageCmdKeywords = oldKeywords
		generateImageCmdStyle = oldStyle
		generateImageCmdAspect = oldAspect
		generateImageCmdSize = oldSize
		generateImageCmdModel = oldModel
		newImageProcessor = oldNewImageProcessor
		newImageProcessorWithConfig = oldNewImageProcessorWithConfig
	})

	cfg = &config.Config{
		WechatAppID:  "appid",
		WechatSecret: "secret",
		ImageAPIKey:  "image-key",
	}
	generateImageCmdPreset = "infographic-comparison"
	generateImageCmdTitle = "提示词系统设计"
	generateImageCmdSummary = "比较不同图片提示词组织方式的优缺点"
	generateImageCmdStyle = "technical schematic"

	expectedPrompt, err := resolveGenerateImagePrompt(generateImageInput{
		Preset:  generateImageCmdPreset,
		Title:   generateImageCmdTitle,
		Summary: generateImageCmdSummary,
		Style:   generateImageCmdStyle,
	})
	if err != nil {
		t.Fatalf("resolveGenerateImagePrompt() error = %v", err)
	}

	processor := &fakeImageProcessor{
		generateResults: map[string]*image.GenerateAndUploadResult{
			expectedPrompt: {
				Prompt:      expectedPrompt,
				OriginalURL: "https://provider.example/image.png",
				MediaID:     "media-123",
				WechatURL:   "https://wechat.local/media-123",
			},
		},
	}
	newImageProcessor = func() imageProcessor { return processor }
	newImageProcessorWithConfig = func(runtimeCfg *config.Config) imageProcessor { return processor }

	stdout := captureStdout(t, func() {
		if err := runGenerateImage(nil); err != nil {
			t.Fatalf("runGenerateImage() error = %v", err)
		}
	})

	if len(processor.generateCalls) != 1 {
		t.Fatalf("generateCalls = %#v", processor.generateCalls)
	}
	if processor.generateCalls[0] != expectedPrompt {
		t.Fatalf("generate prompt = %q, want %q", processor.generateCalls[0], expectedPrompt)
	}

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["success"] != true {
		t.Fatalf("unexpected response: %#v", response)
	}
	data, _ := response["data"].(map[string]any)
	if data["media_id"] != "media-123" {
		t.Fatalf("unexpected response data: %#v", data)
	}
}

func TestRunGenerateImagePlanWithRawPromptHasNoSideEffects(t *testing.T) {
	oldCfg, oldJSON := cfg, jsonOutput
	oldNewImageProcessor, oldNewImageProcessorWithConfig := newImageProcessor, newImageProcessorWithConfig
	t.Cleanup(func() {
		cfg = oldCfg
		jsonOutput = oldJSON
		newImageProcessor = oldNewImageProcessor
		newImageProcessorWithConfig = oldNewImageProcessorWithConfig
	})

	cfg = &config.Config{}
	jsonOutput = true
	newImageProcessor = func() imageProcessor {
		t.Fatal("newImageProcessor should not be called in plan mode")
		return nil
	}
	newImageProcessorWithConfig = func(runtimeCfg *config.Config) imageProcessor {
		t.Fatal("newImageProcessorWithConfig should not be called in plan mode")
		return nil
	}

	stdout := captureStdout(t, func() {
		if err := runGenerateImageWithInput(generateImageInput{
			Command:   "generate_image",
			Plan:      true,
			RawPrompt: "A quiet editorial cover about agents planning images",
			Size:      "1792x1024",
			Model:     "image-model-hint",
		}); err != nil {
			t.Fatalf("runGenerateImageWithInput() error = %v", err)
		}
	})

	response := decodeResponse(t, stdout)
	if response["code"] != codeImagePlanReady {
		t.Fatalf("code = %v, want %s", response["code"], codeImagePlanReady)
	}
	if response["status"] != "action_required" {
		t.Fatalf("status = %v, want action_required", response["status"])
	}
	if response["message"] != "Image plan ready; generate the image with a host Agent or configured provider." {
		t.Fatalf("message = %v", response["message"])
	}
	data := responseData(t, response)
	if data["mode"] != "plan" {
		t.Fatalf("mode = %v", data["mode"])
	}
	if data["command"] != "generate_image" {
		t.Fatalf("command = %v", data["command"])
	}
	if data["execution_owner"] != "host_agent" {
		t.Fatalf("execution_owner = %v", data["execution_owner"])
	}
	if data["side_effects"] != false || data["requires_provider"] != false || data["requires_image_api_key"] != false {
		t.Fatalf("unexpected side effect flags: %#v", data)
	}
	if data["prompt"] != "A quiet editorial cover about agents planning images" {
		t.Fatalf("prompt = %v", data["prompt"])
	}
	if data["raw_prompt"] != "A quiet editorial cover about agents planning images" {
		t.Fatalf("raw_prompt = %v", data["raw_prompt"])
	}
	if data["size"] != "1792x1024" || data["model_hint"] != "image-model-hint" {
		t.Fatalf("size/model fields = %#v", data)
	}
	assertStablePlanFieldsPresent(t, data)
	assertNoGeneratedAssetFields(t, data)
}

func TestRunGenerateImagePlanRequiresJSONOutput(t *testing.T) {
	oldCfg, oldJSON := cfg, jsonOutput
	t.Cleanup(func() {
		cfg = oldCfg
		jsonOutput = oldJSON
	})

	cfg = &config.Config{}
	jsonOutput = false

	err := runGenerateImageWithInput(generateImageInput{
		Command:   "generate_image",
		Plan:      true,
		RawPrompt: "plan this image",
	})
	cliErr, ok := extractCLIError(err)
	if !ok {
		t.Fatalf("error type = %T, want *cliError", err)
	}
	if cliErr.Code != codeConfigInvalid {
		t.Fatalf("Code = %q, want %q", cliErr.Code, codeConfigInvalid)
	}
	if !strings.Contains(cliErr.Message, "--plan requires --json") {
		t.Fatalf("Message = %q", cliErr.Message)
	}
}

func TestRunGenerateImageUsesModelOverride(t *testing.T) {
	oldCfg := cfg
	oldNewImageProcessor := newImageProcessor
	oldNewImageProcessorWithConfig := newImageProcessorWithConfig
	t.Cleanup(func() {
		cfg = oldCfg
		newImageProcessor = oldNewImageProcessor
		newImageProcessorWithConfig = oldNewImageProcessorWithConfig
	})

	cfg = &config.Config{
		WechatAppID:  "appid",
		WechatSecret: "secret",
		ImageAPIKey:  "image-key",
		ImageModel:   "default-model",
	}

	processor := &fakeImageProcessor{
		generateResults: map[string]*image.GenerateAndUploadResult{
			"test prompt": {
				Prompt:      "test prompt",
				OriginalURL: "https://provider.example/image.png",
				MediaID:     "media-override",
				WechatURL:   "https://wechat.local/media-override",
			},
		},
	}

	newImageProcessor = func() imageProcessor {
		t.Fatal("newImageProcessor should not be used when --model is set")
		return nil
	}

	newImageProcessorWithConfig = func(runtimeCfg *config.Config) imageProcessor {
		if runtimeCfg == cfg {
			t.Fatal("expected model override to use a config copy")
		}
		if runtimeCfg.ImageModel != "override-model" {
			t.Fatalf("ImageModel = %q, want override-model", runtimeCfg.ImageModel)
		}
		if cfg.ImageModel != "default-model" {
			t.Fatalf("original cfg.ImageModel mutated = %q", cfg.ImageModel)
		}
		return processor
	}

	stdout := captureStdout(t, func() {
		if err := runGenerateImageWithInput(generateImageInput{
			RawPrompt: "test prompt",
			Model:     "override-model",
		}); err != nil {
			t.Fatalf("runGenerateImageWithInput() error = %v", err)
		}
	})

	if len(processor.generateCalls) != 1 || processor.generateCalls[0] != "test prompt" {
		t.Fatalf("generateCalls = %#v", processor.generateCalls)
	}

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["success"] != true {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestRunGenerateImageWithRawPromptIgnoresUnusedContextFlags(t *testing.T) {
	oldCfg := cfg
	oldNewImageProcessor := newImageProcessor
	oldNewImageProcessorWithConfig := newImageProcessorWithConfig
	t.Cleanup(func() {
		cfg = oldCfg
		newImageProcessor = oldNewImageProcessor
		newImageProcessorWithConfig = oldNewImageProcessorWithConfig
	})

	cfg = &config.Config{
		WechatAppID:  "appid",
		WechatSecret: "secret",
		ImageAPIKey:  "image-key",
	}

	processor := &fakeImageProcessor{
		generateResults: map[string]*image.GenerateAndUploadResult{
			"raw prompt wins": {
				Prompt:      "raw prompt wins",
				OriginalURL: "https://provider.example/raw.png",
				MediaID:     "raw-media",
				WechatURL:   "https://wechat.local/raw-media",
			},
		},
	}
	newImageProcessor = func() imageProcessor { return processor }
	newImageProcessorWithConfig = func(runtimeCfg *config.Config) imageProcessor {
		t.Fatal("newImageProcessorWithConfig should not be called without model override")
		return nil
	}

	stdout := captureStdout(t, func() {
		if err := runGenerateImageWithInput(generateImageInput{
			RawPrompt: "raw prompt wins",
			Title:     "unused title",
			Aspect:    "16:9",
			Style:     "unused style",
		}); err != nil {
			t.Fatalf("runGenerateImageWithInput() error = %v", err)
		}
	})

	if len(processor.generateCalls) != 1 || processor.generateCalls[0] != "raw prompt wins" {
		t.Fatalf("generateCalls = %#v", processor.generateCalls)
	}
	response := decodeResponse(t, stdout)
	if response["success"] != true {
		t.Fatalf("unexpected response: %#v", response)
	}
	data := responseData(t, response)
	if data["media_id"] != "raw-media" {
		t.Fatalf("unexpected response data: %#v", data)
	}
}

func TestRunGenerateCoverPlanUsesDefaultPresetAndArticleContext(t *testing.T) {
	oldCfg, oldJSON := cfg, jsonOutput
	t.Cleanup(func() {
		cfg = oldCfg
		jsonOutput = oldJSON
	})

	cfg = &config.Config{}
	jsonOutput = true

	article := strings.Join([]string{
		"---",
		"title: Agent Image Plan Mode",
		"digest: Plan images locally before any provider or upload side effects.",
		"---",
		"",
		"# Ignored body heading",
		"",
		"Body text should not replace frontmatter digest.",
	}, "\n")
	articlePath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(articlePath, []byte(article), 0600); err != nil {
		t.Fatalf("write article: %v", err)
	}

	stdout := captureStdout(t, func() {
		if err := runGeneratePresetImage("cover", "cover-default", generateImageInput{
			Command: "generate_cover",
			Plan:    true,
			Article: articlePath,
		}); err != nil {
			t.Fatalf("runGeneratePresetImage() error = %v", err)
		}
	})

	data := responseData(t, decodeResponse(t, stdout))
	if data["preset"] != "cover-default" {
		t.Fatalf("preset = %v", data["preset"])
	}
	if data["archetype"] != "cover" || data["primary_use_case"] != "cover" {
		t.Fatalf("archetype/use case = %#v", data)
	}
	if data["article"] != articlePath {
		t.Fatalf("article = %v", data["article"])
	}
	if data["default_aspect_ratio"] != "16:9" {
		t.Fatalf("default_aspect_ratio = %v", data["default_aspect_ratio"])
	}
	compatibleUseCases, ok := data["compatible_use_cases"].([]any)
	if !ok {
		t.Fatalf("compatible_use_cases = %#v, want JSON array", data["compatible_use_cases"])
	}
	if compatibleUseCases == nil {
		t.Fatalf("compatible_use_cases decoded as nil")
	}
	recommendedAspectRatios, ok := data["recommended_aspect_ratios"].([]any)
	if !ok {
		t.Fatalf("recommended_aspect_ratios = %#v, want JSON array", data["recommended_aspect_ratios"])
	}
	if recommendedAspectRatios == nil {
		t.Fatalf("recommended_aspect_ratios decoded as nil")
	}
	prompt, _ := data["prompt"].(string)
	if !strings.Contains(prompt, "Agent Image Plan Mode") || !strings.Contains(prompt, "16:9") {
		t.Fatalf("prompt missing title or aspect: %q", prompt)
	}
	assertNoGeneratedAssetFields(t, data)
}

func TestRunGenerateInfographicPlanUsesDefaultPresetAndSummaryContext(t *testing.T) {
	oldCfg, oldJSON := cfg, jsonOutput
	t.Cleanup(func() {
		cfg = oldCfg
		jsonOutput = oldJSON
	})

	cfg = &config.Config{}
	jsonOutput = true

	stdout := captureStdout(t, func() {
		if err := runGeneratePresetImage("infographic", "infographic-default", generateImageInput{
			Command: "generate_infographic",
			Plan:    true,
			Title:   "Plan Mode Architecture",
			Summary: "A no-side-effect handoff for host agents to generate images.",
		}); err != nil {
			t.Fatalf("runGeneratePresetImage() error = %v", err)
		}
	})

	data := responseData(t, decodeResponse(t, stdout))
	if data["preset"] != "infographic-default" {
		t.Fatalf("preset = %v", data["preset"])
	}
	if data["archetype"] != "infographic" {
		t.Fatalf("archetype = %v", data["archetype"])
	}
	if data["title"] != "Plan Mode Architecture" || data["summary"] != "A no-side-effect handoff for host agents to generate images." {
		t.Fatalf("context fields = %#v", data)
	}
	prompt, _ := data["prompt"].(string)
	if !strings.Contains(prompt, "Plan Mode Architecture") || !strings.Contains(prompt, "no-side-effect handoff") {
		t.Fatalf("prompt missing context: %q", prompt)
	}
	assertNoGeneratedAssetFields(t, data)
}

func TestRunGenerateCoverPlanRejectsWrongArchetype(t *testing.T) {
	oldCfg, oldJSON := cfg, jsonOutput
	t.Cleanup(func() {
		cfg = oldCfg
		jsonOutput = oldJSON
	})

	cfg = &config.Config{}
	jsonOutput = true

	err := runGeneratePresetImage("cover", "cover-default", generateImageInput{
		Command: "generate_cover",
		Plan:    true,
		Preset:  "infographic-default",
		Title:   "标题",
		Summary: "摘要",
	})
	cliErr, ok := extractCLIError(err)
	if !ok {
		t.Fatalf("error type = %T, want *cliError", err)
	}
	if cliErr.Code != codeConfigInvalid {
		t.Fatalf("Code = %q, want %q", cliErr.Code, codeConfigInvalid)
	}
	if !strings.Contains(cliErr.Message, "expected cover") {
		t.Fatalf("Message = %q", cliErr.Message)
	}
}

func TestResolveGenerateImagePromptRejectsMixedRawPromptAndPreset(t *testing.T) {
	oldPreset := generateImageCmdPreset
	t.Cleanup(func() { generateImageCmdPreset = oldPreset })

	generateImageCmdPreset = "cover-default"
	if _, err := resolveGenerateImagePrompt(generateImageInput{
		RawPrompt: "raw prompt",
		Preset:    generateImageCmdPreset,
	}); err == nil {
		t.Fatal("expected error for mixed raw prompt and preset")
	}
}

func TestRunGeneratePresetImageRejectsWrongArchetype(t *testing.T) {
	cfg = &config.Config{
		WechatAppID:  "appid",
		WechatSecret: "secret",
		ImageAPIKey:  "image-key",
	}

	err := runGeneratePresetImage("cover", "cover-default", generateImageInput{
		Preset:  "infographic-default",
		Title:   "标题",
		Summary: "摘要",
	})
	if err == nil || !strings.Contains(err.Error(), "expected cover") {
		t.Fatalf("runGeneratePresetImage() error = %v", err)
	}
}

func TestRunGeneratePresetImageAllowsCompatibleCoverUseCase(t *testing.T) {
	cfg = &config.Config{
		WechatAppID:  "appid",
		WechatSecret: "secret",
		ImageAPIKey:  "image-key",
	}

	_, err := resolveGenerateImagePrompt(generateImageInput{
		Preset:            "infographic-victorian-engraving-banner",
		Title:             "标题",
		Summary:           "摘要",
		RequiredArchetype: "cover",
	})
	if err != nil {
		t.Fatalf("resolveGenerateImagePrompt() error = %v", err)
	}
}

func TestResolveGenerateImagePromptUsesSpecDefaultAspectRatio(t *testing.T) {
	cfg = &config.Config{
		WechatAppID:  "appid",
		WechatSecret: "secret",
		ImageAPIKey:  "image-key",
	}

	prompt, err := resolveGenerateImagePrompt(generateImageInput{
		Preset:  "infographic-victorian-engraving-banner",
		Title:   "标题",
		Summary: "摘要",
	})
	if err != nil {
		t.Fatalf("resolveGenerateImagePrompt() error = %v", err)
	}
	if !strings.Contains(prompt, "21:9") {
		t.Fatalf("expected 21:9 default aspect ratio in prompt: %q", prompt)
	}
}

func TestRunGenerateCoverUsesDefaultPreset(t *testing.T) {
	oldCfg := cfg
	oldNewImageProcessor := newImageProcessor
	t.Cleanup(func() {
		cfg = oldCfg
		newImageProcessor = oldNewImageProcessor
	})

	cfg = &config.Config{
		WechatAppID:  "appid",
		WechatSecret: "secret",
		ImageAPIKey:  "image-key",
	}

	expectedPrompt, err := resolveGenerateImagePrompt(generateImageInput{
		Preset:            "cover-default",
		Title:             "标题",
		Summary:           "摘要",
		RequiredArchetype: "cover",
	})
	if err != nil {
		t.Fatalf("resolveGenerateImagePrompt() error = %v", err)
	}

	processor := &fakeImageProcessor{
		generateResults: map[string]*image.GenerateAndUploadResult{
			expectedPrompt: {
				Prompt:      expectedPrompt,
				OriginalURL: "https://provider.example/cover.png",
				MediaID:     "cover-1",
				WechatURL:   "https://wechat.local/cover-1",
			},
		},
	}
	newImageProcessor = func() imageProcessor { return processor }

	stdout := captureStdout(t, func() {
		if err := runGeneratePresetImage("cover", "cover-default", generateImageInput{
			Title:   "标题",
			Summary: "摘要",
		}); err != nil {
			t.Fatalf("runGeneratePresetImage() error = %v", err)
		}
	})

	if len(processor.generateCalls) != 1 || processor.generateCalls[0] != expectedPrompt {
		t.Fatalf("generateCalls = %#v", processor.generateCalls)
	}

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["success"] != true {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func decodeResponse(t *testing.T, stdout []byte) map[string]any {
	t.Helper()
	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	return response
}

func responseData(t *testing.T, response map[string]any) map[string]any {
	t.Helper()
	data, ok := response["data"].(map[string]any)
	if !ok {
		t.Fatalf("response data = %#v", response["data"])
	}
	return data
}

func assertNoGeneratedAssetFields(t *testing.T, data map[string]any) {
	t.Helper()
	for _, field := range []string{"media_id", "wechat_url", "original_url", "image_url"} {
		if _, ok := data[field]; ok {
			t.Fatalf("plan data unexpectedly includes %s: %#v", field, data)
		}
	}
}

func assertStablePlanFieldsPresent(t *testing.T, data map[string]any) {
	t.Helper()
	for _, field := range []string{
		"raw_prompt",
		"preset",
		"archetype",
		"primary_use_case",
		"compatible_use_cases",
		"recommended_aspect_ratios",
		"default_aspect_ratio",
		"article",
		"title",
		"summary",
		"keywords",
		"style",
		"aspect",
		"size",
		"model_hint",
		"suggested_filename",
		"alt_text",
	} {
		if _, ok := data[field]; !ok {
			t.Fatalf("plan data missing stable field %s: %#v", field, data)
		}
	}
}
