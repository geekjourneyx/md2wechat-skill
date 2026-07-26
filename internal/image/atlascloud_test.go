package image

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
)

func TestAtlasCloudProviderDefaults(t *testing.T) {
	p, err := NewAtlasCloudProvider(&config.Config{ImageAPIKey: "test-key"})
	if err != nil {
		t.Fatalf("NewAtlasCloudProvider() error = %v", err)
	}
	if p.baseURL != "https://api.atlascloud.ai/api/v1/model" {
		t.Fatalf("baseURL = %q", p.baseURL)
	}
	if p.model != "openai/gpt-image-2/text-to-image" {
		t.Fatalf("model = %q", p.model)
	}
	if p.size != "1024x1024" {
		t.Fatalf("size = %q", p.size)
	}
}

func TestAtlasCloudProviderGenerate(t *testing.T) {
	const taskID = "task-123"
	const outputURL = "https://example.com/generated.jpg"
	p, _ := NewAtlasCloudProvider(&config.Config{
		ImageAPIKey:  "test-key",
		ImageAPIBase: "https://mock.local/api/v1/model/",
		ImageModel:   "openai/gpt-image-2/text-to-image",
		ImageSize:    "1024x1024",
	})
	p.pollInterval = time.Millisecond
	p.maxPollTime = time.Second
	pollCount := 0
	p.client = newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("Authorization = %q", req.Header.Get("Authorization"))
		}
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/api/v1/model/generateImage":
			var body map[string]any
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if body["model"] != "openai/gpt-image-2/text-to-image" || body["size"] != "1024x1024" {
				t.Fatalf("request body = %#v", body)
			}
			return jsonResponse(http.StatusOK, map[string]any{
				"code": 200,
				"data": map[string]any{
					"id":     taskID,
					"status": "processing",
					"urls": map[string]string{
						"get": "https://mock.local/api/v1/model/prediction/" + taskID,
					},
				},
			}), nil
		case req.Method == http.MethodGet && req.URL.Path == "/api/v1/model/prediction/"+taskID:
			pollCount++
			status := "processing"
			outputs := []string(nil)
			if pollCount == 2 {
				status = "completed"
				outputs = []string{outputURL}
			}
			return jsonResponse(http.StatusOK, map[string]any{
				"code": 200,
				"data": map[string]any{
					"id":      taskID,
					"model":   "openai/gpt-image-2/text-to-image",
					"status":  status,
					"outputs": outputs,
				},
			}), nil
		default:
			return jsonResponse(http.StatusNotFound, map[string]string{"message": "not found"}), nil
		}
	})

	result, err := p.Generate(context.Background(), "a blue circle")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.URL != outputURL || result.Model != "openai/gpt-image-2/text-to-image" {
		t.Fatalf("result = %#v", result)
	}
	if pollCount != 2 {
		t.Fatalf("pollCount = %d, want 2", pollCount)
	}
}

func TestAtlasCloudProviderRejectsForeignPollURL(t *testing.T) {
	p, _ := NewAtlasCloudProvider(&config.Config{
		ImageAPIKey:  "test-key",
		ImageAPIBase: "https://api.atlascloud.ai/api/v1/model",
	})
	prediction := atlasCloudPrediction{ID: "task-123"}
	prediction.URLs.Get = "https://example.com/task-123"

	_, err := p.resolvePollURL(prediction)
	if err == nil || !strings.Contains(err.Error(), "不属于") {
		t.Fatalf("resolvePollURL() error = %v", err)
	}
}

func TestAtlasCloudProviderTaskFailure(t *testing.T) {
	p, _ := NewAtlasCloudProvider(&config.Config{
		ImageAPIKey:  "test-key",
		ImageAPIBase: "https://mock.local/api/v1/model",
	})
	p.pollInterval = time.Millisecond
	p.maxPollTime = time.Second
	p.client = newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPost {
			return jsonResponse(http.StatusOK, map[string]any{
				"code": 200,
				"data": map[string]any{"id": "task-failed", "status": "processing"},
			}), nil
		}
		return jsonResponse(http.StatusOK, map[string]any{
			"code": 200,
			"data": map[string]any{"id": "task-failed", "status": "failed", "error": "moderation rejected"},
		}), nil
	})

	_, err := p.Generate(context.Background(), "test")
	if err == nil {
		t.Fatal("Generate() should fail")
	}
	genErr, ok := err.(*GenerateError)
	if !ok || genErr.Code != "task_failed" {
		t.Fatalf("error = %#v", err)
	}
}

func TestLookupProviderMetaSupportsAtlasAliases(t *testing.T) {
	for _, alias := range []string{"atlascloud", "atlas-cloud", "atlas"} {
		meta, ok := LookupProviderMeta(alias)
		if !ok || meta.Name != "atlascloud" {
			t.Fatalf("LookupProviderMeta(%q) = %#v, %v", alias, meta, ok)
		}
	}
}

func TestNewProviderSupportsAtlasAlias(t *testing.T) {
	provider, err := NewProvider(&config.Config{
		ImageProvider: "atlas",
		ImageAPIKey:   "test-key",
	})
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}
	if provider.Name() != "Atlas Cloud" {
		t.Fatalf("Name() = %q", provider.Name())
	}
}
