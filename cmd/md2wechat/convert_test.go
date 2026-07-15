package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"github.com/geekjourneyx/md2wechat-skill/internal/converter"
	"github.com/geekjourneyx/md2wechat-skill/internal/image"
	"github.com/geekjourneyx/md2wechat-skill/internal/publish"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type fakeConverter struct {
	result       *converter.ConvertResult
	images       []converter.ImageRef
	reqs         []*converter.ConvertRequest
	extractCalls int
}

func TestRunConvertUsesConfiguredDefaultThemeUnlessFlagged(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldTheme, oldMode, oldAPIKey := convertTheme, convertMode, convertAPIKey
	oldFontSize, oldBackground := convertFontSize, convertBackgroundType
	oldCustomPrompt, oldOutput := convertCustomPrompt, convertOutput
	oldPreview, oldUpload, oldDraft := convertPreview, convertUpload, convertDraft
	oldSaveDraft, oldCover, oldCoverMediaID := convertSaveDraft, convertCoverImage, convertCoverMediaID
	oldTitle, oldAuthor, oldDigest := convertTitle, convertAuthor, convertDigest
	oldNewConverter := newMarkdownConverter
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		convertTheme, convertMode, convertAPIKey = oldTheme, oldMode, oldAPIKey
		convertFontSize, convertBackgroundType = oldFontSize, oldBackground
		convertCustomPrompt, convertOutput = oldCustomPrompt, oldOutput
		convertPreview, convertUpload, convertDraft = oldPreview, oldUpload, oldDraft
		convertSaveDraft, convertCoverImage, convertCoverMediaID = oldSaveDraft, oldCover, oldCoverMediaID
		convertTitle, convertAuthor, convertDigest = oldTitle, oldAuthor, oldDigest
		newMarkdownConverter = oldNewConverter
	})

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(markdownPath, []byte("# Title\n\nBody"), 0o600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}
	cfg = &config.Config{MD2WechatAPIKey: "api-key", DefaultTheme: "chinese"}
	log = zap.NewNop()
	convertMode = "api"
	convertAPIKey = "api-key"
	convertFontSize = "medium"
	convertBackgroundType = "none"
	convertCustomPrompt = ""
	convertOutput = ""
	convertPreview = false
	convertUpload = false
	convertDraft = false
	convertSaveDraft = ""
	convertCoverImage = ""
	convertCoverMediaID = ""
	convertTitle = ""
	convertAuthor = ""
	convertDigest = ""

	for _, tt := range []struct {
		name      string
		explicit  bool
		wantTheme string
	}{
		{name: "unflagged uses configured theme", wantTheme: "chinese"},
		{name: "explicit default overrides config", explicit: true, wantTheme: "default"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			convertTheme = "default"
			cmd := &cobra.Command{Use: "convert"}
			cmd.Flags().StringVar(&convertTheme, "theme", "default", "Theme name")
			if tt.explicit {
				if err := cmd.Flags().Set("theme", "default"); err != nil {
					t.Fatal(err)
				}
			}
			conv := &fakeConverter{result: &converter.ConvertResult{
				Success: true,
				Mode:    converter.ModeAPI,
				Theme:   tt.wantTheme,
				HTML:    "<p>Body</p>",
			}}
			newMarkdownConverter = func() converter.Converter { return conv }

			if err := runConvert(cmd, []string{markdownPath}); err != nil {
				t.Fatalf("runConvert() error = %v", err)
			}
			if len(conv.reqs) != 1 || conv.reqs[0].Theme != tt.wantTheme {
				t.Fatalf("converter requests = %#v, want theme %q", conv.reqs, tt.wantTheme)
			}
		})
	}
}

func TestConvertHelpUsesRuntimeThemeDiscovery(t *testing.T) {
	help := convertCmd.Long
	if !strings.Contains(help, "md2wechat themes list --json") {
		t.Fatalf("convert help missing runtime theme discovery command: %q", help)
	}
	if !strings.Contains(help, "md2wechat themes show <name> --json") {
		t.Fatalf("convert help missing theme detail command: %q", help)
	}
	for _, stale := range []string{"Supported professional themes (48 total)", "minimal-gold", "autumn-warm, spring-fresh"} {
		if strings.Contains(help, stale) {
			t.Fatalf("convert help retained hardcoded theme catalog %q: %q", stale, help)
		}
	}
}

