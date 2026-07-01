package image

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
)

func TestNewRequestyProvider(t *testing.T) {
	tests := []struct {
		name            string
		cfg             *config.Config
		wantModel       string
		wantAspectRatio string
		wantImageSize   string
	}{
		{
			name: "default values",
			cfg: &config.Config{
				ImageAPIKey: "test-key",
			},
			wantModel:       "google/gemini-3.1-flash-image-preview",
			wantAspectRatio: "1:1",
			wantImageSize:   "2K",
		},
		{
			name: "custom model and size",
			cfg: &config.Config{
				ImageAPIKey: "test-key",
				ImageModel:  "vertex/gemini-2.5-flash-image",
				ImageSize:   "1920x1080",
			},
			wantModel:       "vertex/gemini-2.5-flash-image",
			wantAspectRatio: "16:9",
			wantImageSize:   "2K",
		},
		{
			name: "aspect ratio format",
			cfg: &config.Config{
				ImageAPIKey: "test-key",
				ImageSize:   "16:9",
			},
			wantModel:       "google/gemini-3.1-flash-image-preview",
			wantAspectRatio: "16:9",
			wantImageSize:   "2K",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewRequestyProvider(tt.cfg)
			if err != nil {
				t.Fatalf("NewRequestyProvider() error = %v", err)
			}

			if p.Name() != "Requesty" {
				t.Errorf("Name() = %v, want Requesty", p.Name())
			}

			if p.model != tt.wantModel {
				t.Errorf("model = %v, want %v", p.model, tt.wantModel)
			}

			if p.aspectRatio != tt.wantAspectRatio {
				t.Errorf("aspectRatio = %v, want %v", p.aspectRatio, tt.wantAspectRatio)
			}

			if p.imageSize != tt.wantImageSize {
				t.Errorf("imageSize = %v, want %v", p.imageSize, tt.wantImageSize)
			}
		})
	}
}

func TestRequestyProvider_DefaultBaseURL(t *testing.T) {
	cfg := &config.Config{ImageAPIKey: "test-key"}
	p, err := NewRequestyProvider(cfg)
	if err != nil {
		t.Fatalf("NewRequestyProvider() error = %v", err)
	}
	if p.baseURL != "https://router.requesty.ai/v1" {
		t.Errorf("baseURL = %v, want https://router.requesty.ai/v1", p.baseURL)
	}
}

func TestRequestyProvider_Generate(t *testing.T) {
	// Minimal valid PNG data for the test.
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
	}
	b64 := base64.StdEncoding.EncodeToString(pngData)
	dataURL := "data:image/png;base64," + b64

	cfg := &config.Config{
		ImageAPIKey:  "test-key",
		ImageAPIBase: "https://mock.local",
		ImageModel:   "google/gemini-3.1-flash-image-preview",
	}

	p, err := NewRequestyProvider(cfg)
	if err != nil {
		t.Fatalf("NewRequestyProvider() error = %v", err)
	}
	p.client = newMockHTTPClient(func(r *http.Request) (*http.Response, error) {
		// Verify request.
		if r.Method != "POST" {
			t.Errorf("Method = %v, want POST", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Path = %v, want /chat/completions", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization header = %v, want Bearer test-key", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %v, want application/json", r.Header.Get("Content-Type"))
		}

		// Verify request body.
		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		if reqBody["model"] != "google/gemini-3.1-flash-image-preview" {
			t.Errorf("model = %v, want google/gemini-3.1-flash-image-preview", reqBody["model"])
		}

		modalities, ok := reqBody["modalities"].([]any)
		if !ok || len(modalities) != 1 || modalities[0] != "image" {
			t.Errorf("modalities = %v, want [image]", reqBody["modalities"])
		}

		response := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"images": []map[string]any{
							{
								"image_url": map[string]string{
									"url": dataURL,
								},
							},
						},
					},
				},
			},
		}

		return jsonResponse(http.StatusOK, response), nil
	})

	result, err := p.Generate(context.Background(), "a test image prompt")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify result.
	if result.URL == "" {
		t.Error("URL is empty")
	}

	// Verify file exists.
	if _, err := os.Stat(result.URL); os.IsNotExist(err) {
		t.Errorf("Generated file does not exist: %s", result.URL)
	} else {
		_ = os.Remove(result.URL) // clean up temp file
	}

	if result.Model != "google/gemini-3.1-flash-image-preview" {
		t.Errorf("Model = %v, want google/gemini-3.1-flash-image-preview", result.Model)
	}
}

