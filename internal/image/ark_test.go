package image

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
)

func TestNewArkProviderDefaults(t *testing.T) {
	p, err := NewArkProvider(&config.Config{
		ImageAPIKey: "test-key",
	})
	if err != nil {
		t.Fatalf("NewArkProvider() error = %v", err)
	}
	if p.Name() != "Ark" {
		t.Fatalf("Name() = %v, want Ark", p.Name())
	}
	if p.baseURL != "https://ark.cn-beijing.volces.com/api/v3" {
		t.Fatalf("baseURL = %q", p.baseURL)
	}
	if p.model != "doubao-seedream-5.0-lite" {
		t.Fatalf("model = %q", p.model)
	}
	if p.size != "2K" {
		t.Fatalf("size = %q", p.size)
	}
}

func TestGetArkSupportedModels(t *testing.T) {
	models := GetArkSupportedModels()
	if len(models) == 0 {
		t.Fatal("GetArkSupportedModels() returned empty list")
	}
	found := false
	for _, model := range models {
		if model == "doubao-seedream-5.0-lite" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("doubao-seedream-5.0-lite not found in supported models")
	}
}

func TestArkGenerateUsesSeedreamEndpoint(t *testing.T) {
	p, err := NewArkProvider(&config.Config{
		ImageAPIKey: "test-key",
	})
	if err != nil {
		t.Fatalf("NewArkProvider() error = %v", err)
	}

	p.client = newMockHTTPClient(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.URL.String(); got != "https://ark.cn-beijing.volces.com/api/v3/images/generations" {
			t.Fatalf("url = %s", got)
		}
		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if reqBody["model"] != "doubao-seedream-5.0-lite" {
			t.Fatalf("model = %v", reqBody["model"])
		}
		if reqBody["size"] != "2K" {
			t.Fatalf("size = %v", reqBody["size"])
		}
		return jsonResponse(http.StatusOK, map[string]any{
			"data": []map[string]any{{
				"url": "https://example.com/generated.png",
			}},
		}), nil
	})

	result, err := p.Generate(context.Background(), "a clean editorial cover")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.URL != "https://example.com/generated.png" {
		t.Fatalf("URL = %q", result.URL)
	}
	if result.Model != "doubao-seedream-5.0-lite" {
		t.Fatalf("Model = %q", result.Model)
	}
	if result.Size != "2K" {
		t.Fatalf("Size = %q", result.Size)
	}
}