func TestValidateConvertConfigRejectsInvalidEffectIntent(t *testing.T) {
	oldCfg := cfg
	oldMode, oldTheme, oldAPIKey := convertMode, convertTheme, convertAPIKey
	oldCustomPrompt := convertCustomPrompt
	oldPreview, oldUpload, oldDraft := convertPreview, convertUpload, convertDraft
	oldCover, oldCoverMediaID := convertCoverImage, convertCoverMediaID
	t.Cleanup(func() {
		cfg = oldCfg
		convertMode, convertTheme, convertAPIKey = oldMode, oldTheme, oldAPIKey
		convertCustomPrompt = oldCustomPrompt
		convertPreview, convertUpload, convertDraft = oldPreview, oldUpload, oldDraft
		convertCoverImage, convertCoverMediaID = oldCover, oldCoverMediaID
	})

	tests := []struct {
		name    string
		preview bool
		upload  bool
		draft   bool
		cover   string
		coverID string
		wantErr string
	}{
		{name: "preview plus upload", preview: true, upload: true, wantErr: "--preview cannot be combined with --upload or --draft"},
		{name: "preview plus draft", preview: true, draft: true, wantErr: "--preview cannot be combined with --upload or --draft"},
		{name: "cover without draft", cover: "cover.jpg", wantErr: "--cover and --cover-media-id require --draft"},
		{name: "cover id without draft", coverID: "media-id", wantErr: "--cover and --cover-media-id require --draft"},
		{name: "conflicting draft cover", draft: true, cover: "cover.jpg", coverID: "media-id", wantErr: "cannot be used together"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg = &config.Config{}
			convertMode = "ai"
			convertTheme = "default"
			convertAPIKey = ""
			convertCustomPrompt = "test prompt"
			convertPreview = tt.preview
			convertUpload = tt.upload
			convertDraft = tt.draft
			convertCoverImage = tt.cover
			convertCoverMediaID = tt.coverID

			err := validateConvertConfig()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("validateConvertConfig() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func (f *fakeConverter) Convert(req *converter.ConvertRequest) *converter.ConvertResult {
	f.reqs = append(f.reqs, req)
	return f.result
}

func (f *fakeConverter) ExtractImages(markdown string) []converter.ImageRef {
	f.extractCalls++
	return f.images
}

type fakeImageProcessor struct {
	localCalls    []string
	onlineCalls   []string
	generateCalls []string

	localResults    map[string]*image.UploadResult
	onlineResults   map[string]*image.UploadResult
	generateResults map[string]*image.GenerateAndUploadResult

	localErrs    map[string]error
	onlineErrs   map[string]error
	generateErrs map[string]error
}

func (f *fakeImageProcessor) totalCalls() int {
	return len(f.localCalls) + len(f.onlineCalls) + len(f.generateCalls)
}

func (f *fakeImageProcessor) UploadLocalImage(filePath string) (*image.UploadResult, error) {
	f.localCalls = append(f.localCalls, filePath)
	if err := f.localErrs[filePath]; err != nil {
		return nil, err
	}
	if result, ok := f.localResults[filePath]; ok {
		return result, nil
	}
	return nil, fmt.Errorf("unexpected local path: %s", filePath)
}

func (f *fakeImageProcessor) DownloadAndUpload(url string) (*image.UploadResult, error) {
	f.onlineCalls = append(f.onlineCalls, url)
	if err := f.onlineErrs[url]; err != nil {
		return nil, err
	}
	if result, ok := f.onlineResults[url]; ok {
		return result, nil
	}
	return nil, fmt.Errorf("unexpected online url: %s", url)
}

func (f *fakeImageProcessor) GenerateAndUpload(prompt string) (*image.GenerateAndUploadResult, error) {
	f.generateCalls = append(f.generateCalls, prompt)
	if err := f.generateErrs[prompt]; err != nil {
		return nil, err
	}
	if result, ok := f.generateResults[prompt]; ok {
		return result, nil
	}
	return nil, fmt.Errorf("unexpected prompt: %s", prompt)
}

func (f *fakeImageProcessor) GenerateAndUploadWithSize(prompt string, size string) (*image.GenerateAndUploadResult, error) {
	return f.GenerateAndUpload(prompt)
}

type fakeDraftCreator struct {
	artifacts []publish.Artifact
	result    *publish.DraftResult
	err       error
}

func (f *fakeDraftCreator) CreateDraft(artifact publish.Artifact) (*publish.DraftResult, error) {
	f.artifacts = append(f.artifacts, artifact)
	if f.err != nil {
		return nil, f.err
	}
	if f.result != nil {
		return f.result, nil
	}
	return &publish.DraftResult{MediaID: "draft-media-id"}, nil
}

func TestRunConvertDraftPipelineReplacesMixedImagesAndUsesMarkdownTitle(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldMode, oldTheme, oldAPIKey := convertMode, convertTheme, convertAPIKey
	oldFontSize, oldBackground := convertFontSize, convertBackgroundType
	oldCustomPrompt, oldOutput := convertCustomPrompt, convertOutput
	oldPreview, oldUpload, oldDraft := convertPreview, convertUpload, convertDraft
	oldSaveDraft, oldCover := convertSaveDraft, convertCoverImage
	oldTitle, oldAuthor, oldDigest := convertTitle, convertAuthor, convertDigest
	oldNewConverter, oldNewProcessor := newMarkdownConverter, newImageProcessor
	oldNewDraftCreator, oldUploadCoverImageFn := newDraftCreator, uploadCoverImageFn
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		convertMode, convertTheme, convertAPIKey = oldMode, oldTheme, oldAPIKey
		convertFontSize, convertBackgroundType = oldFontSize, oldBackground
		convertCustomPrompt, convertOutput = oldCustomPrompt, oldOutput
		convertPreview, convertUpload, convertDraft = oldPreview, oldUpload, oldDraft
		convertSaveDraft, convertCoverImage = oldSaveDraft, oldCover
		convertTitle, convertAuthor, convertDigest = oldTitle, oldAuthor, oldDigest
		newMarkdownConverter, newImageProcessor = oldNewConverter, oldNewProcessor
		newDraftCreator, uploadCoverImageFn = oldNewDraftCreator, oldUploadCoverImageFn
	})

	cfg = &config.Config{
		WechatAppID:        "appid",
		WechatSecret:       "secret",
		MD2WechatAPIKey:    "api-key",
		DefaultConvertMode: "api",
		MaxImageWidth:      1920,
		MaxImageSize:       5 * 1024 * 1024,
		HTTPTimeout:        30,
	}
	log = zap.NewNop()

	convertMode = "api"
	convertTheme = "default"
	convertPreview = false
	convertUpload = false
	convertDraft = true
	convertSaveDraft = ""
	convertAPIKey = ""
	convertFontSize = "medium"
	convertBackgroundType = "default"
	convertCustomPrompt = ""
	convertOutput = ""
	convertTitle = ""
	convertAuthor = ""
	convertDigest = ""

	dir := t.TempDir()
	convertCoverImage = filepath.Join(dir, "cover.jpg")
	if err := os.WriteFile(convertCoverImage, []byte("cover"), 0600); err != nil {
		t.Fatalf("write cover: %v", err)
	}
	markdownPath := filepath.Join(dir, "article.md")
	localRelative := filepath.Join("images", "local.png")
	localPath := filepath.Join(dir, localRelative)
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		t.Fatalf("create local image directory: %v", err)
	}
	if err := os.WriteFile(localPath, []byte("png"), 0o600); err != nil {
		t.Fatalf("write local image: %v", err)
	}
	markdown := strings.Join([]string{
		"---",
		"title: Frontmatter 标题",
		"author: 张三",
		"digest: 来自 frontmatter 的摘要",
		"---",
		"",
		"# 发布标题",
		"",
		"![local](images/local.png)",
		"![online](https://example.com/remote.png)",
		"![ai](__generate:draw a fox__)",
	}, "\n")
	if err := os.WriteFile(markdownPath, []byte(markdown), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	conv := &fakeConverter{
		result: &converter.ConvertResult{
			Success: true,
			Mode:    converter.ModeAPI,
			Theme:   "default",
			HTML:    `<p>a</p><img src="https://cdn.example.com/1"><p>b</p><img src="https://cdn.example.com/2"><p>c</p><img src="https://cdn.example.com/3">`,
			Images: []converter.ImageRef{
				{Index: 0, Type: converter.ImageTypeLocal, Original: localRelative, Placeholder: "<!-- IMG:0 -->"},
				{Index: 1, Type: converter.ImageTypeOnline, Original: "https://example.com/remote.png", Placeholder: "<!-- IMG:1 -->"},
				{Index: 2, Type: converter.ImageTypeAI, Original: "draw a fox", AIPrompt: "draw a fox", Placeholder: "<!-- IMG:2 -->"},
			},
		},
	}
	processor := &fakeImageProcessor{
		localResults: map[string]*image.UploadResult{
			localPath: {MediaID: "m-local", WechatURL: "https://wechat.local/local"},
		},
		onlineResults: map[string]*image.UploadResult{
			"https://example.com/remote.png": {MediaID: "m-online", WechatURL: "https://wechat.local/remote"},
		},
		generateResults: map[string]*image.GenerateAndUploadResult{
			"draw a fox": {MediaID: "m-ai", WechatURL: "https://wechat.local/ai"},
		},
	}
	drafter := &fakeDraftCreator{result: &publish.DraftResult{MediaID: "draft-1"}}

	newMarkdownConverter = func() converter.Converter { return conv }
	newImageProcessor = func() imageProcessor { return processor }
	newDraftCreator = func() publish.DraftCreator { return drafter }
	uploadCoverImageFn = func(imagePath string) (string, error) {
		if imagePath != convertCoverImage {
			t.Fatalf("cover image path = %q, want %q", imagePath, convertCoverImage)
		}
		return "cover-media-id", nil
	}

	if err := runConvert(nil, []string{markdownPath}); err != nil {
		t.Fatalf("runConvert() error = %v", err)
	}

	if len(processor.localCalls) != 1 || processor.localCalls[0] != localPath {
		t.Fatalf("local upload calls = %#v", processor.localCalls)
	}
	if len(processor.onlineCalls) != 1 || processor.onlineCalls[0] != "https://example.com/remote.png" {
		t.Fatalf("online upload calls = %#v", processor.onlineCalls)
	}
	if len(processor.generateCalls) != 1 || processor.generateCalls[0] != "draw a fox" {
		t.Fatalf("generate calls = %#v", processor.generateCalls)
	}
	if len(drafter.artifacts) != 1 {
		t.Fatalf("draft artifacts = %#v", drafter.artifacts)
	}

	artifact := drafter.artifacts[0]
	if artifact.Metadata.Title != "Frontmatter 标题" {
		t.Fatalf("article title = %q, want %q", artifact.Metadata.Title, "Frontmatter 标题")
	}
	if artifact.Metadata.Author != "张三" {
		t.Fatalf("article author = %q, want %q", artifact.Metadata.Author, "张三")
	}
	if artifact.Metadata.Digest != "来自 frontmatter 的摘要" {
		t.Fatalf("article digest = %q, want %q", artifact.Metadata.Digest, "来自 frontmatter 的摘要")
	}
	if artifact.CoverMediaID != "cover-media-id" {
		t.Fatalf("thumb media id = %q", artifact.CoverMediaID)
	}
	for _, expected := range []string{
		"https://wechat.local/local",
		"https://wechat.local/remote",
		"https://wechat.local/ai",
	} {
		if !strings.Contains(artifact.HTML, expected) {
			t.Fatalf("article content missing %q: %s", expected, artifact.HTML)
		}
	}
	if strings.Contains(artifact.HTML, "cdn.example.com") {
		t.Fatalf("article content still contains rewritten original URLs: %s", artifact.HTML)
	}
}

func TestSaveDraftWritesMetadataFromFrontMatter(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldMode, oldTheme, oldAPIKey := convertMode, convertTheme, convertAPIKey
	oldFontSize, oldBackground := convertFontSize, convertBackgroundType
	oldCustomPrompt, oldOutput := convertCustomPrompt, convertOutput
	oldPreview, oldUpload, oldDraft := convertPreview, convertUpload, convertDraft
	oldSaveDraftPath, oldCover := convertSaveDraft, convertCoverImage
	oldTitle, oldAuthor, oldDigest := convertTitle, convertAuthor, convertDigest
	oldNewConverter := newMarkdownConverter
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		convertMode, convertTheme, convertAPIKey = oldMode, oldTheme, oldAPIKey
		convertFontSize, convertBackgroundType = oldFontSize, oldBackground
		convertCustomPrompt, convertOutput = oldCustomPrompt, oldOutput
		convertPreview, convertUpload, convertDraft = oldPreview, oldUpload, oldDraft
		convertSaveDraft, convertCoverImage = oldSaveDraftPath, oldCover
		convertTitle, convertAuthor, convertDigest = oldTitle, oldAuthor, oldDigest
		newMarkdownConverter = oldNewConverter
	})

	cfg = &config.Config{MD2WechatAPIKey: "api-key"}
	log = zap.NewNop()
	convertMode = "api"
	convertTheme = "default"
	convertAPIKey = ""
	convertFontSize = "medium"
	convertBackgroundType = "default"
	convertCustomPrompt = ""
	convertOutput = ""
	convertPreview = false
	convertUpload = false
	convertDraft = false
	convertTitle = ""
	convertAuthor = ""
	convertDigest = ""

	outputPath := filepath.Join(t.TempDir(), "draft.json")
	convertSaveDraft = outputPath
	markdownPath := filepath.Join(t.TempDir(), "article.md")
	markdown := strings.Join([]string{
		"---",
		"title: 文章标题",
		"author: 作者名",
		"digest: 文章摘要",
		"---",
		"",
		"正文",
	}, "\n")
	if err := os.WriteFile(markdownPath, []byte(markdown), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	newMarkdownConverter = func() converter.Converter {
		return &fakeConverter{
			result: &converter.ConvertResult{
				Success: true,
				Mode:    converter.ModeAPI,
				Theme:   "default",
				HTML:    "<p>content</p>",
			},
		}
	}

	if err := runConvert(nil, []string{markdownPath}); err != nil {
		t.Fatalf("runConvert() error = %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read saved draft: %v", err)
	}
	content := string(data)
	for _, expected := range []string{`"title": "文章标题"`, `"author": "作者名"`, `"digest": "文章摘要"`} {
		if !strings.Contains(content, expected) {
			t.Fatalf("saved draft missing %q: %s", expected, content)
		}
	}
}

