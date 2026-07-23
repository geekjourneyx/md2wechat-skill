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
	result       *converter.ConvertResult
	images       []converter.ImageRef
	reqs         []*converter.ConvertRequest
	calls        int
	extractCalls int
}

func (f *fakeMarkdownConverter) Convert(req *converter.ConvertRequest) *converter.ConvertResult {
	f.calls++
	f.reqs = append(f.reqs, req)
	return f.result
}

func (f *fakeMarkdownConverter) ExtractImages(string) []converter.ImageRef {
	f.extractCalls++
	return f.images
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

func (f *fakeAssetProcessor) totalCalls() int {
	return len(f.localCalls) + len(f.onlineCalls) + len(f.generateCalls)
}

type fakeDraftCreator struct {
	artifacts []Artifact
	result    *DraftResult
	err       error
	calls     int
}

func (f *fakeDraftCreator) CreateDraft(artifact Artifact) (*DraftResult, error) {
	f.calls++
	f.artifacts = append(f.artifacts, artifact)
	if f.err != nil {
		return nil, f.err
	}
	if f.result != nil {
		return f.result, nil
	}
	return &DraftResult{MediaID: "draft-id"}, nil
}

type fakeCoverUploader struct {
	calls int
}

func (f *fakeCoverUploader) upload(string) (string, error) {
	f.calls++
	return "cover-id", nil
}

func TestServiceConvertRejectsDraftBlockersBeforeEffects(t *testing.T) {
	dir := t.TempDir()
	validCover := filepath.Join(dir, "cover.jpg")
	if err := os.WriteFile(validCover, []byte("cover"), 0600); err != nil {
		t.Fatalf("write valid cover: %v", err)
	}
	unsupportedCover := filepath.Join(dir, "cover.txt")
	if err := os.WriteFile(unsupportedCover, []byte("cover"), 0600); err != nil {
		t.Fatalf("write unsupported cover: %v", err)
	}
	coverDir := filepath.Join(dir, "cover-dir.jpg")
	if err := os.Mkdir(coverDir, 0700); err != nil {
		t.Fatalf("create cover directory: %v", err)
	}

	tests := []struct {
		name    string
		cover   string
		coverID string
	}{
		{name: "missing both cover inputs"},
		{name: "conflicting local cover and media ID", cover: validCover, coverID: "media-id"},
		{name: "URL passed as media ID", coverID: "https://example.com/cover.jpg"},
		{name: "missing local cover file", cover: filepath.Join(dir, "missing.jpg")},
		{name: "local cover is a directory", cover: coverDir},
		{name: "unsupported local cover extension", cover: unsupportedCover},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converterFake := &fakeMarkdownConverter{result: &converter.ConvertResult{Success: true}}
			processor := &fakeAssetProcessor{}
			cover := &fakeCoverUploader{}
			drafts := &fakeDraftCreator{}
			svc := NewService(zap.NewNop(), converterFake, processor, drafts, cover.upload)

			_, err := svc.Convert(&ConvertInput{
				Intent:         PublishIntent{CreateDraft: true},
				ConvertRequest: &converter.ConvertRequest{},
				CoverImagePath: tt.cover,
				CoverMediaID:   tt.coverID,
			})
			if err == nil {
				t.Fatal("Convert() error = nil, want draft intent rejection")
			}
			var draftErr *DraftError
			if !errors.As(err, &draftErr) {
				t.Fatalf("Convert() error type = %T, want *DraftError", err)
			}
			if converterFake.calls != 0 || processor.totalCalls() != 0 || cover.calls != 0 || drafts.calls != 0 {
				t.Fatalf("known blocker caused effects: converter=%d assets=%d cover=%d draft=%d", converterFake.calls, processor.totalCalls(), cover.calls, drafts.calls)
			}
		})
	}
}

