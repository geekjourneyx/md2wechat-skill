package publish

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/action"
	"github.com/geekjourneyx/md2wechat-skill/internal/converter"
	"github.com/geekjourneyx/md2wechat-skill/internal/image"
	"go.uber.org/zap"
)

type fakeMarkdownConverter struct {
	result *converter.ConvertResult
	reqs   []*converter.ConvertRequest
}

func (f *fakeMarkdownConverter) Convert(req *converter.ConvertRequest) *converter.ConvertResult {
	f.reqs = append(f.reqs, req)
	return f.result
}

type fakeAssetProcessor struct {
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

func (f *fakeAssetProcessor) UploadLocalImage(filePath string) (*image.UploadResult, error) {
	f.localCalls = append(f.localCalls, filePath)
	if err := f.localErrs[filePath]; err != nil {
		return nil, err
	}
	return f.localResults[filePath], nil
}

func (f *fakeAssetProcessor) DownloadAndUpload(url string) (*image.UploadResult, error) {
	f.onlineCalls = append(f.onlineCalls, url)
	if err := f.onlineErrs[url]; err != nil {
		return nil, err
	}
	return f.onlineResults[url], nil
}

func (f *fakeAssetProcessor) GenerateAndUpload(prompt string) (*image.GenerateAndUploadResult, error) {
	f.generateCalls = append(f.generateCalls, prompt)
	if err := f.generateErrs[prompt]; err != nil {
		return nil, err
	}
	return f.generateResults[prompt], nil
}

type fakeDraftCreator struct {
	artifacts []Artifact
	result    *DraftResult
	err       error
}

func (f *fakeDraftCreator) CreateDraft(artifact Artifact) (*DraftResult, error) {
	f.artifacts = append(f.artifacts, artifact)
	if f.err != nil {
		return nil, f.err
	}
	if f.result != nil {
		return f.result, nil
	}
	return &DraftResult{MediaID: "draft-id"}, nil
}

func TestServiceConvertReturnsAIRequestWithoutRunningSideEffects(t *testing.T) {
	svc := NewService(
		zap.NewNop(),
		&fakeMarkdownConverter{
			result: &converter.ConvertResult{
				Mode:    converter.ModeAI,
				Status:  action.StatusActionRequired,
				Action:  action.ActionConvert,
				Prompt:  "prompt body",
				Success: true,
				Images: []converter.ImageRef{
					{Index: 0, Type: converter.ImageTypeLocal, Original: "images/a.png", Placeholder: "<!-- IMG:0 -->"},
				},
			},
		},
		&fakeAssetProcessor{},
		&fakeDraftCreator{},
		func(path string) (string, error) { return "unused", nil },
	)

	output, err := svc.Convert(&ConvertInput{
		Source: ArticleSource{
			Path:     "article.md",
			Markdown: "![x](images/a.png)",
			Metadata: Metadata{Title: "标题"},
		},
		Intent: PublishIntent{
			Mode:        "ai",
			Upload:      true,
			CreateDraft: true,
		},
		ConvertRequest: &converter.ConvertRequest{
			Markdown: "![x](images/a.png)",
			Mode:     converter.ModeAI,
			Theme:    "autumn-warm",
		},
		MarkdownDir:    "/tmp/work",
		CoverImagePath: "/tmp/cover.jpg",
	})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if output == nil || output.Conversion == nil {
		t.Fatal("expected conversion output")
	}
	if output.Conversion.Status != action.StatusActionRequired {
		t.Fatalf("status = %q, want %q", output.Conversion.Status, action.StatusActionRequired)
	}
	if output.Artifact.HTML != "" {
		t.Fatalf("expected no HTML for AI request, got %q", output.Artifact.HTML)
	}
	if len(output.Artifact.Assets) != 1 {
		t.Fatalf("asset count = %d, want 1", len(output.Artifact.Assets))
	}
	if output.Artifact.Assets[0].ResolvedSource != filepath.Join("/tmp/work", "images/a.png") {
		t.Fatalf("resolved source = %q", output.Artifact.Assets[0].ResolvedSource)
	}
}

func TestServiceConvertProcessesAssetsAndCreatesDraft(t *testing.T) {
	dir := t.TempDir()
	localPath := filepath.Join(dir, "images", "local.png")
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(localPath, []byte("x"), 0644); err != nil {
		t.Fatalf("write local image: %v", err)
	}

	assets := &fakeAssetProcessor{
		localResults: map[string]*image.UploadResult{
			localPath: {MediaID: "m-local", WechatURL: "https://wechat.local/local"},
		},
		onlineResults: map[string]*image.UploadResult{
			"https://example.com/r.png": {MediaID: "m-remote", WechatURL: "https://wechat.local/remote"},
		},
		generateResults: map[string]*image.GenerateAndUploadResult{
			"draw fox": {MediaID: "m-ai", WechatURL: "https://wechat.local/ai"},
		},
	}
	drafter := &fakeDraftCreator{result: &DraftResult{MediaID: "draft-1"}}
	svc := NewService(
		zap.NewNop(),
		&fakeMarkdownConverter{
			result: &converter.ConvertResult{
				Mode:    converter.ModeAI,
				Theme:   "default",
				Success: true,
				Status:  action.StatusCompleted,
				Action:  action.ActionConvert,
				HTML:    `<img src="https://cdn.example.com/1"><img src="https://cdn.example.com/2"><img src="https://cdn.example.com/3">`,
				Images: []converter.ImageRef{
					{Index: 0, Type: converter.ImageTypeLocal, Original: filepath.Join("images", "local.png"), Placeholder: "<!-- IMG:0 -->"},
					{Index: 1, Type: converter.ImageTypeOnline, Original: "https://example.com/r.png", Placeholder: "<!-- IMG:1 -->"},
					{Index: 2, Type: converter.ImageTypeAI, Original: "draw fox", AIPrompt: "draw fox", Placeholder: "<!-- IMG:2 -->"},
				},
			},
		},
		assets,
		drafter,
		func(path string) (string, error) {
			if path != filepath.Join(dir, "cover.jpg") {
				t.Fatalf("cover path = %q", path)
			}
			return "cover-id", nil
		},
	)

	draftPath := filepath.Join(dir, "draft.json")
	output, err := svc.Convert(&ConvertInput{
		Source: ArticleSource{
			Path:     filepath.Join(dir, "article.md"),
			Markdown: "body",
			Metadata: Metadata{
				Title:  "文章标题",
				Author: "作者",
				Digest: "摘要",
			},
		},
		Intent: PublishIntent{
			Mode:        "api",
			Upload:      true,
			CreateDraft: true,
			SaveDraft:   true,
		},
		ConvertRequest: &converter.ConvertRequest{
			Markdown: "body",
			Mode:     converter.ModeAI,
			Theme:    "default",
		},
		MarkdownDir:    dir,
		OutputFile:     filepath.Join(dir, "out.html"),
		SaveDraftPath:  draftPath,
		CoverImagePath: filepath.Join(dir, "cover.jpg"),
	})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	if len(assets.localCalls) != 1 || assets.localCalls[0] != localPath {
		t.Fatalf("local calls = %#v", assets.localCalls)
	}
	if len(assets.onlineCalls) != 1 || assets.onlineCalls[0] != "https://example.com/r.png" {
		t.Fatalf("online calls = %#v", assets.onlineCalls)
	}
	if len(assets.generateCalls) != 1 || assets.generateCalls[0] != "draw fox" {
		t.Fatalf("generate calls = %#v", assets.generateCalls)
	}
	if output.Artifact.CoverMediaID != "cover-id" || output.Artifact.DraftMediaID != "draft-1" {
		t.Fatalf("artifact ids = %#v", output.Artifact)
	}
	for _, expected := range []string{
		"https://wechat.local/local",
		"https://wechat.local/remote",
		"https://wechat.local/ai",
	} {
		if !strings.Contains(output.Artifact.HTML, expected) {
			t.Fatalf("artifact html missing %q: %s", expected, output.Artifact.HTML)
		}
	}
	if len(drafter.artifacts) != 1 {
		t.Fatalf("draft artifacts = %#v", drafter.artifacts)
	}
	artifact := drafter.artifacts[0]
	if artifact.Metadata.Title != "文章标题" || artifact.Metadata.Author != "作者" || artifact.Metadata.Digest != "摘要" {
		t.Fatalf("draft artifact = %#v", artifact)
	}
	if artifact.CoverMediaID != "cover-id" {
		t.Fatalf("draft cover fields = %#v", artifact)
	}
	if _, err := os.Stat(draftPath); err != nil {
		t.Fatalf("draft file not written: %v", err)
	}
}

func TestServiceConvertReturnsTypedStageErrors(t *testing.T) {
	svc := NewService(
		zap.NewNop(),
		&fakeMarkdownConverter{
			result: &converter.ConvertResult{
				Mode:    converter.ModeAI,
				Theme:   "default",
				Success: true,
				Status:  action.StatusCompleted,
				Action:  action.ActionConvert,
				HTML:    `<img src="https://cdn.example.com/1">`,
				Images: []converter.ImageRef{
					{Index: 0, Type: converter.ImageTypeOnline, Original: "https://example.com/fail.png", Placeholder: "<!-- IMG:0 -->"},
				},
			},
		},
		&fakeAssetProcessor{
			onlineErrs: map[string]error{
				"https://example.com/fail.png": errors.New("boom"),
			},
		},
		&fakeDraftCreator{},
		func(path string) (string, error) { return "cover", nil },
	)

	_, err := svc.Convert(&ConvertInput{
		Source: ArticleSource{Markdown: "body"},
		Intent: PublishIntent{Upload: true},
		ConvertRequest: &converter.ConvertRequest{
			Markdown: "body",
			Mode:     converter.ModeAI,
			Theme:    "default",
		},
	})
	if !IsAssetError(err) {
		t.Fatalf("expected AssetError, got %T (%v)", err, err)
	}
}
