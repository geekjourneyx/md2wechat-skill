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

func TestGenerateImageSeedreamModels(t *testing.T) {
	oldCfg, oldFactory := cfg, newImageProcessorWithConfig
	t.Cleanup(func() { cfg, newImageProcessorWithConfig = oldCfg, oldFactory })
	const imageURL = "https://cdn.example/result.jpg"
	for _, tc := range []struct {
		name, model, size string
		status            int
		body, wantError   string
	}{
		{"seedream55", "doubao-seedream-5-5-261028", "2K", 200, `{"data":[{"url":"https://cdn.example/result.jpg","size":"2048x2048"}]}`, ""},
		{"flash", "doubao-seedream-5-0-flash-260915", "1K", 200, `{"data":[{"url":"https://cdn.example/result.jpg","size":"1024x1024"}]}`, ""},
		{"dimensions", "doubao-seedream-5-5-261028", "1536x1024", 200, `{"data":[{"url":"https://cdn.example/result.jpg","size":"1536x1024"}]}`, ""},
		{"not_open", "doubao-seedream-5-5-261028", "2K", 404, `{"error":{"code":"ModelNotOpen","message":"model not activated"}}`, "当前账户尚未开通所选模型"},
		{"rate_limit", "doubao-seedream-5-0-flash-260915", "2K", 429, `{"error":{"code":"RateLimitExceeded","message":"rate limited"}}`, "请求过于频繁"},
		{"empty_data", "doubao-seedream-5-5-261028", "2K", 200, `{"data":[]}`, "未生成图片"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path != "/api/v3/images/generations" || r.Method != "POST" || r.Header.Get("Authorization") != "Bearer image-key" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				want := map[string]any{"model": tc.model, "prompt": "A windowsill", "output_format": "png", "watermark": false, "size": tc.size}
				if !reflect.DeepEqual(payload, want) {
					t.Errorf("request = %#v, want %#v", payload, want)
				}
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()
			cfg = &config.Config{WechatAppID: "appid", WechatSecret: "secret", ImageAPIKey: "image-key", ImageProvider: "volcengine", ImageModel: "doubao-seedream-5-0-pro-260628", ImageAPIBase: server.URL + "/api/v3"}
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