func TestServiceConvertRejectsCoverWithoutDraftBeforeEffects(t *testing.T) {
	converterFake := &fakeMarkdownConverter{result: &converter.ConvertResult{Success: true}}
	processor := &fakeAssetProcessor{}
	cover := &fakeCoverUploader{}
	drafts := &fakeDraftCreator{}
	svc := NewService(zap.NewNop(), converterFake, processor, drafts, cover.upload)

	_, err := svc.Convert(&ConvertInput{
		ConvertRequest: &converter.ConvertRequest{},
		CoverMediaID:   "media-id",
	})
	var draftErr *DraftError
	if !errors.As(err, &draftErr) {
		t.Fatalf("Convert() error = %T (%v), want *DraftError", err, err)
	}
	if converterFake.calls != 0 || processor.totalCalls() != 0 || cover.calls != 0 || drafts.calls != 0 {
		t.Fatalf("known blocker caused effects: converter=%d assets=%d cover=%d draft=%d", converterFake.calls, processor.totalCalls(), cover.calls, drafts.calls)
	}
}

func TestServiceConvertRejectsInvalidLocalAssetsBeforeEffects(t *testing.T) {
	dir := t.TempDir()
	unsupported := filepath.Join(dir, "image.txt")
	if err := os.WriteFile(unsupported, []byte("not an image"), 0600); err != nil {
		t.Fatalf("write unsupported image: %v", err)
	}
	directory := filepath.Join(dir, "image.png")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatalf("create image directory: %v", err)
	}

	tests := []struct {
		name   string
		source string
	}{
		{name: "missing", source: "missing.png"},
		{name: "unsupported", source: filepath.Base(unsupported)},
		{name: "directory", source: filepath.Base(directory)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageRef := converter.ImageRef{Index: 0, Type: converter.ImageTypeLocal, Original: tt.source}
			converterFake := &fakeMarkdownConverter{
				images: []converter.ImageRef{imageRef},
			}
			processor := &fakeAssetProcessor{}
			cover := &fakeCoverUploader{}
			drafts := &fakeDraftCreator{}
			svc := NewService(zap.NewNop(), converterFake, processor, drafts, cover.upload)

			_, err := svc.Convert(&ConvertInput{
				Intent: PublishIntent{Upload: true},
				ConvertRequest: &converter.ConvertRequest{
					Markdown: "![image](" + tt.source + ")",
				},
				MarkdownDir: dir,
			})
			if !IsAssetError(err) {
				t.Fatalf("Convert() error = %T (%v), want *AssetError", err, err)
			}
			if converterFake.calls != 0 || processor.totalCalls() != 0 || cover.calls != 0 || drafts.calls != 0 {
				t.Fatalf("invalid local asset caused effects: converter=%d assets=%d cover=%d draft=%d", converterFake.calls, processor.totalCalls(), cover.calls, drafts.calls)
			}
		})
	}
}