func TestRunConvertCommandLineMetadataOverridesMarkdown(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldMode, oldTheme, oldAPIKey := convertMode, convertTheme, convertAPIKey
	oldFontSize, oldBackground := convertFontSize, convertBackgroundType
	oldCustomPrompt, oldOutput := convertCustomPrompt, convertOutput
	oldPreview, oldUpload, oldDraft := convertPreview, convertUpload, convertDraft
	oldSaveDraftPath, oldCover := convertSaveDraft, convertCoverImage
	oldTitle, oldAuthor, oldDigest := convertTitle, convertAuthor, convertDigest
	oldNewConverter := newMarkdownConverter
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		convertMode, convertTheme, convertAPIKey = oldMode, oldTheme, oldAPIKey
		convertFontSize, convertBackgroundType = oldFontSize, oldBackground
		convertCustomPrompt, convertOutput = oldCustomPrompt, oldOutput
		convertPreview, convertUpload, convertDraft = oldPreview, oldUpload, oldDraft
		convertSaveDraft, convertCoverImage = oldSaveDraftPath, oldCover
		convertTitle, convertAuthor, convertDigest = oldTitle, oldAuthor, oldDigest
		newMarkdownConverter = oldNewConverter
	})

	cfg = &config.Config{MD2WechatAPIKey: "api-key"}
	log = zap.NewNop()
	convertMode = "api"
	convertTheme = "default"
	convertAPIKey = ""
	convertFontSize = "medium"
	convertBackgroundType = "default"
	convertCustomPrompt = ""
	convertOutput = ""
	convertPreview = false
	convertUpload = false
	convertDraft = false
	convertSaveDraft = ""
	convertCoverImage = ""
	convertTitle = "命令行标题"
	convertAuthor = "命令行作者"
	convertDigest = "命令行摘要"

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	markdown := strings.Join([]string{
		"---",
		"title: Frontmatter 标题",
		"author: Frontmatter 作者",
		"digest: Frontmatter 摘要",
		"---",
		"",
		"# 正文标题",
		"",
		"正文",
	}, "\n")
	if err := os.WriteFile(markdownPath, []byte(markdown), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	newMarkdownConverter = func() converter.Converter {
		return &fakeConverter{
			result: &converter.ConvertResult{
				Success: true,
				Mode:    converter.ModeAPI,
				Theme:   "default",
				HTML:    "<p>content</p>",
			},
		}
	}

	outputPath := filepath.Join(t.TempDir(), "draft.json")
	convertSaveDraft = outputPath

	if err := runConvert(nil, []string{markdownPath}); err != nil {
		t.Fatalf("runConvert() error = %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read saved draft: %v", err)
	}
	content := string(data)
	for _, expected := range []string{`"title": "命令行标题"`, `"author": "命令行作者"`, `"digest": "命令行摘要"`} {
		if !strings.Contains(content, expected) {
			t.Fatalf("saved draft missing %q: %s", expected, content)
		}
	}
}

