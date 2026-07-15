package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"github.com/geekjourneyx/md2wechat-skill/internal/converter"
	"github.com/geekjourneyx/md2wechat-skill/internal/publish"
	"go.uber.org/zap"
)

func TestValidateConvertConfigRejectsInvalidMode(t *testing.T) {
	oldCfg, oldMode := cfg, convertMode
	t.Cleanup(func() {
		cfg = oldCfg
		convertMode = oldMode
	})

	cfg = &config.Config{}
	convertMode = "invalid"

	if err := validateConvertConfig(); err == nil || !strings.Contains(err.Error(), "invalid convert mode") {
		t.Fatalf("validateConvertConfig() error = %v", err)
	}
}

func TestValidateConvertConfigRequiresAPIKeyInAPIMode(t *testing.T) {
	oldCfg, oldMode := cfg, convertMode
	oldUpload, oldDraft := convertUpload, convertDraft
	t.Cleanup(func() {
		cfg = oldCfg
		convertMode = oldMode
		convertUpload = oldUpload
		convertDraft = oldDraft
	})

	cfg = &config.Config{}
	convertMode = "api"
	convertUpload = false
	convertDraft = false

	if err := validateConvertConfig(); err == nil || !strings.Contains(err.Error(), "MD2WECHAT_API_KEY") {
		t.Fatalf("validateConvertConfig() error = %v", err)
	}
}

func TestValidateConvertConfigRejectsConflictingCoverInputs(t *testing.T) {
	oldCfg, oldMode := cfg, convertMode
	oldUpload, oldDraft := convertUpload, convertDraft
	oldCover, oldCoverMediaID := convertCoverImage, convertCoverMediaID
	t.Cleanup(func() {
		cfg = oldCfg
		convertMode = oldMode
		convertUpload = oldUpload
		convertDraft = oldDraft
		convertCoverImage = oldCover
		convertCoverMediaID = oldCoverMediaID
	})

	cfg = &config.Config{WechatAppID: "appid", WechatSecret: "secret"}
	convertMode = "ai"
	convertUpload = false
	convertDraft = true
	convertCoverImage = "/tmp/cover.jpg"
	convertCoverMediaID = "existing-cover-id"

	err := validateConvertConfig()
	if err == nil || !strings.Contains(err.Error(), "--cover and --cover-media-id are mutually exclusive") {
		t.Fatalf("validateConvertConfig() error = %v", err)
	}
}

func TestValidateConvertConfigRejectsURLLikeCoverMediaID(t *testing.T) {
	oldCfg, oldMode := cfg, convertMode
	oldUpload, oldDraft := convertUpload, convertDraft
	oldCover, oldCoverMediaID := convertCoverImage, convertCoverMediaID
	t.Cleanup(func() {
		cfg = oldCfg
		convertMode = oldMode
		convertUpload = oldUpload
		convertDraft = oldDraft
		convertCoverImage = oldCover
		convertCoverMediaID = oldCoverMediaID
	})

	cfg = &config.Config{WechatAppID: "appid", WechatSecret: "secret"}
	convertMode = "ai"
	convertUpload = false
	convertDraft = true
	convertCoverImage = ""
	convertCoverMediaID = "https://mmbiz.qpic.cn/example"

	err := validateConvertConfig()
	if err == nil || !strings.Contains(err.Error(), "--cover-media-id expects a WeChat media_id, not a URL") {
		t.Fatalf("validateConvertConfig() error = %v", err)
	}
}

func TestCreateWeChatDraftRequiresCoverImage(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldMode, oldTheme, oldAPIKey := convertMode, convertTheme, convertAPIKey
	oldFontSize, oldBackground := convertFontSize, convertBackgroundType
	oldCustomPrompt, oldOutput := convertCustomPrompt, convertOutput
	oldPreview, oldUpload, oldDraft := convertPreview, convertUpload, convertDraft
	oldSaveDraft, oldCover := convertSaveDraft, convertCoverImage
	oldNewConverter := newMarkdownConverter
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		convertMode, convertTheme, convertAPIKey = oldMode, oldTheme, oldAPIKey
		convertFontSize, convertBackgroundType = oldFontSize, oldBackground
		convertCustomPrompt, convertOutput = oldCustomPrompt, oldOutput
		convertPreview, convertUpload, convertDraft = oldPreview, oldUpload, oldDraft
		convertSaveDraft, convertCoverImage = oldSaveDraft, oldCover
		newMarkdownConverter = oldNewConverter
	})

	cfg = &config.Config{WechatAppID: "appid", WechatSecret: "secret", MD2WechatAPIKey: "api-key"}
	log = zap.NewNop()
	convertMode = "api"
	convertTheme = "default"
	convertPreview = false
	convertUpload = false
	convertDraft = true
	convertSaveDraft = ""
	convertCoverImage = ""
	convertFontSize = "medium"
	convertBackgroundType = "default"

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

	err := runConvert(nil, []string{markdownPath})
	if err == nil {
		t.Fatal("expected error for missing cover image")
	}

	cliErr, ok := err.(*cliError)
	if !ok {
		t.Fatalf("error type = %T, want *cliError", err)
	}
	if cliErr.Code != codeConvertDraftFailed {
		t.Fatalf("error code = %q", cliErr.Code)
	}
	if !strings.Contains(cliErr.Error(), "--cover") {
		t.Fatalf("draft error = %v", cliErr)
	}
}