func TestServiceConvertRejectsInvalidLocalSinksBeforeEffects(t *testing.T) {
	dir := t.TempDir()
	bodyPath := filepath.Join(dir, "body.png")
	if err := os.WriteFile(bodyPath, []byte("body"), 0600); err != nil {
		t.Fatalf("write body: %v", err)
	}
	coverPath := filepath.Join(dir, "cover.jpg")
	if err := os.WriteFile(coverPath, []byte("cover"), 0600); err != nil {
		t.Fatalf("write cover: %v", err)
	}

	tests := []struct {
		name          string
		outputFile    string
		saveDraftPath string
		wantError     string
	}{
		{
			name:       "output parent missing",
			outputFile: filepath.Join(dir, "missing-output", "article.html"),
			wantError:  "prepare output file",
		},
		{
			name:          "saved draft parent missing",
			saveDraftPath: filepath.Join(dir, "missing-draft", "draft.json"),
			wantError:     "prepare draft file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageRef := converter.ImageRef{
				Index:    0,
				Type:     converter.ImageTypeLocal,
				Original: "body.png",
			}
			converterFake := &fakeMarkdownConverter{
				result: &converter.ConvertResult{
					Mode:    converter.ModeAPI,
					Success: true,
					HTML:    "<p>body</p>",
					Images:  []converter.ImageRef{imageRef},
				},
				images: []converter.ImageRef{imageRef},
			}
			processor := &fakeAssetProcessor{
				localResults: map[string]*image.UploadResult{
					bodyPath: {MediaID: "body-id", WechatURL: "https://wechat.local/body"},
				},
			}
			cover := &fakeCoverUploader{}
			drafts := &fakeDraftCreator{}
			svc := NewService(zap.NewNop(), converterFake, processor, drafts, cover.upload)

			_, err := svc.Convert(&ConvertInput{
				Intent: PublishIntent{
					Upload:      true,
					CreateDraft: true,
					SaveDraft:   tt.saveDraftPath != "",
				},
				ConvertRequest: &converter.ConvertRequest{
					Mode:     converter.ModeAPI,
					Markdown: "![body](body.png)",
				},
				MarkdownDir:    dir,
				OutputFile:     tt.outputFile,
				SaveDraftPath:  tt.saveDraftPath,
				CoverImagePath: coverPath,
			})
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("Convert() error = %v, want %q", err, tt.wantError)
			}
			if converterFake.calls != 0 ||
				converterFake.extractCalls != 0 ||
				processor.totalCalls() != 0 ||
				cover.calls != 0 ||
				drafts.calls != 0 {
				t.Fatalf(
					"effects: convert=%d extract=%d assets=%d cover=%d draft=%d",
					converterFake.calls,
					converterFake.extractCalls,
					processor.totalCalls(),
					cover.calls,
					drafts.calls,
				)
			}
		})
	}
}

func TestServiceConvertRejectsBrokenDraftSymlinkBeforeEffects(t *testing.T) {
	dir := t.TempDir()
	bodyPath := filepath.Join(dir, "body.png")
	if err := os.WriteFile(bodyPath, []byte("body"), 0600); err != nil {
		t.Fatalf("write body: %v", err)
	}
	coverPath := filepath.Join(dir, "cover.jpg")
	if err := os.WriteFile(coverPath, []byte("cover"), 0600); err != nil {
		t.Fatalf("write cover: %v", err)
	}
	saveDraftPath := filepath.Join(dir, "draft.json")
	brokenTarget := filepath.Join(dir, "missing-target", "draft.json")
	if err := os.Symlink(brokenTarget, saveDraftPath); err != nil {
		t.Fatalf("create broken draft symlink: %v", err)
	}

	imageRef := converter.ImageRef{
		Index:    0,
		Type:     converter.ImageTypeLocal,
		Original: "body.png",
	}
	converterFake := &fakeMarkdownConverter{
		result: &converter.ConvertResult{
			Mode:    converter.ModeAPI,
			Success: true,
			HTML:    "<p>body</p>",
			Images:  []converter.ImageRef{imageRef},
		},
		images: []converter.ImageRef{imageRef},
	}
	processor := &fakeAssetProcessor{
		localResults: map[string]*image.UploadResult{
			bodyPath: {MediaID: "body-id", WechatURL: "https://wechat.local/body"},
		},
	}
	cover := &fakeCoverUploader{}
	drafts := &fakeDraftCreator{}
	svc := NewService(zap.NewNop(), converterFake, processor, drafts, cover.upload)

	_, err := svc.Convert(&ConvertInput{
		Intent: PublishIntent{
			Upload:      true,
			CreateDraft: true,
			SaveDraft:   true,
		},
		ConvertRequest: &converter.ConvertRequest{
			Mode:     converter.ModeAPI,
			Markdown: "![body](body.png)",
		},
		MarkdownDir:    dir,
		SaveDraftPath:  saveDraftPath,
		CoverImagePath: coverPath,
	})
	if err == nil || !strings.Contains(err.Error(), "prepare draft file") {
		t.Errorf("Convert() error = %v, want prepare draft file", err)
	}
	if converterFake.calls != 0 ||
		converterFake.extractCalls != 0 ||
		processor.totalCalls() != 0 ||
		cover.calls != 0 ||
		drafts.calls != 0 {
		t.Fatalf(
			"effects: convert=%d extract=%d assets=%d cover=%d draft=%d",
			converterFake.calls,
			converterFake.extractCalls,
			processor.totalCalls(),
			cover.calls,
			drafts.calls,
		)
	}
}

