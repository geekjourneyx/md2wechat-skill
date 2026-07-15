//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package publish

import (
	"path/filepath"
	"syscall"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/converter"
	"github.com/geekjourneyx/md2wechat-skill/internal/image"
	"go.uber.org/zap"
)

func makeNamedPipe(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatalf("create FIFO: %v", err)
	}
	return path
}

func TestAssetPipelineRejectsNamedPipeBeforeProcessor(t *testing.T) {
	path := makeNamedPipe(t, "image.png")
	processor := &fakeAssetProcessor{
		localResults: map[string]*image.UploadResult{
			path: {MediaID: "unexpected", WechatURL: "https://example.com/unexpected"},
		},
	}

	_, err := NewAssetPipeline(processor).Process(&ProcessInput{
		Assets: []AssetRef{{Kind: AssetKindLocal, Source: path}},
	})
	if err == nil {
		t.Fatal("Process() error = nil, want non-regular-file rejection")
	}
	if calls := processor.totalCalls(); calls != 0 {
		t.Fatalf("processor calls = %d, want 0", calls)
	}
}

func TestServiceRejectsNamedPipeCoverBeforeEffects(t *testing.T) {
	coverPath := makeNamedPipe(t, "cover.jpg")
	converterFake := &fakeMarkdownConverter{result: &converter.ConvertResult{Success: true}}
	processor := &fakeAssetProcessor{}
	cover := &fakeCoverUploader{}
	drafts := &fakeDraftCreator{}
	svc := NewService(zap.NewNop(), converterFake, processor, drafts, cover.upload)

	_, err := svc.Convert(&ConvertInput{
		Intent:         PublishIntent{CreateDraft: true},
		ConvertRequest: &converter.ConvertRequest{},
		CoverImagePath: coverPath,
	})
	if err == nil {
		t.Fatal("Convert() error = nil, want non-regular cover rejection")
	}
	if converterFake.calls != 0 || processor.totalCalls() != 0 || cover.calls != 0 || drafts.calls != 0 {
		t.Fatalf("named-pipe cover caused effects: converter=%d assets=%d cover=%d draft=%d", converterFake.calls, processor.totalCalls(), cover.calls, drafts.calls)
	}
}

func TestImagePostRejectsNamedPipeBeforePreviewOrCreateEffects(t *testing.T) {
	path := makeNamedPipe(t, "image.jpg")
	processor := &fakeAssetProcessor{
		localResults: map[string]*image.UploadResult{
			path: {MediaID: "unexpected", WechatURL: "https://example.com/unexpected"},
		},
	}
	creator := &fakeImagePostCreator{}
	svc := NewImagePostService(processor, creator)
	input := &ImagePostInput{Title: "Title", Images: []string{path}}

	if _, err := svc.PreviewImagePost(input); err == nil {
		t.Fatal("PreviewImagePost() error = nil, want non-regular image rejection")
	}
	if _, err := svc.CreateImagePost(input); err == nil {
		t.Fatal("CreateImagePost() error = nil, want non-regular image rejection")
	}
	if calls := processor.totalCalls(); calls != 0 {
		t.Fatalf("asset processor calls = %d, want 0", calls)
	}
	if len(creator.artifacts) != 0 {
		t.Fatalf("creator artifacts = %#v, want none", creator.artifacts)
	}
}
