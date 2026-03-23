package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"github.com/geekjourneyx/md2wechat-skill/internal/publish"
	"go.uber.org/zap"
)

type fakeImagePostService struct {
	previewReqs []*publish.ImagePostInput
	createReqs  []*publish.ImagePostInput

	previewResult *publish.ImagePostPreview
	createResult  *publish.ImagePostResult
	previewErr    error
	createErr     error
}

func (f *fakeImagePostService) PreviewImagePost(req *publish.ImagePostInput) (*publish.ImagePostPreview, error) {
	cloned := cloneImagePostRequest(req)
	f.previewReqs = append(f.previewReqs, cloned)
	if f.previewErr != nil {
		return nil, f.previewErr
	}
	return f.previewResult, nil
}

func (f *fakeImagePostService) CreateImagePost(req *publish.ImagePostInput) (*publish.ImagePostResult, error) {
	cloned := cloneImagePostRequest(req)
	f.createReqs = append(f.createReqs, cloned)
	if f.createErr != nil {
		return nil, f.createErr
	}
	return f.createResult, nil
}

func cloneImagePostRequest(req *publish.ImagePostInput) *publish.ImagePostInput {
	cloned := *req
	cloned.Images = append([]string(nil), req.Images...)
	return &cloned
}

func TestRunCreateImagePostDryRunBuildsPreviewRequestAndWritesOutput(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldTitle, oldContent := imagePostTitle, imagePostContent
	oldImages, oldFromMD := imagePostImages, imagePostFromMD
	oldOpenComment, oldFansOnly := imagePostOpenComment, imagePostFansOnly
	oldDryRun, oldOutput := imagePostDryRun, imagePostOutput
	oldServiceFactory, oldIsTerminalFn := newImagePostService, isTerminalFn
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		imagePostTitle, imagePostContent = oldTitle, oldContent
		imagePostImages, imagePostFromMD = oldImages, oldFromMD
		imagePostOpenComment, imagePostFansOnly = oldOpenComment, oldFansOnly
		imagePostDryRun, imagePostOutput = oldDryRun, oldOutput
		newImagePostService, isTerminalFn = oldServiceFactory, oldIsTerminalFn
	})

	cfg = &config.Config{}
	log = zap.NewNop()

	outputPath := filepath.Join(t.TempDir(), "preview.json")
	imagePostTitle = "Weekend Trip"
	imagePostContent = "notes"
	imagePostImages = "a.jpg, b.jpg"
	imagePostFromMD = "/tmp/article.md"
	imagePostOpenComment = true
	imagePostFansOnly = true
	imagePostDryRun = true
	imagePostOutput = outputPath
	isTerminalFn = func() bool { return true }

	service := &fakeImagePostService{
		previewResult: &publish.ImagePostPreview{
			Title:      "Weekend Trip",
			ImageCount: 3,
		},
	}
	newImagePostService = func() imagePostService { return service }

	response, err := runCreateImagePost()
	if err != nil {
		t.Fatalf("runCreateImagePost() error = %v", err)
	}

	gotResponse, ok := response.(map[string]any)
	if !ok {
		t.Fatalf("response type = %T, want map[string]any", response)
	}
	if gotResponse["mode"] != "dry-run" {
		t.Fatalf("response mode = %#v", gotResponse["mode"])
	}
	if len(service.previewReqs) != 1 {
		t.Fatalf("preview reqs = %#v", service.previewReqs)
	}

	req := service.previewReqs[0]
	if req.Title != "Weekend Trip" || req.Content != "notes" || req.FromMarkdown != "/tmp/article.md" {
		t.Fatalf("unexpected preview request: %#v", req)
	}
	if !reflect.DeepEqual(req.Images, []string{"a.jpg", "b.jpg"}) {
		t.Fatalf("request images = %#v", req.Images)
	}
	if !req.OpenComment || !req.FansOnly {
		t.Fatalf("comment flags not preserved: %#v", req)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read preview output: %v", err)
	}
	var saved publish.ImagePostPreview
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("unmarshal preview output: %v", err)
	}
	if saved.Title != "Weekend Trip" {
		t.Fatalf("saved preview = %#v", saved)
	}
}

