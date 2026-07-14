package publish

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/image"
)

func TestAssetPipelineProcessRewritesHTMLAndAssets(t *testing.T) {
	dir := t.TempDir()
	localPath := filepath.Join(dir, "images", "local.png")
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		t.Fatalf("create image directory: %v", err)
	}
	if err := os.WriteFile(localPath, []byte("png"), 0o644); err != nil {
		t.Fatalf("write local image: %v", err)
	}

	pipeline := NewAssetPipeline(&fakeAssetProcessor{
		localResults: map[string]*image.UploadResult{
			localPath: {MediaID: "m-local", WechatURL: "https://wechat.local/local"},
		},
		onlineResults: map[string]*image.UploadResult{
			"https://example.com/remote.png": {MediaID: "m-remote", WechatURL: "https://wechat.local/remote"},
		},
		generateResults: map[string]*image.GenerateAndUploadResult{
			"draw fox": {MediaID: "m-ai", WechatURL: "https://wechat.local/ai"},
		},
	})

	output, err := pipeline.Process(&ProcessInput{
		HTML: `<img src="images/local.png"><img src="https://example.com/remote.png"><img src="https://old.example/ai.png">`,
		Assets: []AssetRef{
			{Index: 0, Kind: AssetKindLocal, Source: filepath.Join("images", "local.png"), Placeholder: "<!-- IMG:0 -->"},
			{Index: 1, Kind: AssetKindRemote, Source: "https://example.com/remote.png", Placeholder: "<!-- IMG:1 -->"},
			{Index: 2, Kind: AssetKindAI, Source: "draw fox", Prompt: "draw fox", Placeholder: "<!-- IMG:2 -->"},
		},
		MarkdownDir: dir,
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	for _, expected := range []string{
		"https://wechat.local/local",
		"https://wechat.local/remote",
		"https://wechat.local/ai",
	} {
		if !strings.Contains(output.HTML, expected) {
			t.Fatalf("output HTML missing %q: %s", expected, output.HTML)
		}
	}
	if output.Assets[0].ResolvedSource != localPath {
		t.Fatalf("resolved source = %q, want %q", output.Assets[0].ResolvedSource, localPath)
	}
	if output.Assets[2].MediaID != "m-ai" {
		t.Fatalf("AI asset media id = %q", output.Assets[2].MediaID)
	}
}

func TestAssetPipelineProcessRejectsStructuralAssetErrorsBeforeProcessorCalls(t *testing.T) {
	dir := t.TempDir()
	existingPNG := filepath.Join(dir, "existing.png")
	existingTXT := filepath.Join(dir, "existing.txt")
	for path, contents := range map[string]string{
		existingPNG: "png",
		existingTXT: "text",
	} {
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("write fixture %s: %v", path, err)
		}
	}

	missingPNG := filepath.Join(t.TempDir(), "missing.png")
	directoryPath := t.TempDir()
	tests := []struct {
		name   string
		assets []AssetRef
	}{
		{
			name: "second local asset is missing",
			assets: []AssetRef{
				{Kind: AssetKindLocal, Source: existingPNG},
				{Kind: AssetKindLocal, Source: missingPNG},
			},
		},
		{
			name: "second local asset has unsupported extension",
			assets: []AssetRef{
				{Kind: AssetKindLocal, Source: existingPNG},
				{Kind: AssetKindLocal, Source: existingTXT},
			},
		},
		{
			name: "later local asset path is blank",
			assets: []AssetRef{
				{Kind: AssetKindLocal, Source: existingPNG},
				{Kind: AssetKindLocal, Source: " "},
			},
		},
		{
			name: "later local asset is a directory",
			assets: []AssetRef{
				{Kind: AssetKindLocal, Source: existingPNG},
				{Kind: AssetKindLocal, Source: directoryPath},
			},
		},
		{
			name: "later remote asset URL is blank",
			assets: []AssetRef{
				{Kind: AssetKindRemote, Source: "https://example.com/a.png"},
				{Kind: AssetKindRemote, Source: " "},
			},
		},
		{
			name: "later AI asset prompt is blank",
			assets: []AssetRef{
				{Kind: AssetKindRemote, Source: "https://example.com/a.png"},
				{Kind: AssetKindAI, Prompt: " "},
			},
		},
		{
			name: "later asset kind is unsupported",
			assets: []AssetRef{
				{Kind: AssetKindRemote, Source: "https://example.com/a.png"},
				{Kind: AssetKind("unknown"), Source: "x"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := &fakeAssetProcessor{
				localResults: map[string]*image.UploadResult{
					existingPNG: {MediaID: "local-existing", WechatURL: "https://wechat.local/existing"},
					existingTXT: {MediaID: "local-text", WechatURL: "https://wechat.local/text"},
					missingPNG:  {MediaID: "local-missing", WechatURL: "https://wechat.local/missing"},
					" ":         {MediaID: "local-blank", WechatURL: "https://wechat.local/blank"},
					directoryPath: {
						MediaID:   "local-directory",
						WechatURL: "https://wechat.local/directory",
					},
				},
				onlineResults: map[string]*image.UploadResult{
					"https://example.com/a.png": {MediaID: "remote-a", WechatURL: "https://wechat.local/a"},
					" ":                         {MediaID: "remote-blank", WechatURL: "https://wechat.local/blank"},
				},
				generateResults: map[string]*image.GenerateAndUploadResult{
					" ": {MediaID: "ai-blank", WechatURL: "https://wechat.local/ai-blank"},
				},
			}
			pipeline := NewAssetPipeline(processor)

			_, err := pipeline.Process(&ProcessInput{Assets: tt.assets})
			if err == nil {
				t.Fatal("Process() error = nil, want structural asset error")
			}
			if calls := processor.totalCalls(); calls != 0 {
				t.Fatalf("processor calls = %d, want 0", calls)
			}
		})
	}
}

func TestAssetPipelineProcessReturnsTypedStageErrorInput(t *testing.T) {
	pipeline := NewAssetPipeline(&fakeAssetProcessor{
		onlineErrs: map[string]error{
			"https://example.com/remote.png": fmt.Errorf("download failed"),
		},
	})

	_, err := pipeline.Process(&ProcessInput{
		HTML: `<img src="https://example.com/remote.png">`,
		Assets: []AssetRef{
			{Index: 0, Kind: AssetKindRemote, Source: "https://example.com/remote.png", Placeholder: "<!-- IMG:0 -->"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "download failed") {
		t.Fatalf("Process() error = %v", err)
	}
}