func TestRunConvertPassesResolvedMetadataIntoAIRequest(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldMode, oldTheme, oldAPIKey := convertMode, convertTheme, convertAPIKey
	oldFontSize, oldBackground := convertFontSize, convertBackgroundType
	oldCustomPrompt, oldOutput := convertCustomPrompt, convertOutput
	oldPreview, oldUpload, oldDraft := convertPreview, convertUpload, convertDraft
	oldSaveDraftPath, oldCover := convertSaveDraft, convertCoverImage
	oldTitle, oldAuthor, oldDigest := convertTitle, convertAuthor, convertDigest
	oldNewConverter := newMarkdownConverter
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		convertMode, convertTheme, convertAPIKey = oldMode, oldTheme, oldAPIKey
		convertFontSize, convertBackgroundType = oldFontSize, oldBackground
		convertCustomPrompt, convertOutput = oldCustomPrompt, oldOutput
		convertPreview, convertUpload, convertDraft = oldPreview, oldUpload, oldDraft
		convertSaveDraft, convertCoverImage = oldSaveDraftPath, oldCover
		convertTitle, convertAuthor, convertDigest = oldTitle, oldAuthor, oldDigest
		newMarkdownConverter = oldNewConverter
	})

	cfg = &config.Config{MD2WechatAPIKey: "api-key"}
	log = zap.NewNop()
	convertMode = "ai"
	convertTheme = "autumn-warm"
	convertAPIKey = ""
	convertFontSize = "medium"
	convertBackgroundType = "default"
	convertCustomPrompt = ""
	convertOutput = ""
	convertPreview = false
	convertUpload = false
	convertDraft = false
	convertSaveDraft = ""
	convertCoverImage = ""
	convertTitle = "命令行标题"
	convertAuthor = "命令行作者"
	convertDigest = "命令行摘要"

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	markdown := strings.Join([]string{
		"---",
		"title: Frontmatter 标题",
		"author: Frontmatter 作者",
		"digest: Frontmatter 摘要",
		"---",
		"",
		"# 正文标题",
		"",
		"正文",
	}, "\n")
	if err := os.WriteFile(markdownPath, []byte(markdown), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	conv := &fakeConverter{
		result: &converter.ConvertResult{
			Success: true,
			Mode:    converter.ModeAI,
			Theme:   "autumn-warm",
			Status:  "action_required",
			Action:  "convert",
			Prompt:  "prompt",
		},
	}
	newMarkdownConverter = func() converter.Converter { return conv }

	if err := runConvert(nil, []string{markdownPath}); err != nil {
		t.Fatalf("runConvert() error = %v", err)
	}

	if len(conv.reqs) != 1 {
		t.Fatalf("converter requests = %#v", conv.reqs)
	}
	if conv.reqs[0].Metadata.Title != "命令行标题" {
		t.Fatalf("request title = %q", conv.reqs[0].Metadata.Title)
	}
	if conv.reqs[0].Metadata.Author != "命令行作者" {
		t.Fatalf("request author = %q", conv.reqs[0].Metadata.Author)
	}
	if conv.reqs[0].Metadata.Digest != "命令行摘要" {
		t.Fatalf("request digest = %q", conv.reqs[0].Metadata.Digest)
	}
}

