package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"github.com/geekjourneyx/md2wechat-skill/internal/image"
)

func TestGenerateImageMiniMaxH3(t *testing.T) {
	oldCfg, oldFactory := cfg, newImageProcessorWithConfig
	t.Cleanup(func() { cfg, newImageProcessorWithConfig = oldCfg, oldFactory })
	const imageURL = "https://cdn.example/result.jpg"
	for _, tc := range []struct {
		name, model, size, sizeField string
		status                       int
		body, wantError              string
	}{
		{"dimensions", "MiniMax-H3-Image", "2720x1536", "size", 200, `{"data":[{"url":"https://cdn.example/result.jpg","size":"2720x1536"}]}`, ""},
		{"ratio", "MiniMax-H3-Image-Highspeed", "16:9", "aspect_ratio", 200, `{"data":[{"url":"https://cdn.example/result.jpg","size":"2720x1536"}]}`, ""},
		{"item_error", "MiniMax-H3-Image", "1:1", "aspect_ratio", 200, `{"data":[{"error":{"code":"1027","message":"output image failed content safety check"}}]}`, "output image failed content safety check"},
		{"http_error", "MiniMax-H3-Image", "1:1", "aspect_ratio", 429, `{"type":"error","error":{"type":"rate_limit_error","message":"IPM exceeded","http_code":"429"}}`, "IPM exceeded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path != "/v1/images/generations" || r.Method != "POST" || r.Header.Get("Authorization") != "Bearer image-key" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				want := map[string]any{"model": tc.model, "prompt": "A windowsill", "response_format": "url", "n": float64(1), tc.sizeField: tc.size}
				if !reflect.DeepEqual(payload, want) {
					t.Errorf("request = %#v, want %#v", payload, want)
				}
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()
			cfg = &config.Config{WechatAppID: "appid", WechatSecret: "secret", ImageAPIKey: "image-key", ImageProvider: "minimax", ImageAPIBase: server.URL}
			uploaded := false
			newImageProcessorWithConfig = func(c *config.Config) imageProcessor {
				return image.NewProcessor(c, nil, image.WithDownloadFunc(func(url string) (string, error) {
					if url != imageURL {
						t.Fatalf("download URL = %q", url)
					}
					return filepath.Join(t.TempDir(), "image.jpg"), nil
				}), image.WithUploadFunc(func(string) (*image.UploadResult, error) {
					uploaded = true
					return &image.UploadResult{MediaID: "media-id"}, nil
				}))
			}
			err := runGenerateImageWithInput(generateImageInput{RawPrompt: "A windowsill", Model: tc.model, Size: tc.size})
			if tc.wantError == "" {
				if err != nil || !uploaded {
					t.Fatalf("generate_image: uploaded=%v, error=%v", uploaded, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantError) || uploaded {
				t.Fatalf("generate_image: uploaded=%v, error=%v, want %q", uploaded, err, tc.wantError)
			}
			if got := calls.Load(); got != 1 {
				t.Fatalf("generation requests = %d, want 1", got)
			}
		})
	}
}
