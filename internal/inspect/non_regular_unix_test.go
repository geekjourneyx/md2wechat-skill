//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package inspect

import (
	"path/filepath"
	"slices"
	"syscall"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
)

func TestRunRejectsNamedPipesAsNonRegularPublishFiles(t *testing.T) {
	fullCfg := &config.Config{
		MD2WechatAPIKey: "api-key",
		WechatAppID:     "appid",
		WechatSecret:    "secret",
	}
	dir := t.TempDir()
	articleAsset := filepath.Join(dir, "article-image.png")
	cover := filepath.Join(dir, "cover.jpg")
	for _, path := range []string{articleAsset, cover} {
		if err := syscall.Mkfifo(path, 0o600); err != nil {
			t.Fatalf("create FIFO %s: %v", path, err)
		}
	}

	tests := []struct {
		name       string
		input      Input
		code       string
		wantBlocks []string
	}{
		{
			name: "article asset",
			input: Input{
				MarkdownFile:    filepath.Join(dir, "article.md"),
				Markdown:        "# Title\n\n![local](article-image.png)\n",
				Mode:            "api",
				UploadRequested: true,
				Config:          fullCfg,
			},
			code:       "LOCAL_IMAGE_NOT_FILE",
			wantBlocks: []string{"upload", "draft"},
		},
		{
			name: "local cover",
			input: Input{
				MarkdownFile:   filepath.Join(dir, "article.md"),
				Markdown:       "# Title\n",
				Mode:           "api",
				DraftRequested: true,
				CoverImagePath: cover,
				Config:         fullCfg,
			},
			code:       "COVER_IMAGE_NOT_FILE",
			wantBlocks: []string{"draft"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Run(&tt.input)
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			check, ok := findCheck(result.Checks, tt.code)
			if !ok || check.Level != LevelError {
				t.Fatalf("check %s = %#v, found=%v", tt.code, check, ok)
			}
			blocker, ok := findReadinessBlocker(result.Readiness.Blockers, tt.code)
			if !ok || !slices.Equal(blocker.Blocks, tt.wantBlocks) {
				t.Fatalf("blocker %s = %#v, found=%v", tt.code, blocker, ok)
			}
		})
	}
}