func TestRunConvertAICustomPromptAllowsDefaultTheme(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldMode, oldTheme, oldAPIKey := convertMode, convertTheme, convertAPIKey
	oldFontSize, oldBackground := convertFontSize, convertBackgroundType
	oldCustomPrompt, oldOutput := convertCustomPrompt, convertOutput
	oldPreview, oldUpload, oldDraft := convertPreview, convertUpload, convertDraft
	oldSaveDraft, oldCover, oldCoverMediaID := convertSaveDraft, convertCoverImage, convertCoverMediaID
	oldNewConverter := newMarkdownConverter
	oldJSON := jsonOutput
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		convertMode, convertTheme, convertAPIKey = oldMode, oldTheme, oldAPIKey
		convertFontSize, convertBackgroundType = oldFontSize, oldBackground
		convertCustomPrompt, convertOutput = oldCustomPrompt, oldOutput
		convertPreview, convertUpload, convertDraft = oldPreview, oldUpload, oldDraft
		convertSaveDraft, convertCoverImage, convertCoverMediaID = oldSaveDraft, oldCover, oldCoverMediaID
		newMarkdownConverter = oldNewConverter
		jsonOutput = oldJSON
	})

	cfg = &config.Config{}
	log = zap.NewNop()
	jsonOutput = true
	convertMode = "ai"
	convertTheme = "default"
	convertAPIKey = ""
	convertFontSize = "medium"
	convertBackgroundType = "default"
	convertCustomPrompt = "Use a compact editorial layout."
	convertOutput = ""
	convertPreview = false
	convertUpload = false
	convertDraft = false
	convertSaveDraft = ""
	convertCoverImage = ""
	convertCoverMediaID = ""
	newMarkdownConverter = func() converter.Converter {
		return converter.NewConverter(cfg, log)
	}

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(markdownPath, []byte("# Title\n\nBody"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	stdout := captureStdout(t, func() {
		if err := runConvert(nil, []string{markdownPath}); err != nil {
			t.Fatalf("runConvert() error = %v", err)
		}
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["success"] != true || response["code"] != codeConvertAIRequestReady {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response["code"] == "THEME_MODE_MISMATCH" {
		t.Fatalf("custom prompt should not fail theme compatibility: %#v", response)
	}
}

func TestRunConvertStripsFrontMatterBeforeCallingConverter(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldMode, oldTheme, oldAPIKey := convertMode, convertTheme, convertAPIKey
	oldFontSize, oldBackground := convertFontSize, convertBackgroundType
	oldCustomPrompt, oldOutput := convertCustomPrompt, convertOutput
	oldPreview, oldUpload, oldDraft := convertPreview, convertUpload, convertDraft
	oldSaveDraftPath, oldCover := convertSaveDraft, convertCoverImage
	oldTitle, oldAuthor, oldDigest := convertTitle, convertAuthor, convertDigest
	oldNewConverter := newMarkdownConverter
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		convertMode, convertTheme, convertAPIKey = oldMode, oldTheme, oldAPIKey
		convertFontSize, convertBackgroundType = oldFontSize, oldBackground
		convertCustomPrompt, convertOutput = oldCustomPrompt, oldOutput
		convertPreview, convertUpload, convertDraft = oldPreview, oldUpload, oldDraft
		convertSaveDraft, convertCoverImage = oldSaveDraftPath, oldCover
		convertTitle, convertAuthor, convertDigest = oldTitle, oldAuthor, oldDigest
		newMarkdownConverter = oldNewConverter
	})

	cfg = &config.Config{MD2WechatAPIKey: "api-key"}
	log = zap.NewNop()
	convertMode = "api"
	convertTheme = "default"
	convertAPIKey = ""
	convertFontSize = "medium"
	convertBackgroundType = "default"
	convertCustomPrompt = ""
	convertOutput = ""
	convertPreview = false
	convertUpload = false
	convertDraft = false
	convertSaveDraft = ""
	convertCoverImage = ""
	convertTitle = ""
	convertAuthor = ""
	convertDigest = ""

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	markdown := strings.Join([]string{
		"---",
		"title: Frontmatter 标题",
		"author: Frontmatter 作者",
		"---",
		"",
		"# 正文标题",
		"",
		"正文",
	}, "\n")
	if err := os.WriteFile(markdownPath, []byte(markdown), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	conv := &fakeConverter{
		result: &converter.ConvertResult{
			Success: true,
			Mode:    converter.ModeAPI,
			Theme:   "default",
			HTML:    "<p>content</p>",
		},
	}
	newMarkdownConverter = func() converter.Converter { return conv }

	if err := runConvert(nil, []string{markdownPath}); err != nil {
		t.Fatalf("runConvert() error = %v", err)
	}

	if len(conv.reqs) != 1 {
		t.Fatalf("converter requests = %#v", conv.reqs)
	}
	if strings.Contains(conv.reqs[0].Markdown, "title: Frontmatter 标题") {
		t.Fatalf("converter markdown should not contain frontmatter: %q", conv.reqs[0].Markdown)
	}
	if !strings.Contains(conv.reqs[0].Markdown, "# 正文标题") {
		t.Fatalf("converter markdown missing body heading: %q", conv.reqs[0].Markdown)
	}
}

func TestRunConvertRejectsMetadataThatExceedsLimits(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldMode, oldTheme, oldAPIKey := convertMode, convertTheme, convertAPIKey
	oldFontSize, oldBackground := convertFontSize, convertBackgroundType
	oldCustomPrompt, oldOutput := convertCustomPrompt, convertOutput
	oldPreview, oldUpload, oldDraft := convertPreview, convertUpload, convertDraft
	oldSaveDraftPath, oldCover := convertSaveDraft, convertCoverImage
	oldTitle, oldAuthor, oldDigest := convertTitle, convertAuthor, convertDigest
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		convertMode, convertTheme, convertAPIKey = oldMode, oldTheme, oldAPIKey
		convertFontSize, convertBackgroundType = oldFontSize, oldBackground
		convertCustomPrompt, convertOutput = oldCustomPrompt, oldOutput
		convertPreview, convertUpload, convertDraft = oldPreview, oldUpload, oldDraft
		convertSaveDraft, convertCoverImage = oldSaveDraftPath, oldCover
		convertTitle, convertAuthor, convertDigest = oldTitle, oldAuthor, oldDigest
	})

	cfg = &config.Config{MD2WechatAPIKey: "api-key"}
	log = zap.NewNop()
	convertMode = "api"
	convertTheme = "default"
	convertAPIKey = ""
	convertFontSize = "medium"
	convertBackgroundType = "default"
	convertCustomPrompt = ""
	convertOutput = ""
	convertPreview = false
	convertUpload = false
	convertDraft = false
	convertSaveDraft = ""
	convertCoverImage = ""
	convertTitle = strings.Repeat("标", 33)
	convertAuthor = ""
	convertDigest = ""

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(markdownPath, []byte("# 正文标题\n"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	err := runConvert(nil, []string{markdownPath})
	if err == nil {
		t.Fatal("expected runConvert to fail")
	}
	if !strings.Contains(err.Error(), "--title exceeds 32 characters") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunConvertImageFailureBlocksDraftCreation(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldMode, oldTheme, oldAPIKey := convertMode, convertTheme, convertAPIKey
	oldFontSize, oldBackground := convertFontSize, convertBackgroundType
	oldCustomPrompt, oldOutput := convertCustomPrompt, convertOutput
	oldPreview, oldUpload, oldDraft := convertPreview, convertUpload, convertDraft
	oldSaveDraft, oldCover := convertSaveDraft, convertCoverImage
	oldNewConverter, oldNewProcessor := newMarkdownConverter, newImageProcessor
	oldNewDraftCreator, oldUploadCoverImageFn := newDraftCreator, uploadCoverImageFn
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		convertMode, convertTheme, convertAPIKey = oldMode, oldTheme, oldAPIKey
		convertFontSize, convertBackgroundType = oldFontSize, oldBackground
		convertCustomPrompt, convertOutput = oldCustomPrompt, oldOutput
		convertPreview, convertUpload, convertDraft = oldPreview, oldUpload, oldDraft
		convertSaveDraft, convertCoverImage = oldSaveDraft, oldCover
		newMarkdownConverter, newImageProcessor = oldNewConverter, oldNewProcessor
		newDraftCreator, uploadCoverImageFn = oldNewDraftCreator, oldUploadCoverImageFn
	})

	cfg = &config.Config{
		WechatAppID:        "appid",
		WechatSecret:       "secret",
		MD2WechatAPIKey:    "api-key",
		DefaultConvertMode: "api",
		MaxImageWidth:      1920,
		MaxImageSize:       5 * 1024 * 1024,
		HTTPTimeout:        30,
	}
	log = zap.NewNop()

	convertMode = "api"
	convertTheme = "default"
	convertPreview = false
	convertUpload = false
	convertDraft = true
	convertSaveDraft = ""
	convertAPIKey = ""
	convertFontSize = "medium"
	convertBackgroundType = "default"
	convertCustomPrompt = ""
	convertOutput = ""

	dir := t.TempDir()
	convertCoverImage = filepath.Join(dir, "cover.jpg")
	if err := os.WriteFile(convertCoverImage, []byte("cover"), 0600); err != nil {
		t.Fatalf("write cover: %v", err)
	}
	markdownPath := filepath.Join(dir, "article.md")
	if err := os.WriteFile(markdownPath, []byte("# Title\n\n![local](images/local.png)\n"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}
	localPath := filepath.Join(dir, "images", "local.png")
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		t.Fatalf("create local image directory: %v", err)
	}
	if err := os.WriteFile(localPath, []byte("png"), 0o600); err != nil {
		t.Fatalf("write local image: %v", err)
	}

	newMarkdownConverter = func() converter.Converter {
		return &fakeConverter{
			result: &converter.ConvertResult{
				Success: true,
				Mode:    converter.ModeAPI,
				Theme:   "default",
				HTML:    `<img src="images/local.png">`,
				Images: []converter.ImageRef{
					{Index: 0, Type: converter.ImageTypeLocal, Original: "images/local.png", Placeholder: "<!-- IMG:0 -->"},
				},
			},
		}
	}

	processor := &fakeImageProcessor{
		localResults: map[string]*image.UploadResult{},
		localErrs: map[string]error{
			localPath: fmt.Errorf("upload failed"),
		},
	}
	newImageProcessor = func() imageProcessor { return processor }

	drafter := &fakeDraftCreator{}
	newDraftCreator = func() publish.DraftCreator { return drafter }
	uploadCoverImageFn = func(imagePath string) (string, error) {
		t.Fatalf("uploadCoverImageFn should not be called when image processing fails")
		return "", nil
	}

	err := runConvert(nil, []string{markdownPath})
	if err == nil {
		t.Fatalf("expected runConvert to fail")
	}
	cliErr, ok := err.(*cliError)
	if !ok {
		t.Fatalf("error type = %T, want *cliError", err)
	}
	if cliErr.Code != codeConvertImageFailed || !strings.Contains(cliErr.Error(), "upload failed") {
		t.Fatalf("unexpected error: %#v", cliErr)
	}
	if len(drafter.artifacts) != 0 {
		t.Fatalf("draft creator should not be called on image failure: %#v", drafter.artifacts)
	}
}

func TestRunConvertDraftUsesCoverMediaIDWithoutUploadingCover(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldMode, oldTheme, oldAPIKey := convertMode, convertTheme, convertAPIKey
	oldFontSize, oldBackground := convertFontSize, convertBackgroundType
	oldCustomPrompt, oldOutput := convertCustomPrompt, convertOutput
	oldPreview, oldUpload, oldDraft := convertPreview, convertUpload, convertDraft
	oldSaveDraft, oldCover, oldCoverMediaID := convertSaveDraft, convertCoverImage, convertCoverMediaID
	oldNewConverter, oldNewProcessor := newMarkdownConverter, newImageProcessor
	oldNewDraftCreator, oldUploadCoverImageFn := newDraftCreator, uploadCoverImageFn
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		convertMode, convertTheme, convertAPIKey = oldMode, oldTheme, oldAPIKey
		convertFontSize, convertBackgroundType = oldFontSize, oldBackground
		convertCustomPrompt, convertOutput = oldCustomPrompt, oldOutput
		convertPreview, convertUpload, convertDraft = oldPreview, oldUpload, oldDraft
		convertSaveDraft, convertCoverImage, convertCoverMediaID = oldSaveDraft, oldCover, oldCoverMediaID
		newMarkdownConverter, newImageProcessor = oldNewConverter, oldNewProcessor
		newDraftCreator, uploadCoverImageFn = oldNewDraftCreator, oldUploadCoverImageFn
	})

	cfg = &config.Config{
		WechatAppID:        "appid",
		WechatSecret:       "secret",
		MD2WechatAPIKey:    "api-key",
		DefaultConvertMode: "api",
		MaxImageWidth:      1920,
		MaxImageSize:       5 * 1024 * 1024,
		HTTPTimeout:        30,
	}
	log = zap.NewNop()

	convertMode = "api"
	convertTheme = "default"
	convertPreview = false
	convertUpload = false
	convertDraft = true
	convertSaveDraft = ""
	convertCoverImage = ""
	convertCoverMediaID = "existing-cover-id"
	convertAPIKey = ""
	convertFontSize = "medium"
	convertBackgroundType = "default"
	convertCustomPrompt = ""
	convertOutput = ""

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(markdownPath, []byte("# Title\n"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	newMarkdownConverter = func() converter.Converter {
		return &fakeConverter{
			result: &converter.ConvertResult{
				Success: true,
				Mode:    converter.ModeAPI,
				Theme:   "default",
				HTML:    "<p>body</p>",
			},
		}
	}
	newImageProcessor = func() imageProcessor { return &fakeImageProcessor{} }

	drafter := &fakeDraftCreator{result: &publish.DraftResult{MediaID: "draft-existing"}}
	newDraftCreator = func() publish.DraftCreator { return drafter }
	uploadCoverImageFn = func(imagePath string) (string, error) {
		t.Fatalf("uploadCoverImageFn should not be called when --cover-media-id is used")
		return "", nil
	}

	if err := runConvert(nil, []string{markdownPath}); err != nil {
		t.Fatalf("runConvert() error = %v", err)
	}
	if len(drafter.artifacts) != 1 {
		t.Fatalf("draft artifacts = %#v", drafter.artifacts)
	}
	if drafter.artifacts[0].CoverMediaID != "existing-cover-id" {
		t.Fatalf("cover media id = %q", drafter.artifacts[0].CoverMediaID)
	}
}

func TestHandleAIResultUsesStableJSONEnvelopeWhenRequested(t *testing.T) {
	oldLog := log
	oldJSON := jsonOutput
	oldOutput := convertOutput
	t.Cleanup(func() {
		log = oldLog
		jsonOutput = oldJSON
		convertOutput = oldOutput
	})
	log = zap.NewNop()
	jsonOutput = true
	convertOutput = ""

	result := &converter.ConvertResult{
		Error: "AI_MODE_REQUEST:prompt body",
		Images: []converter.ImageRef{
			{Index: 0, Original: "./a.png"},
		},
	}

	stdout := captureStdout(t, func() {
		if err := handleAIResult(result, "article.md"); err != nil {
			t.Fatalf("handleAIResult() error = %v", err)
		}
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["success"] != true || response["code"] != codeConvertAIRequestReady {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response["schema_version"] != "v1" || response["status"] != "action_required" || response["retryable"] != false {
		t.Fatalf("unexpected envelope: %#v", response)
	}
	data, _ := response["data"].(map[string]any)
	if data["markdown_file"] != "article.md" || data["mode"] != "ai" || data["action"] != "ai_request" {
		t.Fatalf("unexpected data payload: %#v", data)
	}
	if data["prompt"] != "prompt body" {
		t.Fatalf("prompt = %#v", data["prompt"])
	}
}

func TestHandleAIResultWritesPromptToPromptFileInsteadOfHTML(t *testing.T) {
	oldLog := log
	oldJSON := jsonOutput
	oldOutput := convertOutput
	t.Cleanup(func() {
		log = oldLog
		jsonOutput = oldJSON
		convertOutput = oldOutput
	})
	log = zap.NewNop()
	jsonOutput = true

	dir := t.TempDir()
	convertOutput = filepath.Join(dir, "result.html")
	result := &converter.ConvertResult{
		Error:  "AI_MODE_REQUEST:prompt body",
		Images: []converter.ImageRef{{Index: 0, Original: "./a.png"}},
	}

	stdout := captureStdout(t, func() {
		if err := handleAIResult(result, "article.md"); err != nil {
			t.Fatalf("handleAIResult() error = %v", err)
		}
	})

	if _, err := os.Stat(convertOutput); !os.IsNotExist(err) {
		t.Fatalf("expected no html output file, got err=%v", err)
	}

	promptPath := filepath.Join(dir, "result.prompt.txt")
	data, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("read prompt file: %v", err)
	}
	if string(data) != "prompt body" {
		t.Fatalf("prompt file = %q", data)
	}

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	payload, _ := response["data"].(map[string]any)
	if payload["prompt_file"] != promptPath {
		t.Fatalf("prompt_file = %#v, want %q", payload["prompt_file"], promptPath)
	}
	if payload["requested_output_file"] != convertOutput {
		t.Fatalf("requested_output_file = %#v, want %q", payload["requested_output_file"], convertOutput)
	}
}

func TestHandleAIResultWriteFailureReturnsBeforeActionRequired(t *testing.T) {
	oldLog, oldJSON, oldOutput := log, jsonOutput, convertOutput
	t.Cleanup(func() {
		log, jsonOutput, convertOutput = oldLog, oldJSON, oldOutput
	})
	log = zap.NewNop()
	jsonOutput = true
	convertOutput = filepath.Join(t.TempDir(), "missing", "result.html")
	result := &converter.ConvertResult{Error: "AI_MODE_REQUEST:prompt body"}

	stdout := captureStdout(t, func() {
		err := handleAIResult(result, "article.md")
		if err == nil {
			t.Fatal("handleAIResult() error = nil")
		}
		cliErr, ok := err.(*cliError)
		if !ok || cliErr.Code != codeConvertFailed {
			t.Fatalf("handleAIResult() error = %#v, want %s", err, codeConvertFailed)
		}
	})
	if strings.Contains(string(stdout), codeConvertAIRequestReady) || strings.Contains(string(stdout), "action_required") {
		t.Fatalf("stdout announced readiness before prompt write: %s", stdout)
	}
	if len(stdout) != 0 {
		t.Fatalf("stdout = %q, want empty on prompt write failure", stdout)
	}
}

func TestRunConvertOutputWriteFailureDoesNotReportCompletion(t *testing.T) {
	oldCfg, oldLog, oldJSON := cfg, log, jsonOutput
	oldMode, oldTheme, oldAPIKey := convertMode, convertTheme, convertAPIKey
	oldFontSize, oldBackground := convertFontSize, convertBackgroundType
	oldCustomPrompt, oldOutput := convertCustomPrompt, convertOutput
	oldPreview, oldUpload, oldDraft := convertPreview, convertUpload, convertDraft
	oldSaveDraft, oldCover, oldCoverMediaID := convertSaveDraft, convertCoverImage, convertCoverMediaID
	oldTitle, oldAuthor, oldDigest := convertTitle, convertAuthor, convertDigest
	oldNewConverter := newMarkdownConverter
	oldReplaceOutputFile := replaceOutputFileFn
	t.Cleanup(func() {
		cfg, log, jsonOutput = oldCfg, oldLog, oldJSON
		convertMode, convertTheme, convertAPIKey = oldMode, oldTheme, oldAPIKey
		convertFontSize, convertBackgroundType = oldFontSize, oldBackground
		convertCustomPrompt, convertOutput = oldCustomPrompt, oldOutput
		convertPreview, convertUpload, convertDraft = oldPreview, oldUpload, oldDraft
		convertSaveDraft, convertCoverImage, convertCoverMediaID = oldSaveDraft, oldCover, oldCoverMediaID
		convertTitle, convertAuthor, convertDigest = oldTitle, oldAuthor, oldDigest
		newMarkdownConverter = oldNewConverter
		replaceOutputFileFn = oldReplaceOutputFile
	})

	cfg = &config.Config{MD2WechatAPIKey: "api-key"}
	log = zap.NewNop()
	jsonOutput = true
	convertMode = "api"
	convertTheme = "default"
	convertAPIKey = "api-key"
	convertFontSize = "medium"
	convertBackgroundType = "none"
	convertCustomPrompt = ""
	convertPreview = false
	convertUpload = false
	convertDraft = false
	convertSaveDraft = ""
	convertCoverImage = ""
	convertCoverMediaID = ""
	convertTitle = ""
	convertAuthor = ""
	convertDigest = ""
	newMarkdownConverter = func() converter.Converter {
		return &fakeConverter{result: &converter.ConvertResult{
			Success: true,
			Mode:    converter.ModeAPI,
			Theme:   "default",
			HTML:    "<p>body</p>",
		}}
	}
	markdownPath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(markdownPath, []byte("# Title\n\nBody"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	assertFailedWithoutCompletion := func(t *testing.T) {
		t.Helper()
		stdout := captureStdout(t, func() {
			err := runConvert(nil, []string{markdownPath})
			if err == nil {
				t.Fatal("runConvert() error = nil")
			}
			cliErr, ok := err.(*cliError)
			if !ok || cliErr.Code != codeConvertFailed {
				t.Fatalf("runConvert() error = %#v, want %s", err, codeConvertFailed)
			}
		})
		if strings.Contains(string(stdout), codeConvertCompleted) || strings.Contains(string(stdout), `"output_file"`) {
			t.Fatalf("stdout reported a usable completed artifact: %s", stdout)
		}
		if len(stdout) != 0 {
			t.Fatalf("stdout = %q, want empty on output write failure", stdout)
		}
	}

	t.Run("cannot create temp output", func(t *testing.T) {
		convertOutput = filepath.Join(t.TempDir(), "missing", "out.html")
		assertFailedWithoutCompletion(t)
	})

	t.Run("replace failure preserves existing output", func(t *testing.T) {
		outputDir := t.TempDir()
		convertOutput = filepath.Join(outputDir, "out.html")
		sentinel := []byte("PREEXISTING_SENTINEL")
		if err := os.WriteFile(convertOutput, sentinel, 0600); err != nil {
			t.Fatalf("write existing output: %v", err)
		}
		replaceOutputFileFn = func(_, _ string) error {
			return errors.New("injected replace failure")
		}

		assertFailedWithoutCompletion(t)
		data, err := os.ReadFile(convertOutput)
		if err != nil || !bytes.Equal(data, sentinel) {
			t.Fatalf("existing output changed: data=%q err=%v", data, err)
		}
		entries, err := os.ReadDir(outputDir)
		if err != nil {
			t.Fatalf("read output directory: %v", err)
		}
		if len(entries) != 1 || entries[0].Name() != "out.html" {
			t.Fatalf("temporary output leaked after replace failure: %#v", entries)
		}
	})
}

func TestRunConvertOutputsStableJSONEnvelopeWhenRequested(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldMode, oldTheme, oldAPIKey := convertMode, convertTheme, convertAPIKey
	oldFontSize, oldBackground := convertFontSize, convertBackgroundType
	oldCustomPrompt, oldOutput := convertCustomPrompt, convertOutput
	oldPreview, oldUpload, oldDraft := convertPreview, convertUpload, convertDraft
	oldSaveDraft, oldCover := convertSaveDraft, convertCoverImage
	oldNewConverter := newMarkdownConverter
	oldJSON := jsonOutput
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		convertMode, convertTheme, convertAPIKey = oldMode, oldTheme, oldAPIKey
		convertFontSize, convertBackgroundType = oldFontSize, oldBackground
		convertCustomPrompt, convertOutput = oldCustomPrompt, oldOutput
		convertPreview, convertUpload, convertDraft = oldPreview, oldUpload, oldDraft
		convertSaveDraft, convertCoverImage = oldSaveDraft, oldCover
		newMarkdownConverter = oldNewConverter
		jsonOutput = oldJSON
	})

	cfg = &config.Config{MD2WechatAPIKey: "api-key"}
	log = zap.NewNop()
	jsonOutput = true
	convertMode = "api"
	convertTheme = "default"
	convertAPIKey = ""
	convertFontSize = "medium"
	convertBackgroundType = "default"
	convertCustomPrompt = ""
	convertOutput = ""
	convertPreview = false
	convertUpload = false
	convertDraft = false
	convertSaveDraft = ""
	convertCoverImage = ""

	dir := t.TempDir()
	markdownPath := filepath.Join(dir, "article.md")
	markdown := "# 标题\n\n正文"
	if err := os.WriteFile(markdownPath, []byte(markdown), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	newMarkdownConverter = func() converter.Converter {
		return &fakeConverter{
			result: &converter.ConvertResult{
				Success: true,
				Mode:    converter.ModeAPI,
				Theme:   "default",
				HTML:    "<p>正文</p>",
				Images:  nil,
			},
		}
	}

	stdout := captureStdout(t, func() {
		if err := runConvert(nil, []string{markdownPath}); err != nil {
			t.Fatalf("runConvert() error = %v", err)
		}
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["success"] != true || response["code"] != codeConvertCompleted {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response["schema_version"] != "v1" || response["status"] != "completed" || response["retryable"] != false {
		t.Fatalf("unexpected envelope: %#v", response)
	}
	data, _ := response["data"].(map[string]any)
	if data["html"] != "<p>正文</p>" || data["mode"] != "api" || data["title"] != "标题" {
		t.Fatalf("unexpected data payload: %#v", data)
	}
}

func TestRunConvertJSONRejectsThemeModeMismatch(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldMode, oldTheme, oldAPIKey := convertMode, convertTheme, convertAPIKey
	oldFontSize, oldBackground := convertFontSize, convertBackgroundType
	oldCustomPrompt, oldOutput := convertCustomPrompt, convertOutput
	oldPreview, oldUpload, oldDraft := convertPreview, convertUpload, convertDraft
	oldSaveDraft, oldCover, oldCoverMediaID := convertSaveDraft, convertCoverImage, convertCoverMediaID
	oldJSON := jsonOutput
	oldExit := exitFunc
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		convertMode, convertTheme, convertAPIKey = oldMode, oldTheme, oldAPIKey
		convertFontSize, convertBackgroundType = oldFontSize, oldBackground
		convertCustomPrompt, convertOutput = oldCustomPrompt, oldOutput
		convertPreview, convertUpload, convertDraft = oldPreview, oldUpload, oldDraft
		convertSaveDraft, convertCoverImage, convertCoverMediaID = oldSaveDraft, oldCover, oldCoverMediaID
		jsonOutput = oldJSON
		exitFunc = oldExit
	})

	cfg = &config.Config{MD2WechatAPIKey: "api-key"}
	log = zap.NewNop()
	jsonOutput = true
	exitFunc = func(code int) {}
	convertMode = "api"
	convertTheme = "autumn-warm"
	convertAPIKey = "key"
	convertFontSize = "medium"
	convertBackgroundType = "default"
	convertCustomPrompt = ""
	convertOutput = ""
	convertPreview = false
	convertUpload = false
	convertDraft = false
	convertSaveDraft = ""
	convertCoverImage = ""
	convertCoverMediaID = ""

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(markdownPath, []byte("# Title\n\nBody"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	err := runConvert(nil, []string{markdownPath})
	if err == nil {
		t.Fatal("expected runConvert to fail")
	}

	stdout := captureStdout(t, func() {
		responseError(err)
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["success"] != false || response["code"] != "THEME_MODE_MISMATCH" {
		t.Fatalf("unexpected response: %#v", response)
	}
	errorDetails, _ := response["error_details"].(map[string]any)
	if errorDetails["theme"] != "autumn-warm" || errorDetails["mode"] != "api" {
		t.Fatalf("unexpected error details: %#v", response["error_details"])
	}
	if _, ok := response["next_actions"].([]any); !ok {
		t.Fatalf("missing next actions: %#v", response)
	}
}

func TestRunConvertJSONStillWritesOutputFileWhenRequested(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldMode, oldTheme, oldAPIKey := convertMode, convertTheme, convertAPIKey
	oldFontSize, oldBackground := convertFontSize, convertBackgroundType
	oldCustomPrompt, oldOutput := convertCustomPrompt, convertOutput
	oldPreview, oldUpload, oldDraft := convertPreview, convertUpload, convertDraft
	oldSaveDraft, oldCover := convertSaveDraft, convertCoverImage
	oldNewConverter := newMarkdownConverter
	oldJSON := jsonOutput
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		convertMode, convertTheme, convertAPIKey = oldMode, oldTheme, oldAPIKey
		convertFontSize, convertBackgroundType = oldFontSize, oldBackground
		convertCustomPrompt, convertOutput = oldCustomPrompt, oldOutput
		convertPreview, convertUpload, convertDraft = oldPreview, oldUpload, oldDraft
		convertSaveDraft, convertCoverImage = oldSaveDraft, oldCover
		newMarkdownConverter = oldNewConverter
		jsonOutput = oldJSON
	})

	cfg = &config.Config{MD2WechatAPIKey: "api-key"}
	log = zap.NewNop()
	jsonOutput = true
	convertMode = "api"
	convertTheme = "default"
	convertAPIKey = ""
	convertFontSize = "medium"
	convertBackgroundType = "default"
	convertCustomPrompt = ""
	convertPreview = false
	convertUpload = false
	convertDraft = false
	convertSaveDraft = ""
	convertCoverImage = ""

	dir := t.TempDir()
	markdownPath := filepath.Join(dir, "article.md")
	outputPath := filepath.Join(dir, "article.html")
	convertOutput = outputPath
	if err := os.WriteFile(markdownPath, []byte("# 标题\n\n正文"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	newMarkdownConverter = func() converter.Converter {
		return &fakeConverter{
			result: &converter.ConvertResult{
				Success: true,
				Mode:    converter.ModeAPI,
				Theme:   "default",
				HTML:    "<p>正文</p>",
			},
		}
	}

	stdout := captureStdout(t, func() {
		if err := runConvert(nil, []string{markdownPath}); err != nil {
			t.Fatalf("runConvert() error = %v", err)
		}
	})

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if string(data) != "<p>正文</p>" {
		t.Fatalf("output file content = %q", string(data))
	}

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	payload, _ := response["data"].(map[string]any)
	if payload["output_file"] != outputPath {
		t.Fatalf("output_file = %#v, want %q", payload["output_file"], outputPath)
	}
}

func preserveConvertPreflightGlobals(t *testing.T) {
	t.Helper()
	oldCfg, oldLog := cfg, log
	oldAccount := wechatAccountName
	oldValidateAPIKey := validateAPIKeyForWeChatAccount
	oldMode, oldTheme, oldAPIKey := convertMode, convertTheme, convertAPIKey
	oldFontSize, oldBackground := convertFontSize, convertBackgroundType
	oldCustomPrompt, oldOutput := convertCustomPrompt, convertOutput
	oldPreview, oldUpload, oldDraft := convertPreview, convertUpload, convertDraft
	oldSaveDraft, oldCover, oldCoverMediaID := convertSaveDraft, convertCoverImage, convertCoverMediaID
	oldTitle, oldAuthor, oldDigest := convertTitle, convertAuthor, convertDigest
	oldJSON := jsonOutput
	oldNewConverter, oldNewProcessor := newMarkdownConverter, newImageProcessor
	oldNewDraftCreator, oldUploadCover := newDraftCreator, uploadCoverImageFn
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		wechatAccountName = oldAccount
		validateAPIKeyForWeChatAccount = oldValidateAPIKey
		convertMode, convertTheme, convertAPIKey = oldMode, oldTheme, oldAPIKey
		convertFontSize, convertBackgroundType = oldFontSize, oldBackground
		convertCustomPrompt, convertOutput = oldCustomPrompt, oldOutput
		convertPreview, convertUpload, convertDraft = oldPreview, oldUpload, oldDraft
		convertSaveDraft, convertCoverImage, convertCoverMediaID = oldSaveDraft, oldCover, oldCoverMediaID
		convertTitle, convertAuthor, convertDigest = oldTitle, oldAuthor, oldDigest
		jsonOutput = oldJSON
		newMarkdownConverter, newImageProcessor = oldNewConverter, oldNewProcessor
		newDraftCreator, uploadCoverImageFn = oldNewDraftCreator, oldUploadCover
	})
}

func setConvertPreflightDefaults() {
	log = zap.NewNop()
	convertMode = "api"
	convertTheme = "default"
	convertAPIKey = ""
	convertFontSize = "medium"
	convertBackgroundType = "default"
	convertCustomPrompt = ""
	convertOutput = ""
	convertPreview = false
	convertUpload = false
	convertDraft = false
	convertSaveDraft = ""
	convertCoverImage = ""
	convertCoverMediaID = ""
	convertTitle = ""
	convertAuthor = ""
	convertDigest = ""
	jsonOutput = false
}

func TestRunConvertRejectsInvalidCoverBeforeNamedAccountAPIKeyValidation(t *testing.T) {
	preserveConvertPreflightGlobals(t)
	setConvertPreflightDefaults()

	cfg = &config.Config{
		MD2WechatAPIKey: "api-key",
		WechatAccounts: map[string]config.WechatAccount{
			"main": {AppID: "appid", Secret: "secret"},
		},
	}
	wechatAccountName = "main"
	convertDraft = true
	convertCoverImage = filepath.Join(t.TempDir(), "missing-cover.jpg")

	conv := &fakeConverter{result: &converter.ConvertResult{Success: true, Mode: converter.ModeAPI, Theme: "default", HTML: "<p>body</p>"}}
	processor := &fakeImageProcessor{}
	drafter := &fakeDraftCreator{}
	newMarkdownConverter = func() converter.Converter { return conv }
	newImageProcessor = func() imageProcessor { return processor }
	newDraftCreator = func() publish.DraftCreator { return drafter }
	uploadCoverImageFn = func(string) (string, error) { return "cover-id", nil }

	validatorCalls := 0
	validateAPIKeyForWeChatAccount = func(string) error {
		validatorCalls++
		return nil
	}
	markdownPath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(markdownPath, []byte("# Title\n"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	err := runConvert(nil, []string{markdownPath})
	cliErr, ok := err.(*cliError)
	if !ok || cliErr.Code != codeConvertDraftFailed {
		t.Fatalf("runConvert() error = %#v, want %s", err, codeConvertDraftFailed)
	}
	if validatorCalls != 0 || len(conv.reqs) != 0 || processor.totalCalls() != 0 || len(drafter.artifacts) != 0 {
		t.Fatalf("invalid cover caused effects: validator=%d convert=%d assets=%d drafts=%d", validatorCalls, len(conv.reqs), processor.totalCalls(), len(drafter.artifacts))
	}
}

func TestRunConvertRejectsInvalidLocalAssetBeforeProxyAPIKeyValidation(t *testing.T) {
	preserveConvertPreflightGlobals(t)
	setConvertPreflightDefaults()

	cfg = &config.Config{
		WechatAppID:     "appid",
		WechatSecret:    "secret",
		WechatProxyURL:  "https://proxy.example.com",
		MD2WechatAPIKey: "api-key",
	}
	wechatAccountName = ""
	convertUpload = true
	imageRef := converter.ImageRef{Index: 0, Type: converter.ImageTypeLocal, Original: "missing.png", Placeholder: "<!-- IMG:0 -->"}
	conv := &fakeConverter{
		images: []converter.ImageRef{imageRef},
		result: &converter.ConvertResult{
			Success: true,
			Mode:    converter.ModeAPI,
			Theme:   "default",
			HTML:    `<img src="missing.png">`,
			Images:  []converter.ImageRef{imageRef},
		},
	}
	processor := &fakeImageProcessor{}
	newMarkdownConverter = func() converter.Converter { return conv }
	newImageProcessor = func() imageProcessor { return processor }

	validatorCalls := 0
	validateAPIKeyForWeChatAccount = func(string) error {
		validatorCalls++
		return nil
	}
	dir := t.TempDir()
	markdownPath := filepath.Join(dir, "article.md")
	if err := os.WriteFile(markdownPath, []byte("# Title\n\n![image](missing.png)\n"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	err := runConvert(nil, []string{markdownPath})
	cliErr, ok := err.(*cliError)
	if !ok || cliErr.Code != codeConvertImageFailed {
		t.Fatalf("runConvert() error = %#v, want %s", err, codeConvertImageFailed)
	}
	if validatorCalls != 0 || len(conv.reqs) != 0 || processor.totalCalls() != 0 {
		t.Fatalf("invalid local asset caused effects: validator=%d convert=%d assets=%d", validatorCalls, len(conv.reqs), processor.totalCalls())
	}
}