func TestCreateImagePostCmdDryRunOutputsStableEnvelope(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldTitle, oldContent := imagePostTitle, imagePostContent
	oldImages, oldFromMD := imagePostImages, imagePostFromMD
	oldOpenComment, oldFansOnly := imagePostOpenComment, imagePostFansOnly
	oldDryRun, oldOutput := imagePostDryRun, imagePostOutput
	oldServiceFactory, oldIsTerminalFn := newImagePostService, isTerminalFn
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		imagePostTitle, imagePostContent = oldTitle, oldContent
		imagePostImages, imagePostFromMD = oldImages, oldFromMD
		imagePostOpenComment, imagePostFansOnly = oldOpenComment, oldFansOnly
		imagePostDryRun, imagePostOutput = oldDryRun, oldOutput
		newImagePostService, isTerminalFn = oldServiceFactory, oldIsTerminalFn
		createImagePostCmd.SetArgs(nil)
	})

	cfg = &config.Config{}
	log = zap.NewNop()
	isTerminalFn = func() bool { return true }

	service := &fakeImagePostService{
		previewResult: &publish.ImagePostPreview{
			Title:      "Weekend Trip",
			ImageCount: 2,
		},
	}
	newImagePostService = func() imagePostService { return service }

	createImagePostCmd.SetArgs([]string{
		"--title", "Weekend Trip",
		"--images", "a.jpg,b.jpg",
		"--dry-run",
	})

	stdout := captureStdout(t, func() {
		if err := createImagePostCmd.Execute(); err != nil {
			t.Fatalf("createImagePostCmd.Execute() error = %v", err)
		}
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["success"] != true || response["code"] != codeImagePostPreviewReady {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response["schema_version"] != "v1" || response["status"] != "completed" || response["retryable"] != false {
		t.Fatalf("unexpected envelope: %#v", response)
	}
	data, _ := response["data"].(map[string]any)
	if data["mode"] != "dry-run" {
		t.Fatalf("unexpected dry-run data: %#v", data)
	}
}

func TestCreateImagePostCmdCreateOutputsStableEnvelope(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldTitle, oldContent := imagePostTitle, imagePostContent
	oldImages, oldFromMD := imagePostImages, imagePostFromMD
	oldOpenComment, oldFansOnly := imagePostOpenComment, imagePostFansOnly
	oldDryRun, oldOutput := imagePostDryRun, imagePostOutput
	oldServiceFactory, oldIsTerminalFn := newImagePostService, isTerminalFn
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		imagePostTitle, imagePostContent = oldTitle, oldContent
		imagePostImages, imagePostFromMD = oldImages, oldFromMD
		imagePostOpenComment, imagePostFansOnly = oldOpenComment, oldFansOnly
		imagePostDryRun, imagePostOutput = oldDryRun, oldOutput
		newImagePostService, isTerminalFn = oldServiceFactory, oldIsTerminalFn
		createImagePostCmd.SetArgs(nil)
	})

	cfg = &config.Config{WechatAppID: "appid", WechatSecret: "secret"}
	log = zap.NewNop()
	isTerminalFn = func() bool { return true }

	service := &fakeImagePostService{
		createResult: &publish.ImagePostResult{
			MediaID:     "media-123",
			ImageCount:  1,
			UploadedIDs: []string{"img-1"},
		},
	}
	newImagePostService = func() imagePostService { return service }

	createImagePostCmd.SetArgs([]string{
		"--title", "Food Blog",
		"--images", "food.jpg",
	})

	stdout := captureStdout(t, func() {
		if err := createImagePostCmd.Execute(); err != nil {
			t.Fatalf("createImagePostCmd.Execute() error = %v", err)
		}
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["success"] != true || response["code"] != codeImagePostCreated {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response["schema_version"] != "v1" || response["status"] != "completed" || response["retryable"] != false {
		t.Fatalf("unexpected envelope: %#v", response)
	}
	data, _ := response["data"].(map[string]any)
	if data["media_id"] != "media-123" {
		t.Fatalf("unexpected create data: %#v", data)
	}
}

func TestRunCreateImagePostCreateUsesServiceAndWritesResult(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldTitle, oldContent := imagePostTitle, imagePostContent
	oldImages, oldFromMD := imagePostImages, imagePostFromMD
	oldOpenComment, oldFansOnly := imagePostOpenComment, imagePostFansOnly
	oldDryRun, oldOutput := imagePostDryRun, imagePostOutput
	oldServiceFactory, oldIsTerminalFn := newImagePostService, isTerminalFn
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		imagePostTitle, imagePostContent = oldTitle, oldContent
		imagePostImages, imagePostFromMD = oldImages, oldFromMD
		imagePostOpenComment, imagePostFansOnly = oldOpenComment, oldFansOnly
		imagePostDryRun, imagePostOutput = oldDryRun, oldOutput
		newImagePostService, isTerminalFn = oldServiceFactory, oldIsTerminalFn
	})

	cfg = &config.Config{WechatAppID: "appid", WechatSecret: "secret"}
	log = zap.NewNop()

	outputPath := filepath.Join(t.TempDir(), "result.json")
	imagePostTitle = "Food Blog"
	imagePostContent = "today"
	imagePostImages = "food.jpg"
	imagePostFromMD = ""
	imagePostOpenComment = false
	imagePostFansOnly = false
	imagePostDryRun = false
	imagePostOutput = outputPath
	isTerminalFn = func() bool { return true }

	service := &fakeImagePostService{
		createResult: &publish.ImagePostResult{
			MediaID:     "media-123",
			ImageCount:  1,
			UploadedIDs: []string{"img-1"},
		},
	}
	newImagePostService = func() imagePostService { return service }

	response, err := runCreateImagePost()
	if err != nil {
		t.Fatalf("runCreateImagePost() error = %v", err)
	}

	result, ok := response.(*publish.ImagePostResult)
	if !ok {
		t.Fatalf("response type = %T, want *publish.ImagePostResult", response)
	}
	if result.MediaID != "media-123" {
		t.Fatalf("result = %#v", result)
	}
	if len(service.createReqs) != 1 || service.createReqs[0].Title != "Food Blog" {
		t.Fatalf("create reqs = %#v", service.createReqs)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read result output: %v", err)
	}
	var saved publish.ImagePostResult
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("unmarshal result output: %v", err)
	}
	if saved.MediaID != "media-123" {
		t.Fatalf("saved result = %#v", saved)
	}
}