func TestServiceConvertRejectsMissingDraftDependenciesBeforeEffects(t *testing.T) {
	dir := t.TempDir()
	coverPath := filepath.Join(dir, "cover.jpg")
	if err := os.WriteFile(coverPath, []byte("cover"), 0600); err != nil {
		t.Fatalf("write cover: %v", err)
	}

	tests := []struct {
		name        string
		drafts      DraftCreator
		uploadCover CoverUploader
		wantError   string
	}{
		{
			name:        "missing draft creator",
			uploadCover: (&fakeCoverUploader{}).upload,
			wantError:   "draft creator is required",
		},
		{
			name:      "missing local cover uploader",
			drafts:    &fakeDraftCreator{},
			wantError: "cover uploader is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converterFake := &fakeMarkdownConverter{result: &converter.ConvertResult{Success: true}}
			processor := &fakeAssetProcessor{}
			cover := &fakeCoverUploader{}
			uploadCover := tt.uploadCover
			if uploadCover != nil {
				uploadCover = cover.upload
			}
			draftPath := filepath.Join(dir, strings.ReplaceAll(tt.name, " ", "-")+".json")
			svc := NewService(zap.NewNop(), converterFake, processor, tt.drafts, uploadCover)

			_, err := svc.Convert(&ConvertInput{
				Intent:         PublishIntent{CreateDraft: true, SaveDraft: true},
				ConvertRequest: &converter.ConvertRequest{},
				SaveDraftPath:  draftPath,
				CoverImagePath: coverPath,
			})
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("Convert() error = %v, want %q", err, tt.wantError)
			}
			if converterFake.calls != 0 || processor.totalCalls() != 0 || cover.calls != 0 {
				t.Fatalf("missing dependency caused effects: converter=%d assets=%d cover=%d", converterFake.calls, processor.totalCalls(), cover.calls)
			}
			if draftFake, ok := tt.drafts.(*fakeDraftCreator); ok && draftFake.calls != 0 {
				t.Fatalf("draft creator calls = %d", draftFake.calls)
			}
			if _, statErr := os.Stat(draftPath); !os.IsNotExist(statErr) {
				t.Fatalf("local draft output must not exist, stat error = %v", statErr)
			}
		})
	}
}