func TestRunCreateImagePostValidatesRequiredInputs(t *testing.T) {
	oldTitle, oldImages := imagePostTitle, imagePostImages
	oldFromMD, oldDryRun := imagePostFromMD, imagePostDryRun
	oldContent, oldOutput := imagePostContent, imagePostOutput
	t.Cleanup(func() {
		imagePostTitle, imagePostImages = oldTitle, oldImages
		imagePostFromMD, imagePostDryRun = oldFromMD, oldDryRun
		imagePostContent, imagePostOutput = oldContent, oldOutput
	})

	imagePostTitle = ""
	imagePostImages = ""
	imagePostFromMD = ""
	imagePostContent = ""
	imagePostOutput = filepath.Join(t.TempDir(), "unused.json")
	imagePostDryRun = true

	if _, err := runCreateImagePost(); err == nil || !strings.Contains(err.Error(), "--title is required") {
		t.Fatalf("runCreateImagePost() error = %v", err)
	} else if cliErr, ok := err.(*cliError); !ok || cliErr.Code != codeImagePostInvalid {
		t.Fatalf("runCreateImagePost() error code = %#v", err)
	}

	imagePostTitle = "Title"
	if _, err := runCreateImagePost(); err == nil || !strings.Contains(err.Error(), "--images or --from-markdown is required") {
		t.Fatalf("runCreateImagePost() error = %v", err)
	} else if cliErr, ok := err.(*cliError); !ok || cliErr.Code != codeImagePostInvalid {
		t.Fatalf("runCreateImagePost() error code = %#v", err)
	}
}

func TestRunTestDraftReturnsReadErrorForMissingFile(t *testing.T) {
	oldCfg, oldLog := cfg, log
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
	})

	cfg = &config.Config{WechatAppID: "appid", WechatSecret: "secret"}
	log = zap.NewNop()

	if _, err := runTestDraft("/nonexistent/file.html", "/tmp/cover.jpg"); err == nil || !strings.Contains(err.Error(), "read HTML file") {
		t.Fatalf("runTestDraft() error = %v", err)
	} else if cliErr, ok := err.(*cliError); !ok || cliErr.Code != codeTestDraftReadFailed {
		t.Fatalf("runTestDraft() error code = %#v", err)
	}
}

func TestDraftErrorFormattingIncludesHint(t *testing.T) {
	err := (&publish.DraftError{
		Message: "需要封面",
		Hint:    "使用 --cover",
	}).Error()
	if !strings.Contains(err, "需要封面") || !strings.Contains(err, "--cover") {
		t.Fatalf("DraftError.Error() = %q", err)
	}
}

func TestMaskMediaIDAndHandleAIResultInvalid(t *testing.T) {
	if got := maskMediaID("12345678"); got != "1234***5678" {
		t.Fatalf("maskMediaID() = %q", got)
	}
	if err := handleAIResult(&converter.ConvertResult{}, "article.md"); err == nil {
		t.Fatal("expected invalid AI request error")
	}
}

func TestOutputHTMLWritesFileAndResponseSuccessPrintsJSON(t *testing.T) {
	oldLog := log
	oldStdout := os.Stdout
	t.Cleanup(func() {
		log = oldLog
		os.Stdout = oldStdout
	})

	log = zap.NewNop()
	outputPath := filepath.Join(t.TempDir(), "out.html")
	if err := outputHTML("<p>body</p>", outputPath, false); err != nil {
		t.Fatalf("outputHTML() error = %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output html: %v", err)
	}
	if string(data) != "<p>body</p>" {
		t.Fatalf("output html = %q", string(data))
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	responseSuccess(map[string]any{"ok": true})
	if err := w.Close(); err != nil {
		t.Fatalf("close stdout pipe writer: %v", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read stdout: %v", err)
	}

	var response map[string]any
	if err := json.Unmarshal(buf.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response json: %v", err)
	}
	if response["success"] != true || response["code"] != "OK" || response["message"] != "Success" {
		t.Fatalf("response = %#v", response)
	}
	if response["schema_version"] != "v1" || response["status"] != "completed" || response["retryable"] != false {
		t.Fatalf("response envelope = %#v", response)
	}
	payload, _ := response["data"].(map[string]any)
	if payload["ok"] != true {
		t.Fatalf("response data = %#v", payload)
	}
}

func TestOutputHTMLPreviewWritesPureHTMLToStdout(t *testing.T) {
	oldLog := log
	oldStdout := os.Stdout
	t.Cleanup(func() {
		log = oldLog
		os.Stdout = oldStdout
	})

	log = zap.NewNop()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	if err := outputHTML("<p>body</p>", "", true); err != nil {
		t.Fatalf("outputHTML() error = %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("close stdout pipe writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if got := buf.String(); got != "<p>body</p>" {
		t.Fatalf("stdout = %q", got)
	}
}
