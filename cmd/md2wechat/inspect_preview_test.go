package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/action"
	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"github.com/geekjourneyx/md2wechat-skill/internal/converter"
	inspectpkg "github.com/geekjourneyx/md2wechat-skill/internal/inspect"
	"go.uber.org/zap"
)

func TestRunPreviewWritesExactHTMLPreviewFile(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldMode, oldTheme := previewMode, previewTheme
	oldFont, oldBackground := previewFontSize, previewBackgroundType
	oldOutput := previewOutput
	oldTitle, oldAuthor, oldDigest := previewTitle, previewAuthor, previewDigest
	oldCover, oldUpload, oldDraft := previewCover, previewUpload, previewDraft
	oldNewConverter := newMarkdownConverter
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		previewMode, previewTheme = oldMode, oldTheme
		previewFontSize, previewBackgroundType = oldFont, oldBackground
		previewOutput = oldOutput
		previewTitle, previewAuthor, previewDigest = oldTitle, oldAuthor, oldDigest
		previewCover, previewUpload, previewDraft = oldCover, oldUpload, oldDraft
		newMarkdownConverter = oldNewConverter
	})

	cfg = &config.Config{MD2WechatAPIKey: "api-key"}
	log = zap.NewNop()
	previewMode = "api"
	previewTheme = "default"
	previewFontSize = "medium"
	previewBackgroundType = "none"
	previewOutput = filepath.Join(t.TempDir(), "preview.html")
	previewTitle = ""
	previewAuthor = ""
	previewDigest = ""
	previewCover = ""
	previewUpload = false
	previewDraft = false
	newMarkdownConverter = func() converter.Converter {
		return &fakeConverter{
			result: &converter.ConvertResult{
				Success: true,
				Mode:    converter.ModeAPI,
				Theme:   "default",
				HTML:    "<p>exact preview</p>",
			},
		}
	}

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(markdownPath, []byte("# 标题\n\n正文"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	result, render, err := runPreview(markdownPath)
	if err != nil {
		t.Fatalf("runPreview() error = %v", err)
	}
	if render["fidelity"] != "exact" || render["exact_html"] != true {
		t.Fatalf("render = %#v", render)
	}
	if result.Metadata.Title.Value != "标题" {
		t.Fatalf("title = %#v", result.Metadata.Title)
	}

	data, err := os.ReadFile(previewOutput)
	if err != nil {
		t.Fatalf("read preview: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "<p>exact preview</p>") {
		t.Fatalf("preview file missing exact HTML: %s", content)
	}
}

func TestRunPreviewWritesDegradedPreviewForAIMode(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldMode, oldTheme := previewMode, previewTheme
	oldFont, oldBackground := previewFontSize, previewBackgroundType
	oldOutput := previewOutput
	oldNewConverter := newMarkdownConverter
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		previewMode, previewTheme = oldMode, oldTheme
		previewFontSize, previewBackgroundType = oldFont, oldBackground
		previewOutput = oldOutput
		newMarkdownConverter = oldNewConverter
	})

	cfg = &config.Config{}
	log = zap.NewNop()
	previewMode = "ai"
	previewTheme = "autumn-warm"
	previewFontSize = "medium"
	previewBackgroundType = "none"
	previewOutput = filepath.Join(t.TempDir(), "preview-ai.html")
	newMarkdownConverter = func() converter.Converter {
		return &fakeConverter{
			result: &converter.ConvertResult{
				Success: true,
				Mode:    converter.ModeAI,
				Status:  action.StatusActionRequired,
				Prompt:  "do work",
			},
		}
	}

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(markdownPath, []byte("# 标题\n\n正文"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	_, render, err := runPreview(markdownPath)
	if err != nil {
		t.Fatalf("runPreview() error = %v", err)
	}
	if render["fidelity"] != "degraded" || render["exact_html"] != false {
		t.Fatalf("render = %#v", render)
	}

	data, err := os.ReadFile(previewOutput)
	if err != nil {
		t.Fatalf("read preview: %v", err)
	}
	if !strings.Contains(string(data), "AI mode currently yields a prompt/request instead of final HTML") {
		t.Fatalf("preview file missing degraded message: %s", string(data))
	}
}

func TestRunPreviewDegradesWhenAPIRenderFails(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldMode, oldTheme := previewMode, previewTheme
	oldFont, oldBackground := previewFontSize, previewBackgroundType
	oldOutput := previewOutput
	oldNewConverter := newMarkdownConverter
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		previewMode, previewTheme = oldMode, oldTheme
		previewFontSize, previewBackgroundType = oldFont, oldBackground
		previewOutput = oldOutput
		newMarkdownConverter = oldNewConverter
	})

	cfg = &config.Config{MD2WechatAPIKey: "api-key"}
	log = zap.NewNop()
	previewMode = "api"
	previewTheme = "default"
	previewFontSize = "medium"
	previewBackgroundType = "none"
	previewOutput = filepath.Join(t.TempDir(), "preview-failed.html")
	newMarkdownConverter = func() converter.Converter {
		return &fakeConverter{
			result: &converter.ConvertResult{
				Success: false,
				Mode:    converter.ModeAPI,
				Error:   "upstream render failed",
			},
		}
	}

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(markdownPath, []byte("# 标题\n\n正文"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	result, render, err := runPreview(markdownPath)
	if err != nil {
		t.Fatalf("runPreview() error = %v", err)
	}
	if result.Readiness.PreviewFidelity != "degraded" {
		t.Fatalf("preview_fidelity = %q", result.Readiness.PreviewFidelity)
	}
	if render["fidelity"] != "degraded" || render["exact_html"] != false || render["error"] != "upstream render failed" {
		t.Fatalf("render = %#v", render)
	}

	data, err := os.ReadFile(previewOutput)
	if err != nil {
		t.Fatalf("read preview: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Preview degraded: exact HTML could not be rendered in the current environment.") {
		t.Fatalf("preview file missing degraded banner: %s", content)
	}
	if !strings.Contains(content, "Render error: upstream render failed") {
		t.Fatalf("preview file missing render error: %s", content)
	}
}

func TestRunPreviewUsesTempFileWhenOutputUnset(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldMode, oldTheme := previewMode, previewTheme
	oldFont, oldBackground := previewFontSize, previewBackgroundType
	oldOutput := previewOutput
	oldNewConverter := newMarkdownConverter
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		previewMode, previewTheme = oldMode, oldTheme
		previewFontSize, previewBackgroundType = oldFont, oldBackground
		previewOutput = oldOutput
		newMarkdownConverter = oldNewConverter
	})

	cfg = &config.Config{MD2WechatAPIKey: "api-key"}
	log = zap.NewNop()
	previewMode = "api"
	previewTheme = "default"
	previewFontSize = "medium"
	previewBackgroundType = "none"
	previewOutput = ""
	newMarkdownConverter = func() converter.Converter {
		return &fakeConverter{
			result: &converter.ConvertResult{
				Success: true,
				Mode:    converter.ModeAPI,
				HTML:    "<p>temp preview</p>",
			},
		}
	}

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(markdownPath, []byte("# 标题\n"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	_, _, err := runPreview(markdownPath)
	if err != nil {
		t.Fatalf("runPreview() error = %v", err)
	}
	if previewOutput == "" {
		t.Fatal("expected previewOutput to be populated")
	}
	if _, err := os.Stat(previewOutput); err != nil {
		t.Fatalf("preview output stat: %v", err)
	}
}

func TestRunPreviewReturnsPreviewFailedForInvalidOutputPath(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldMode, oldTheme := previewMode, previewTheme
	oldFont, oldBackground := previewFontSize, previewBackgroundType
	oldOutput := previewOutput
	oldNewConverter := newMarkdownConverter
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		previewMode, previewTheme = oldMode, oldTheme
		previewFontSize, previewBackgroundType = oldFont, oldBackground
		previewOutput = oldOutput
		newMarkdownConverter = oldNewConverter
	})

	cfg = &config.Config{MD2WechatAPIKey: "api-key"}
	log = zap.NewNop()
	previewMode = "api"
	previewTheme = "default"
	previewFontSize = "medium"
	previewBackgroundType = "none"
	previewOutput = filepath.Join(t.TempDir(), "missing", "preview.html")
	newMarkdownConverter = func() converter.Converter {
		return &fakeConverter{
			result: &converter.ConvertResult{
				Success: true,
				Mode:    converter.ModeAPI,
				HTML:    "<p>preview</p>",
			},
		}
	}

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(markdownPath, []byte("# 标题\n"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	_, _, err := runPreview(markdownPath)
	if err == nil {
		t.Fatal("expected error for invalid output path")
	}
	cliErr, ok := err.(*cliError)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if cliErr.Code != codePreviewFailed {
		t.Fatalf("error code = %q", cliErr.Code)
	}
}

func TestInspectCommandStrictExitsWithCodeTwoWhenErrorsExist(t *testing.T) {
	oldCfg, oldJSON, oldStrict := cfg, jsonOutput, inspectStrict
	oldExit := exitFunc
	oldMode, oldTheme := inspectMode, inspectTheme
	oldFont, oldBackground := inspectFontSize, inspectBackgroundType
	oldTitle, oldAuthor, oldDigest := inspectTitle, inspectAuthor, inspectDigest
	oldCover, oldUpload, oldDraft := inspectCover, inspectUpload, inspectDraft
	t.Cleanup(func() {
		cfg, jsonOutput, inspectStrict = oldCfg, oldJSON, oldStrict
		exitFunc = oldExit
		inspectMode, inspectTheme = oldMode, oldTheme
		inspectFontSize, inspectBackgroundType = oldFont, oldBackground
		inspectTitle, inspectAuthor, inspectDigest = oldTitle, oldAuthor, oldDigest
		inspectCover, inspectUpload, inspectDraft = oldCover, oldUpload, oldDraft
	})

	cfg = &config.Config{}
	jsonOutput = true
	inspectStrict = true
	inspectMode = "api"
	inspectTheme = "default"
	inspectFontSize = "medium"
	inspectBackgroundType = "none"
	inspectTitle = ""
	inspectAuthor = ""
	inspectDigest = ""
	inspectCover = ""
	inspectUpload = false
	inspectDraft = false

	exitCode := 0
	exitFunc = func(code int) { exitCode = code }

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(markdownPath, []byte("# 标题\n"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	stdout := captureStdout(t, func() {
		if err := inspectCmd.RunE(inspectCmd, []string{markdownPath}); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})
	if exitCode != 2 {
		t.Fatalf("exit code = %d", exitCode)
	}

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["code"] != codeInspectCompleted || response["status"] != "completed" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInspectCommandJSONIncludesReadinessBlockerMapping(t *testing.T) {
	oldCfg, oldJSON, oldStrict := cfg, jsonOutput, inspectStrict
	oldExit := exitFunc
	oldMode, oldTheme := inspectMode, inspectTheme
	oldFont, oldBackground := inspectFontSize, inspectBackgroundType
	oldTitle, oldAuthor, oldDigest := inspectTitle, inspectAuthor, inspectDigest
	oldCover, oldCoverMediaID := inspectCover, inspectCoverMediaID
	oldUpload, oldDraft := inspectUpload, inspectDraft
	t.Cleanup(func() {
		cfg, jsonOutput, inspectStrict = oldCfg, oldJSON, oldStrict
		exitFunc = oldExit
		inspectMode, inspectTheme = oldMode, oldTheme
		inspectFontSize, inspectBackgroundType = oldFont, oldBackground
		inspectTitle, inspectAuthor, inspectDigest = oldTitle, oldAuthor, oldDigest
		inspectCover, inspectCoverMediaID = oldCover, oldCoverMediaID
		inspectUpload, inspectDraft = oldUpload, oldDraft
	})

	cfg = &config.Config{
		MD2WechatAPIKey: "api-key",
		WechatAppID:     "appid",
		WechatSecret:    "secret",
	}
	jsonOutput = true
	inspectStrict = false
	inspectMode = "api"
	inspectTheme = "default"
	inspectFontSize = "medium"
	inspectBackgroundType = "none"
	inspectTitle = ""
	inspectAuthor = ""
	inspectDigest = ""
	inspectCover = ""
	inspectCoverMediaID = ""
	inspectUpload = false
	inspectDraft = true
	exitFunc = func(code int) {
		t.Fatalf("unexpected exitFunc(%d)", code)
	}

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(markdownPath, []byte("# 标题\n\n正文"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	stdout := captureStdout(t, func() {
		if err := inspectCmd.RunE(inspectCmd, []string{markdownPath}); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})

	var response struct {
		Code   string `json:"code"`
		Status string `json:"status"`
		Data   struct {
			Readiness struct {
				SchemaVersion string `json:"schema_version"`
				Targets       struct {
					Draft string `json:"draft"`
				} `json:"targets"`
				Blockers []struct {
					Code   string   `json:"code"`
					Blocks []string `json:"blocks"`
				} `json:"blockers"`
			} `json:"readiness"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response.Code != codeInspectCompleted {
		t.Fatalf("code = %q", response.Code)
	}
	if response.Status != "completed" {
		t.Fatalf("status = %q", response.Status)
	}
	if response.Data.Readiness.SchemaVersion != "v1" {
		t.Fatalf("readiness.schema_version = %q", response.Data.Readiness.SchemaVersion)
	}
	if response.Data.Readiness.Targets.Draft != "blocked" {
		t.Fatalf("readiness.targets.draft = %q", response.Data.Readiness.Targets.Draft)
	}

	for _, blocker := range response.Data.Readiness.Blockers {
		if blocker.Code == "MISSING_COVER" {
			if len(blocker.Blocks) != 1 || blocker.Blocks[0] != "draft" {
				t.Fatalf("MISSING_COVER blocks = %#v", blocker.Blocks)
			}
			return
		}
	}
	t.Fatalf("MISSING_COVER blocker not found: %#v", response.Data.Readiness.Blockers)
}

func TestInspectCommandJSONBlocksConvertForThemeMismatch(t *testing.T) {
	oldCfg, oldJSON, oldStrict := cfg, jsonOutput, inspectStrict
	oldExit := exitFunc
	oldMode, oldTheme := inspectMode, inspectTheme
	oldFont, oldBackground := inspectFontSize, inspectBackgroundType
	oldTitle, oldAuthor, oldDigest := inspectTitle, inspectAuthor, inspectDigest
	oldCover, oldCoverMediaID := inspectCover, inspectCoverMediaID
	oldUpload, oldDraft := inspectUpload, inspectDraft
	t.Cleanup(func() {
		cfg, jsonOutput, inspectStrict = oldCfg, oldJSON, oldStrict
		exitFunc = oldExit
		inspectMode, inspectTheme = oldMode, oldTheme
		inspectFontSize, inspectBackgroundType = oldFont, oldBackground
		inspectTitle, inspectAuthor, inspectDigest = oldTitle, oldAuthor, oldDigest
		inspectCover, inspectCoverMediaID = oldCover, oldCoverMediaID
		inspectUpload, inspectDraft = oldUpload, oldDraft
	})

	cfg = &config.Config{MD2WechatAPIKey: "api-key"}
	jsonOutput = true
	inspectStrict = false
	inspectMode = "api"
	inspectTheme = "autumn-warm"
	inspectFontSize = "medium"
	inspectBackgroundType = "none"
	inspectTitle = ""
	inspectAuthor = ""
	inspectDigest = ""
	inspectCover = ""
	inspectCoverMediaID = ""
	inspectUpload = false
	inspectDraft = false
	exitFunc = func(code int) {
		t.Fatalf("unexpected exitFunc(%d)", code)
	}

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(markdownPath, []byte("# 标题\n\n正文"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	stdout := captureStdout(t, func() {
		if err := inspectCmd.RunE(inspectCmd, []string{markdownPath}); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})

	var response struct {
		Code string `json:"code"`
		Data struct {
			Readiness struct {
				Targets struct {
					Convert string `json:"convert"`
					Upload  string `json:"upload"`
					Draft   string `json:"draft"`
				} `json:"targets"`
				Blockers []struct {
					Code   string   `json:"code"`
					Blocks []string `json:"blocks"`
				} `json:"blockers"`
			} `json:"readiness"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response.Code != codeInspectCompleted {
		t.Fatalf("code = %q", response.Code)
	}
	targets := response.Data.Readiness.Targets
	if targets.Convert != "blocked" {
		t.Fatalf("readiness.targets.convert = %q", targets.Convert)
	}
	if targets.Upload != "not_requested" {
		t.Fatalf("readiness.targets.upload = %q", targets.Upload)
	}
	if targets.Draft != "not_requested" {
		t.Fatalf("readiness.targets.draft = %q", targets.Draft)
	}

	for _, blocker := range response.Data.Readiness.Blockers {
		if blocker.Code == "THEME_MODE_MISMATCH" {
			want := []string{"convert", "upload", "draft"}
			if len(blocker.Blocks) != len(want) {
				t.Fatalf("THEME_MODE_MISMATCH blocks = %#v", blocker.Blocks)
			}
			for i := range want {
				if blocker.Blocks[i] != want[i] {
					t.Fatalf("THEME_MODE_MISMATCH blocks = %#v", blocker.Blocks)
				}
			}
			return
		}
	}
	t.Fatalf("THEME_MODE_MISMATCH blocker not found: %#v", response.Data.Readiness.Blockers)
}

func TestInspectCommandStrictDoesNotExitTwoForWarnOnlyChecks(t *testing.T) {
	oldCfg, oldJSON, oldStrict := cfg, jsonOutput, inspectStrict
	oldExit := exitFunc
	oldMode, oldTheme := inspectMode, inspectTheme
	oldFont, oldBackground := inspectFontSize, inspectBackgroundType
	oldTitle, oldAuthor, oldDigest := inspectTitle, inspectAuthor, inspectDigest
	oldCover, oldUpload, oldDraft := inspectCover, inspectUpload, inspectDraft
	t.Cleanup(func() {
		cfg, jsonOutput, inspectStrict = oldCfg, oldJSON, oldStrict
		exitFunc = oldExit
		inspectMode, inspectTheme = oldMode, oldTheme
		inspectFontSize, inspectBackgroundType = oldFont, oldBackground
		inspectTitle, inspectAuthor, inspectDigest = oldTitle, oldAuthor, oldDigest
		inspectCover, inspectUpload, inspectDraft = oldCover, oldUpload, oldDraft
	})

	cfg = &config.Config{MD2WechatAPIKey: "api-key"}
	jsonOutput = false
	inspectStrict = true
	inspectMode = "api"
	inspectTheme = "default"
	inspectFontSize = "medium"
	inspectBackgroundType = "none"
	inspectTitle = "最终标题"
	inspectAuthor = ""
	inspectDigest = ""
	inspectCover = ""
	inspectUpload = false
	inspectDraft = false

	exitCode := 0
	exitFunc = func(code int) { exitCode = code }

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(markdownPath, []byte("# 最终标题\n\n正文"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	stdout := captureStdout(t, func() {
		if err := inspectCmd.RunE(inspectCmd, []string{markdownPath}); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})
	if exitCode != 0 {
		t.Fatalf("exit code = %d", exitCode)
	}
	if !strings.Contains(string(stdout), "WARN DUPLICATE_H1") {
		t.Fatalf("inspect output = %s", stdout)
	}
}

func TestInspectCommandStrictDoesNotExitTwoWhenNoErrorChecksExist(t *testing.T) {
	oldCfg, oldJSON, oldStrict := cfg, jsonOutput, inspectStrict
	oldExit := exitFunc
	oldMode, oldTheme := inspectMode, inspectTheme
	oldFont, oldBackground := inspectFontSize, inspectBackgroundType
	oldTitle, oldAuthor, oldDigest := inspectTitle, inspectAuthor, inspectDigest
	oldCover, oldUpload, oldDraft := inspectCover, inspectUpload, inspectDraft
	t.Cleanup(func() {
		cfg, jsonOutput, inspectStrict = oldCfg, oldJSON, oldStrict
		exitFunc = oldExit
		inspectMode, inspectTheme = oldMode, oldTheme
		inspectFontSize, inspectBackgroundType = oldFont, oldBackground
		inspectTitle, inspectAuthor, inspectDigest = oldTitle, oldAuthor, oldDigest
		inspectCover, inspectUpload, inspectDraft = oldCover, oldUpload, oldDraft
	})

	cfg = &config.Config{MD2WechatAPIKey: "api-key"}
	jsonOutput = false
	inspectStrict = true
	inspectMode = "api"
	inspectTheme = "default"
	inspectFontSize = "medium"
	inspectBackgroundType = "none"
	inspectTitle = ""
	inspectAuthor = ""
	inspectDigest = ""
	inspectCover = ""
	inspectUpload = false
	inspectDraft = false

	exitCode := 0
	exitFunc = func(code int) { exitCode = code }

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	markdown := strings.Join([]string{
		"---",
		"title: Frontmatter 标题",
		"---",
		"",
		"正文，不含一级标题。",
	}, "\n")
	if err := os.WriteFile(markdownPath, []byte(markdown), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	stdout := captureStdout(t, func() {
		if err := inspectCmd.RunE(inspectCmd, []string{markdownPath}); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})
	if exitCode != 0 {
		t.Fatalf("exit code = %d", exitCode)
	}
	if !strings.Contains(string(stdout), "- none") {
		t.Fatalf("inspect output = %s", stdout)
	}
}

func TestBuildCapabilitiesDataIncludesInspectAndPreview(t *testing.T) {
	data, err := buildCapabilitiesData()
	if err != nil {
		t.Fatalf("buildCapabilitiesData() error = %v", err)
	}
	commands, ok := data["commands"].([]string)
	if !ok {
		t.Fatalf("commands type = %T", data["commands"])
	}
	if !contains(commands, "inspect") || !contains(commands, "preview") {
		t.Fatalf("commands = %#v", commands)
	}
}

func TestInspectJSONSuppressesConfigBannerAndLogsOnStderr(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldJSON := jsonOutput
	oldMode, oldTheme := inspectMode, inspectTheme
	oldFont, oldBackground := inspectFontSize, inspectBackgroundType
	oldTitle, oldAuthor, oldDigest := inspectTitle, inspectAuthor, inspectDigest
	oldCover, oldUpload, oldDraft := inspectCover, inspectUpload, inspectDraft
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		jsonOutput = oldJSON
		inspectMode, inspectTheme = oldMode, oldTheme
		inspectFontSize, inspectBackgroundType = oldFont, oldBackground
		inspectTitle, inspectAuthor, inspectDigest = oldTitle, oldAuthor, oldDigest
		inspectCover, inspectUpload, inspectDraft = oldCover, oldUpload, oldDraft
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

	cfg, log = nil, nil
	jsonOutput = true
	inspectMode = "ai"
	inspectTheme = "default"
	inspectFontSize = "medium"
	inspectBackgroundType = "none"
	inspectTitle = ""
	inspectAuthor = ""
	inspectDigest = ""
	inspectCover = ""
	inspectUpload = false
	inspectDraft = false

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(markdownPath, []byte("# 标题\n"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	stderr := captureStderr(t, func() {
		stdout := captureStdout(t, func() {
			if err := inspectCmd.RunE(inspectCmd, []string{markdownPath}); err != nil {
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

func TestRunInspectWithInputRejectsInvalidMode(t *testing.T) {
	oldCfg := cfg
	t.Cleanup(func() {
		cfg = oldCfg
	})

	cfg = &config.Config{}
	_, err := runInspectWithInput(filepath.Join(t.TempDir(), "article.md"), "# 标题\n", inspectpkg.Input{
		Mode: "foo",
	})
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
	cliErr, ok := err.(*cliError)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if cliErr.Code != codeConvertInvalid || !strings.Contains(cliErr.Error(), "invalid convert mode") {
		t.Fatalf("error = %#v", cliErr)
	}
}

func TestRunInspectUsesCoverMediaIDForDraftReadiness(t *testing.T) {
	oldCfg := cfg
	oldMode, oldTheme := inspectMode, inspectTheme
	oldFont, oldBackground := inspectFontSize, inspectBackgroundType
	oldTitle, oldAuthor, oldDigest := inspectTitle, inspectAuthor, inspectDigest
	oldCover, oldCoverMediaID := inspectCover, inspectCoverMediaID
	oldUpload, oldDraft := inspectUpload, inspectDraft
	t.Cleanup(func() {
		cfg = oldCfg
		inspectMode, inspectTheme = oldMode, oldTheme
		inspectFontSize, inspectBackgroundType = oldFont, oldBackground
		inspectTitle, inspectAuthor, inspectDigest = oldTitle, oldAuthor, oldDigest
		inspectCover, inspectCoverMediaID = oldCover, oldCoverMediaID
		inspectUpload, inspectDraft = oldUpload, oldDraft
	})

	cfg = &config.Config{
		MD2WechatAPIKey: "api-key",
		WechatAppID:     "appid",
		WechatSecret:    "secret",
	}
	inspectMode = "api"
	inspectTheme = "default"
	inspectFontSize = "medium"
	inspectBackgroundType = "none"
	inspectTitle = ""
	inspectAuthor = ""
	inspectDigest = ""
	inspectCover = ""
	inspectCoverMediaID = "existing-cover-id"
	inspectUpload = false
	inspectDraft = true

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(markdownPath, []byte("# 标题\n"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	result, err := runInspect(markdownPath)
	if err != nil {
		t.Fatalf("runInspect() error = %v", err)
	}
	if !result.Readiness.DraftReady {
		t.Fatalf("draft target state = %#v", result.Readiness)
	}
	if hasErrorCheck(result.Checks) {
		t.Fatalf("checks = %#v", result.Checks)
	}
}

func TestRunPreviewRejectsInvalidMode(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldMode, oldTheme := previewMode, previewTheme
	oldFont, oldBackground := previewFontSize, previewBackgroundType
	oldOutput := previewOutput
	oldTitle, oldAuthor, oldDigest := previewTitle, previewAuthor, previewDigest
	oldCover, oldUpload, oldDraft := previewCover, previewUpload, previewDraft
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		previewMode, previewTheme = oldMode, oldTheme
		previewFontSize, previewBackgroundType = oldFont, oldBackground
		previewOutput = oldOutput
		previewTitle, previewAuthor, previewDigest = oldTitle, oldAuthor, oldDigest
		previewCover, previewUpload, previewDraft = oldCover, oldUpload, oldDraft
	})

	cfg = &config.Config{}
	log = zap.NewNop()
	previewMode = "foo"
	previewTheme = "default"
	previewFontSize = "medium"
	previewBackgroundType = "none"
	previewOutput = filepath.Join(t.TempDir(), "preview.html")
	previewTitle = ""
	previewAuthor = ""
	previewDigest = ""
	previewCover = ""
	previewUpload = false
	previewDraft = false

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(markdownPath, []byte("# 标题\n"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	_, _, err := runPreview(markdownPath)
	if err == nil {
		t.Fatal("expected error for invalid preview mode")
	}
	cliErr, ok := err.(*cliError)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if cliErr.Code != codeConvertInvalid || !strings.Contains(cliErr.Error(), "invalid convert mode") {
		t.Fatalf("error = %#v", cliErr)
	}
}