func TestRequestyProvider_Generate_NoImage(t *testing.T) {
	cfg := &config.Config{
		ImageAPIKey:  "test-key",
		ImageAPIBase: "https://mock.local",
	}

	p, _ := NewRequestyProvider(cfg)
	p.client = newMockHTTPClient(func(r *http.Request) (*http.Response, error) {
		response := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": "I cannot generate that image",
						"images":  []any{},
					},
				},
			},
		}

		return jsonResponse(http.StatusOK, response), nil
	})
	_, err := p.Generate(context.Background(), "test")

	if err == nil {
		t.Fatal("Expected error for no image response")
	}

	genErr, ok := err.(*GenerateError)
	if !ok {
		t.Fatalf("Error type = %T, want *GenerateError", err)
	}

	if genErr.Code != "no_image" {
		t.Errorf("Error code = %v, want no_image", genErr.Code)
	}
}

func TestRequestyProvider_HandleErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantCode   string
	}{
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"error": {"message": "invalid api key"}}`,
			wantCode:   "unauthorized",
		},
		{
			name:       "rate limit",
			statusCode: http.StatusTooManyRequests,
			body:       `{"error": {"message": "rate limit exceeded"}}`,
			wantCode:   "rate_limit",
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			body:       `{"error": {"message": "invalid model"}}`,
			wantCode:   "bad_request",
		},
		{
			name:       "payment required",
			statusCode: http.StatusPaymentRequired,
			body:       `{"error": {"message": "insufficient balance"}}`,
			wantCode:   "payment_required",
		},
		{
			name:       "forbidden",
			statusCode: http.StatusForbidden,
			body:       `{"error": {"message": "access denied"}}`,
			wantCode:   "payment_required",
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			body:       `{"error": {"message": "internal error"}}`,
			wantCode:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				ImageAPIKey:  "test-key",
				ImageAPIBase: "https://mock.local",
			}

			p, _ := NewRequestyProvider(cfg)
			p.client = newMockHTTPClient(func(r *http.Request) (*http.Response, error) {
				return jsonResponse(tt.statusCode, tt.body), nil
			})
			_, err := p.Generate(context.Background(), "test")

			if err == nil {
				t.Fatal("Expected error")
			}

			genErr, ok := err.(*GenerateError)
			if !ok {
				t.Fatalf("Error type = %T, want *GenerateError", err)
			}

			if genErr.Code != tt.wantCode {
				t.Errorf("Error code = %v, want %v", genErr.Code, tt.wantCode)
			}

			if genErr.Provider != "Requesty" {
				t.Errorf("Provider = %v, want Requesty", genErr.Provider)
			}
		})
	}
}

func TestRequestyProvider_BuildRequest(t *testing.T) {
	cfg := &config.Config{
		ImageAPIKey: "test-key",
		ImageModel:  "test-model",
		ImageSize:   "16:9",
	}

	p, _ := NewRequestyProvider(cfg)
	req := p.buildRequest("test prompt")

	if req["model"] != "test-model" {
		t.Errorf("model = %v, want test-model", req["model"])
	}

	messages, ok := req["messages"].([]map[string]string)
	if !ok || len(messages) != 1 {
		t.Fatalf("messages format incorrect")
	}
	if messages[0]["role"] != "user" || messages[0]["content"] != "test prompt" {
		t.Errorf("message = %v, want {role: user, content: test prompt}", messages[0])
	}

	modalities, ok := req["modalities"].([]string)
	if !ok || len(modalities) != 1 || modalities[0] != "image" {
		t.Errorf("modalities = %v, want [image]", req["modalities"])
	}

	imageConfig, ok := req["image_config"].(map[string]string)
	if !ok {
		t.Fatalf("image_config missing")
	}
	if imageConfig["aspect_ratio"] != "16:9" {
		t.Errorf("aspect_ratio = %v, want 16:9", imageConfig["aspect_ratio"])
	}
}

func TestGetRequestySupportedModels(t *testing.T) {
	models := GetRequestySupportedModels()
	if len(models) == 0 {
		t.Error("No supported models returned")
	}

	// Check that the default model is in the list.
	found := false
	for _, m := range models {
		if m == "google/gemini-3.1-flash-image-preview" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Default model not in supported list")
	}
}

func TestGetRequestySupportedAspectRatios(t *testing.T) {
	ratios := GetRequestySupportedAspectRatios()
	if len(ratios) == 0 {
		t.Error("No supported aspect ratios returned")
	}

	expected := []string{"1:1", "16:9", "9:16"}
	for _, e := range expected {
		found := false
		for _, r := range ratios {
			if r == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected aspect ratio %s not found", e)
		}
	}
}

func TestGetRequestySupportedImageSizes(t *testing.T) {
	sizes := GetRequestySupportedImageSizes()
	if len(sizes) != 3 {
		t.Errorf("Expected 3 image sizes, got %d", len(sizes))
	}

	expected := []string{"1K", "2K", "4K"}
	for i, e := range expected {
		if sizes[i] != e {
			t.Errorf("sizes[%d] = %v, want %v", i, sizes[i], e)
		}
	}
}