func TestRunTestDraftUsesCoverUploadAndDraftCreator(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldNewDraftCreator, oldUploadCoverImageFn := newDraftCreator, uploadCoverImageFn
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		newDraftCreator, uploadCoverImageFn = oldNewDraftCreator, oldUploadCoverImageFn
	})

	cfg = &config.Config{WechatAppID: "appid", WechatSecret: "secret"}
	log = zap.NewNop()

	htmlFile := filepath.Join(t.TempDir(), "article.html")
	if err := os.WriteFile(htmlFile, []byte("<p>Hello</p>"), 0600); err != nil {
		t.Fatalf("write html: %v", err)
	}

	drafter := &fakeDraftCreator{result: &publish.DraftResult{MediaID: "draft-1", DraftURL: "https://example.com/draft"}}
	newDraftCreator = func() publish.DraftCreator { return drafter }
	uploadCoverImageFn = func(imagePath string) (string, error) {
		if imagePath != "/tmp/cover.jpg" {
			t.Fatalf("cover image path = %q", imagePath)
		}
		return "cover-media-id", nil
	}

	response, err := runTestDraft(htmlFile, "/tmp/cover.jpg")
	if err != nil {
		t.Fatalf("runTestDraft() error = %v", err)
	}
	if response["media_id"] != "draft-1" {
		t.Fatalf("response = %#v", response)
	}
	if response["draft_url"] != "https://example.com/draft" {
		t.Fatalf("response = %#v", response)
	}
	if len(drafter.artifacts) != 1 {
		t.Fatalf("drafter artifacts = %#v", drafter.artifacts)
	}

	artifact := drafter.artifacts[0]
	if artifact.CoverMediaID != "cover-media-id" || artifact.HTML != "<p>Hello</p>" {
		t.Fatalf("draft artifact = %#v", artifact)
	}
}

func TestRunTestDraftStopsOnCoverUploadFailure(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldNewDraftCreator, oldUploadCoverImageFn := newDraftCreator, uploadCoverImageFn
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		newDraftCreator, uploadCoverImageFn = oldNewDraftCreator, oldUploadCoverImageFn
	})

	cfg = &config.Config{WechatAppID: "appid", WechatSecret: "secret"}
	log = zap.NewNop()

	htmlFile := filepath.Join(t.TempDir(), "article.html")
	if err := os.WriteFile(htmlFile, []byte("<p>Hello</p>"), 0600); err != nil {
		t.Fatalf("write html: %v", err)
	}

	drafter := &fakeDraftCreator{}
	newDraftCreator = func() publish.DraftCreator { return drafter }
	uploadCoverImageFn = func(imagePath string) (string, error) {
		return "", fmt.Errorf("cover upload failed")
	}

	errResponse, err := runTestDraft(htmlFile, "/tmp/cover.jpg")
	if err == nil {
		t.Fatalf("expected error, got response %#v", errResponse)
	}
	if cliErr, ok := err.(*cliError); !ok || cliErr.Code != codeTestDraftCoverFailed {
		t.Fatalf("unexpected error code: %#v", err)
	}
	if len(drafter.artifacts) != 0 {
		t.Fatalf("draft creator should not be called: %#v", drafter.artifacts)
	}
}

