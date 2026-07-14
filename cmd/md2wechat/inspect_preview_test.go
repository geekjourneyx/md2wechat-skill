package main

import (
	"bytes"
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

func TestRunPreviewWritesExactConverterHTML(t *testing.T) {
	oldCfg, oldLog, oldJSON := cfg, log, jsonOutput
	oldMode, oldTheme := previewMode, previewTheme
	oldFont, oldBackground := previewFontSize, previewBackgroundType
	oldOutput := previewOutput
	oldTitle, oldAuthor, oldDigest := previewTitle, previewAuthor, previewDigest
	oldCover, oldUpload, oldDraft := previewCover, previewUpload, previewDraft
	oldNewConverter := newMarkdownConverter
	t.Cleanup(func() {
		cfg, log, jsonOutput = oldCfg, oldLog, oldJSON
		previewMode, previewTheme = oldMode, oldTheme
		previewFontSize, previewBackgroundType = oldFont, oldBackground
		previewOutput = oldOutput
		previewTitle, previewAuthor, previewDigest = oldTitle, oldAuthor, oldDigest
		previewCover, previewUpload, previewDraft = oldCover, oldUpload, oldDraft
		newMarkdownConverter = oldNewConverter
	})

	cfg = &config.Config{MD2WechatAPIKey: "api-key"}
	log = zap.NewNop()
	jsonOutput = true
	previewMode = "api"
	previewTheme = "default"
	previewFontSize = "medium"
	previewBackgroundType = "none"
	outputFile := filepath.Join(t.TempDir(), "preview.html")
	previewOutput = outputFile
	previewTitle = ""
	previewAuthor = ""
	previewDigest = ""
	previewCover = ""
	previewUpload = false
	previewDraft = false
	convertedHTML := "\n<!doctype html>\n<section data-contract=\"纯净预览\">\n  <p>保留这些字节。</p>\n</section>\n"
	newMarkdownConverter = func() converter.Converter {
		return &fakeConverter{
			result: &converter.ConvertResult{
				Success: true,
				Mode:    converter.ModeAPI,
				Theme:   "default",
				HTML:    convertedHTML,
			},
		}
	}

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	markdown := "# PREVIEW_SOURCE_SENTINEL\n\nOriginal Markdown must not leak."
	if err := os.WriteFile(markdownPath, []byte(markdown), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	stdout := captureStdout(t, func() {
		if err := previewCmd.RunE(previewCmd, []string{markdownPath}); err != nil {
			t.Fatalf("preview RunE() error = %v", err)
		}
	})
	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["code"] != codePreviewReady || response["status"] != string(action.StatusCompleted) {
		t.Fatalf("response = %#v", response)
	}
	responseData, _ := response["data"].(map[string]any)
	if responseData["output_file"] != outputFile {
		t.Fatalf("output_file = %#v, want %q", responseData["output_file"], outputFile)
	}
	if _, ok := responseData["inspect"].(map[string]any); !ok {
		t.Fatalf("inspect diagnostics missing: %#v", responseData)
	}
	render, _ := responseData["render"].(map[string]any)
	if render["fidelity"] != inspectpkg.PreviewFidelityExact || render["exact_html"] != true {
		t.Fatalf("render = %#v", render)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("read preview: %v", err)
	}
	if !bytes.Equal(data, []byte(convertedHTML)) {
		t.Fatalf("preview bytes differ from converter HTML\n got: %q\nwant: %q", data, convertedHTML)
	}
	for _, forbidden := range []string{markdownPath, "Readiness", "<aside", "--panel", markdown} {
		if bytes.Contains(data, []byte(forbidden)) {
			t.Fatalf("preview contains wrapper/source content %q: %s", forbidden, data)
		}
	}
	if previewOutput != "" {
		t.Fatalf("previewOutput leaked explicit path %q", previewOutput)
	}
}

func TestRunPreviewAIRequestWritesNoFile(t *testing.T) {
	oldCfg, oldLog, oldJSON := cfg, log, jsonOutput
	oldMode, oldTheme := previewMode, previewTheme
	oldFont, oldBackground := previewFontSize, previewBackgroundType
	oldOutput := previewOutput
	oldNewConverter := newMarkdownConverter
	t.Cleanup(func() {
		cfg, log, jsonOutput = oldCfg, oldLog, oldJSON
		previewMode, previewTheme = oldMode, oldTheme
		previewFontSize, previewBackgroundType = oldFont, oldBackground
		previewOutput = oldOutput
		newMarkdownConverter = oldNewConverter
	})

	cfg = &config.Config{}
	log = zap.NewNop()
	jsonOutput = true
	previewMode = "ai"
	previewTheme = "autumn-warm"
	previewFontSize = "medium"
	previewBackgroundType = "none"
	outputFile := filepath.Join(t.TempDir(), "preview-ai.html")
	previewOutput = outputFile
	newMarkdownConverter = func() converter.Converter {
		return &fakeConverter{
			result: &converter.ConvertResult{
				Success: true,
				Mode:    converter.ModeAI,
				Status:  action.StatusActionRequired,
				Prompt:  "render this exact article",
			},
		}
	}

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(markdownPath, []byte("# 标题\n\n正文"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	stdout := captureStdout(t, func() {
		if err := previewCmd.RunE(previewCmd, []string{markdownPath}); err != nil {
			t.Fatalf("preview RunE() error = %v", err)
		}
	})
	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["code"] != codePreviewActionRequired || response["status"] != string(action.StatusActionRequired) {
		t.Fatalf("response = %#v", response)
	}
	data, _ := response["data"].(map[string]any)
	if data["prompt"] != "render this exact article" || data["output_file"] != "" {
		t.Fatalf("data = %#v", data)
	}
	if _, err := os.Stat(outputFile); !os.IsNotExist(err) {
		t.Fatalf("preview file must not exist, stat error = %v", err)
	}
	if previewOutput != "" {
		t.Fatalf("previewOutput leaked stale path %q", previewOutput)
	}
}

func TestRunPreviewAPIFailureWritesNoFile(t *testing.T) {
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
	outputFile := filepath.Join(t.TempDir(), "preview-failed.html")
	previewOutput = outputFile
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

	err := previewCmd.RunE(previewCmd, []string{markdownPath})
	if err == nil {
		t.Fatal("expected API conversion failure")
	}
	cliErr, ok := err.(*cliError)
	if !ok || cliErr.Code != codePreviewFailed {
		t.Fatalf("error = %#v", err)
	}
	if _, err := os.Stat(outputFile); !os.IsNotExist(err) {
		t.Fatalf("preview file must not exist, stat error = %v", err)
	}
	if previewOutput != "" {
		t.Fatalf("previewOutput leaked stale path %q", previewOutput)
	}
}

func TestRunPreviewNilConverterResultWritesNoFile(t *testing.T) {
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
	outputFile := filepath.Join(t.TempDir(), "preview-nil.html")
	previewOutput = outputFile
	newMarkdownConverter = func() converter.Converter {
		return &fakeConverter{result: nil}
	}

	markdownPath := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(markdownPath, []byte("# 标题\n"), 0600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	err := previewCmd.RunE(previewCmd, []string{markdownPath})
	if err == nil {
		t.Fatal("expected nil conversion failure")
	}
	cliErr, ok := err.(*cliError)
	if !ok || cliErr.Code != codePreviewFailed {
		t.Fatalf("error = %#v", err)
	}
	if _, err := os.Stat(outputFile); !os.IsNotExist(err) {
		t.Fatalf("preview file must not exist, stat error = %v", err)
	}
	if previewOutput != "" {
		t.Fatalf("previewOutput leaked stale path %q", previewOutput)
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

	result, err := runPreview(markdownPath)
	if err != nil {
		t.Fatalf("runPreview() error = %v", err)
	}
	if result.OutputFile == "" {
		t.Fatal("expected temp output path")
	}
	t.Cleanup(func() { _ = os.Remove(result.OutputFile) })
	data, err := os.ReadFile(result.OutputFile)
	if err != nil {
		t.Fatalf("preview output stat: %v", err)
	}
	if string(data) != "<p>temp preview</p>" {
		t.Fatalf("preview bytes = %q", data)
	}
	if previewOutput != "" {
		t.Fatalf("previewOutput leaked temp path %q", previewOutput)
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

	_, err := runPreview(markdownPath)
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
	if previewOutput != "" {
		t.Fatalf("previewOutput leaked invalid path %q", previewOutput)
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

func TestInspectCommandAICustomPromptAllowsDefaultTheme(t *testing.T) {
	oldCfg, oldJSON, oldStrict := cfg, jsonOutput, inspectStrict
	oldExit := exitFunc
	oldMode, oldTheme := inspectMode, inspectTheme
	oldFont, oldBackground := inspectFontSize, inspectBackgroundType
	oldCustomPrompt := inspectCustomPrompt
	oldTitle, oldAuthor, oldDigest := inspectTitle, inspectAuthor, inspectDigest
	oldCover, oldCoverMediaID := inspectCover, inspectCoverMediaID
	oldUpload, oldDraft := inspectUpload, inspectDraft
	t.Cleanup(func() {
		cfg, jsonOutput, inspectStrict = oldCfg, oldJSON, oldStrict
		exitFunc = oldExit
		inspectMode, inspectTheme = oldMode, oldTheme
		inspectFontSize, inspectBackgroundType = oldFont, oldBackground
		inspectCustomPrompt = oldCustomPrompt
		inspectTitle, inspectAuthor, inspectDigest = oldTitle, oldAuthor, oldDigest
		inspectCover, inspectCoverMediaID = oldCover, oldCoverMediaID
		inspectUpload, inspectDraft = oldUpload, oldDraft
	})

	cfg = &config.Config{}
	jsonOutput = true
	inspectStrict = false
	inspectMode = "ai"
	inspectTheme = "default"
	inspectFontSize = "medium"
	inspectBackgroundType = "none"
	inspectCustomPrompt = "Use a compact editorial layout."
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
		Data struct {
			Context struct {
				CustomPrompt bool `json:"custom_prompt"`
			} `json:"context"`
			Readiness struct {
				Targets struct {
					Convert string `json:"convert"`
				} `json:"targets"`
				Blockers []struct {
					Code string `json:"code"`
				} `json:"blockers"`
			} `json:"readiness"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if !response.Data.Context.CustomPrompt {
		t.Fatal("context.custom_prompt = false")
	}
	if response.Data.Readiness.Targets.Convert != "ready" {
		t.Fatalf("readiness.targets.convert = %q, blockers = %#v", response.Data.Readiness.Targets.Convert, response.Data.Readiness.Blockers)
	}
	for _, blocker := range response.Data.Readiness.Blockers {
		if blocker.Code == "THEME_MODE_MISMATCH" {
			t.Fatalf("unexpected THEME_MODE_MISMATCH blocker: %#v", response.Data.Readiness.Blockers)
		}
	}
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

	_, err := runPreview(markdownPath)
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
