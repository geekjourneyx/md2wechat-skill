package image

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
)

func TestNewVolcengineProviderDefaults(t *testing.T) {
	p, err := NewVolcengineProvider(&config.Config{
		ImageAPIKey: "test-key",
	})
	if err != nil {
		t.Fatalf("NewVolcengineProvider() error = %v", err)
	}

	if p.Name() != "Volcengine" {
		t.Fatalf("Name() = %q", p.Name())
	}
	if p.baseURL != "https://ark.cn-beijing.volces.com/api/v3" {
		t.Fatalf("baseURL = %q", p.baseURL)
	}
	if p.model != "doubao-seedream-5-0-pro-260628" {
		t.Fatalf("model = %q", p.model)
	}
	if p.size != "2K" {
		t.Fatalf("size = %q", p.size)
	}
	if p.outputFormat != "png" {
		t.Fatalf("outputFormat = %q", p.outputFormat)
	}
}

func TestVolcengineProviderGenerate(t *testing.T) {
	cfg := &config.Config{
		ImageAPIKey:  "test-key",
		ImageAPIBase: "https://mock.local/api/v3",
		ImageModel:   "doubao-seedream-5-0-pro-260628",
		ImageSize:    "2K",
	}

	p, err := NewVolcengineProvider(cfg)
	if err != nil {
		t.Fatalf("NewVolcengineProvider() error = %v", err)
	}

	p.client = newMockHTTPClient(func(r *http.Request) (*http.Response, error) {
		if r.Method != "POST" {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/api/v3/images/generations" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}

		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if reqBody["model"] != "doubao-seedream-5-0-pro-260628" {
			t.Fatalf("model = %v", reqBody["model"])
		}
		if reqBody["size"] != "2K" {
			t.Fatalf("size = %v", reqBody["size"])
		}
		if reqBody["output_format"] != "png" {
			t.Fatalf("output_format = %v", reqBody["output_format"])
		}
		if reqBody["watermark"] != false {
			t.Fatalf("watermark = %v", reqBody["watermark"])
		}

		return jsonResponse(http.StatusOK, map[string]any{
			"model": "doubao-seedream-5-0-pro-260628",
			"data": []map[string]any{
				{
					"url":  "https://example.com/generated.png",
					"size": "1664x2496",
				},
			},
		}), nil
	})

	result, err := p.Generate(context.Background(), "portrait prompt")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.URL != "https://example.com/generated.png" {
		t.Fatalf("url = %q", result.URL)
	}
	if result.Model != "doubao-seedream-5-0-pro-260628" {
		t.Fatalf("model = %q", result.Model)
	}
	if result.Size != "1664x2496" {
		t.Fatalf("size = %q", result.Size)
	}
}

func TestVolcengineProviderGenerateModelNotOpen(t *testing.T) {
	cfg := &config.Config{
		ImageAPIKey:  "test-key",
		ImageAPIBase: "https://mock.local/api/v3",
		ImageModel:   "doubao-seedream-5-0-pro-260628",
	}

	p, err := NewVolcengineProvider(cfg)
	if err != nil {
		t.Fatalf("NewVolcengineProvider() error = %v", err)
	}

	p.client = newMockHTTPClient(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusNotFound, map[string]any{
			"error": map[string]any{
				"code":    "ModelNotOpen",
				"message": "Your account has not activated the model doubao-seedream-5-0-pro-260628.",
				"type":    "Not Found",
			},
		}), nil
	})

	_, err = p.Generate(context.Background(), "portrait prompt")
	if err == nil {
		t.Fatal("expected error")
	}

	genErr, ok := err.(*GenerateError)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if genErr.Code != "model_not_open" {
		t.Fatalf("code = %q", genErr.Code)
	}
	if genErr.Provider != "Volcengine" {
		t.Fatalf("provider = %q", genErr.Provider)
	}
	if !strings.Contains(genErr.Hint, "豆包大模型控制台") {
		t.Fatalf("hint = %q", genErr.Hint)
	}
	if !strings.Contains(genErr.Hint, "开通管理") {
		t.Fatalf("hint = %q", genErr.Hint)
	}
	if !strings.Contains(genErr.Hint, "doubao-seedream-5-0-pro-260628") {
		t.Fatalf("hint = %q", genErr.Hint)
	}
}