func TestServiceConvertReturnsAIRequestWithoutRunningSideEffects(t *testing.T) {
	dir := t.TempDir()
	coverPath := filepath.Join(dir, "cover.jpg")
	if err := os.WriteFile(coverPath, []byte("cover"), 0600); err != nil {
		t.Fatalf("write cover: %v", err)
	}

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
		CoverImagePath: coverPath,
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

func TestConvertOutputFailurePreventsDraft(t *testing.T) {
	dir := t.TempDir()
	coverPath := filepath.Join(dir, "cover.jpg")
	if err := os.WriteFile(coverPath, []byte("cover"), 0600); err != nil {
		t.Fatalf("write cover: %v", err)
	}

	converterFake := &fakeMarkdownConverter{result: &converter.ConvertResult{
		Mode:    converter.ModeAPI,
		Success: true,
		HTML:    "<p>body</p>",
	}}
	processor := &fakeAssetProcessor{}
	cover := &fakeCoverUploader{}
	drafts := &fakeDraftCreator{}
	svc := NewService(zap.NewNop(), converterFake, processor, drafts, cover.upload)

	_, err := svc.Convert(&ConvertInput{
		Intent:         PublishIntent{CreateDraft: true},
		ConvertRequest: &converter.ConvertRequest{Mode: converter.ModeAPI},
		OutputFile:     filepath.Join(dir, "missing", "article.html"),
		CoverImagePath: coverPath,
	})
	if err == nil || !strings.Contains(err.Error(), "prepare output file") {
		t.Fatalf("Convert() error = %v, want output write failure", err)
	}
	if processor.totalCalls() != 0 || cover.calls != 0 || drafts.calls != 0 {
		t.Fatalf(
			"output failure caused publish effects: assets=%d cover=%d draft=%d",
			processor.totalCalls(),
			cover.calls,
			drafts.calls,
		)
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
	if err := os.WriteFile(filepath.Join(dir, "cover.jpg"), []byte("cover"), 0644); err != nil {
		t.Fatalf("write cover: %v", err)
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
	conversionImages := []converter.ImageRef{
		{Index: 0, Type: converter.ImageTypeLocal, Original: filepath.Join("images", "local.png"), Placeholder: "<!-- IMG:0 -->"},
		{Index: 1, Type: converter.ImageTypeOnline, Original: "https://example.com/r.png", Placeholder: "<!-- IMG:1 -->"},
		{Index: 2, Type: converter.ImageTypeAI, Original: "draw fox", AIPrompt: "draw fox", Placeholder: "<!-- IMG:2 -->"},
	}
	svc := NewService(
		zap.NewNop(),
		&fakeMarkdownConverter{
			images: conversionImages,
			result: &converter.ConvertResult{
				Mode:    converter.ModeAPI,
				Theme:   "default",
				Success: true,
				Status:  action.StatusCompleted,
				Action:  action.ActionConvert,
				HTML:    `<img src="https://cdn.example.com/1"><img src="https://cdn.example.com/2"><img src="https://cdn.example.com/3">`,
				Images:  conversionImages,
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
	outputPath := filepath.Join(dir, "out.html")
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
			Mode:     converter.ModeAPI,
			Theme:    "default",
			APIKey:   "api-key",
		},
		MarkdownDir:    dir,
		OutputFile:     outputPath,
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
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != drafter.artifacts[0].HTML {
		t.Fatalf("output HTML differs from draft artifact")
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

func TestServiceConvertUsesExistingCoverMediaIDWithoutUploadingCover(t *testing.T) {
	svc := NewService(
		zap.NewNop(),
		&fakeMarkdownConverter{
			result: &converter.ConvertResult{
				Mode:    converter.ModeAPI,
				Theme:   "default",
				Success: true,
				Status:  action.StatusCompleted,
				Action:  action.ActionConvert,
				HTML:    "<p>body</p>",
			},
		},
		&fakeAssetProcessor{},
		&fakeDraftCreator{result: &DraftResult{MediaID: "draft-2"}},
		nil,
	)

	output, err := svc.Convert(&ConvertInput{
		Source: ArticleSource{
			Path:     "article.md",
			Markdown: "# 标题\n",
			Metadata: Metadata{Title: "标题"},
		},
		Intent: PublishIntent{
			Mode:        "api",
			CreateDraft: true,
		},
		ConvertRequest: &converter.ConvertRequest{
			Markdown: "# 标题\n",
			Mode:     converter.ModeAPI,
			Theme:    "default",
			APIKey:   "api-key",
		},
		CoverMediaID: "existing-cover-id",
	})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if output.Artifact.CoverMediaID != "existing-cover-id" {
		t.Fatalf("cover media id = %q", output.Artifact.CoverMediaID)
	}
	if output.Artifact.DraftMediaID != "draft-2" {
		t.Fatalf("draft media id = %q", output.Artifact.DraftMediaID)
	}
}

func TestServiceConvertReturnsTypedStageErrors(t *testing.T) {
	svc := NewService(
		zap.NewNop(),
		&fakeMarkdownConverter{
			result: &converter.ConvertResult{
				Mode:    converter.ModeAPI,
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
			Mode:     converter.ModeAPI,
			Theme:    "default",
			APIKey:   "api-key",
		},
	})
	if !IsAssetError(err) {
		t.Fatalf("expected AssetError, got %T (%v)", err, err)
	}
}