func TestRunCreateDraftUsesFlagsAndAutoGeneratesDesc(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldNewDraftCreator, oldUploadCoverImageFn := newDraftCreator, uploadCoverImageFn
	oldFile, oldCover := createDraftFile, createDraftCover
	oldTitle, oldDesc := createDraftTitle, createDraftDesc
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		newDraftCreator, uploadCoverImageFn = oldNewDraftCreator, oldUploadCoverImageFn
		createDraftFile, createDraftCover = oldFile, oldCover
		createDraftTitle, createDraftDesc = oldTitle, oldDesc
	})

	cfg = &config.Config{WechatAppID: "appid", WechatSecret: "secret"}
	log = zap.NewNop()

	htmlFile := filepath.Join(t.TempDir(), "article.html")
	if err := os.WriteFile(htmlFile, []byte("<h1>标题</h1><p>第一段内容</p>"), 0600); err != nil {
		t.Fatalf("write html: %v", err)
	}

	drafter := &fakeDraftCreator{result: &publish.DraftResult{MediaID: "draft-3", DraftURL: "https://example.com/draft/3"}}
	newDraftCreator = func() publish.DraftCreator { return drafter }
	uploadCoverImageFn = func(imagePath string) (string, error) {
		if imagePath != "/tmp/cover.jpg" {
			t.Fatalf("cover image path = %q", imagePath)
		}
		return "cover-media-id", nil
	}

	createDraftFile = htmlFile
	createDraftCover = "/tmp/cover.jpg"
	createDraftTitle = "手动标题"
	createDraftDesc = ""

	result, err := runCreateDraft()
	if err != nil {
		t.Fatalf("runCreateDraft() error = %v", err)
	}
	if result.MediaID != "draft-3" || result.DraftURL != "https://example.com/draft/3" {
		t.Fatalf("runCreateDraft() result = %#v", result)
	}
	if len(drafter.artifacts) != 1 {
		t.Fatalf("drafter artifacts = %#v", drafter.artifacts)
	}

	artifact := drafter.artifacts[0]
	if artifact.Metadata.Title != "手动标题" {
		t.Fatalf("artifact title = %#v", artifact)
	}
	if artifact.Metadata.Digest != "标题\n第一段内容" {
		t.Fatalf("artifact digest = %#v", artifact)
	}
	if artifact.CoverMediaID != "cover-media-id" {
		t.Fatalf("artifact cover = %#v", artifact)
	}
}

func TestTestDraftCmdOutputsStableEnvelope(t *testing.T) {
	oldCfg, oldLog := cfg, log
	oldNewDraftCreator, oldUploadCoverImageFn := newDraftCreator, uploadCoverImageFn
	t.Cleanup(func() {
		cfg, log = oldCfg, oldLog
		newDraftCreator, uploadCoverImageFn = oldNewDraftCreator, oldUploadCoverImageFn
		testHTMLCmd.SetArgs(nil)
	})

	cfg = &config.Config{WechatAppID: "appid", WechatSecret: "secret"}
	log = zap.NewNop()

	htmlFile := filepath.Join(t.TempDir(), "article.html")
	if err := os.WriteFile(htmlFile, []byte("<p>Hello</p>"), 0600); err != nil {
		t.Fatalf("write html: %v", err)
	}

	drafter := &fakeDraftCreator{result: &publish.DraftResult{MediaID: "draft-2", DraftURL: "https://example.com/draft"}}
	newDraftCreator = func() publish.DraftCreator { return drafter }
	uploadCoverImageFn = func(imagePath string) (string, error) {
		if imagePath != "/tmp/cover.jpg" {
			t.Fatalf("cover image path = %q", imagePath)
		}
		return "cover-media-id", nil
	}

	testHTMLCmd.SetArgs([]string{htmlFile, "/tmp/cover.jpg"})

	stdout := captureStdout(t, func() {
		if err := testHTMLCmd.Execute(); err != nil {
			t.Fatalf("testHTMLCmd.Execute() error = %v", err)
		}
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["success"] != true || response["code"] != codeTestDraftCreated {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response["schema_version"] != "v1" || response["status"] != "completed" || response["retryable"] != false {
		t.Fatalf("unexpected envelope: %#v", response)
	}
	data, _ := response["data"].(map[string]any)
	if data["media_id"] != "draft-2" {
		t.Fatalf("unexpected draft data: %#v", data)
	}
}
