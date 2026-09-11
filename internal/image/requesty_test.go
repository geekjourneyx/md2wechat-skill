package image

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
)

const requestyTestDefaultModel = "vertex/gemini-3.1-flash-image"

func requestyImageResponse(dataURL string) map[string]any {
	return map[string]any{
		"choices": []map[string]any{
			{
				"message": map[string]any{
					"content": "",
					"images": []map[string]any{
						{
							"type":      "image_url",
							"image_url": map[string]string{"url": dataURL},
						},
					},
				},
			},
		},
	}
}

func requestyPNGDataURL() string {
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngData)
}

func TestNewRequestyProvider(t *testing.T) {
	tests := []struct {
		name            string
		cfg             *config.Config
		wantModel       string
		wantAspectRatio string
		wantImageSize   string
	}{
		{
			name:            "default values follow the Requesty API defaults",
			cfg:             &config.Config{ImageAPIKey: "test-key"},
			wantModel:       requestyTestDefaultModel,
			wantAspectRatio: "1:1",
			wantImageSize:   "1K",
		},
		{
			name: "custom model and exact 2K dimensions",
			cfg: &config.Config{
				ImageAPIKey: "test-key",
				ImageModel:  "vertex/gemini-2.5-flash-image",
				ImageSize:   "2752x1536",
			},
			wantModel:       "vertex/gemini-2.5-flash-image",
			wantAspectRatio: "16:9",
			wantImageSize:   "2K",
		},
		{
			name:            "aspect ratio alone stays at 1K",
			cfg:             &config.Config{ImageAPIKey: "test-key", ImageSize: "16:9"},
			wantModel:       requestyTestDefaultModel,
			wantAspectRatio: "16:9",
			wantImageSize:   "1K",
		},
		{
			name:            "size tier alone stays square",
			cfg:             &config.Config{ImageAPIKey: "test-key", ImageSize: "4K"},
			wantModel:       requestyTestDefaultModel,
			wantAspectRatio: "1:1",
			wantImageSize:   "4K",
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

func TestNewRequestyProviderRejectsUnsupportedSize(t *testing.T) {
	for _, size := range []string{"1920x1080", "1344x768", "3K", "16x9", "square", "1:1 2K"} {
		t.Run(size, func(t *testing.T) {
			_, err := NewRequestyProvider(&config.Config{ImageAPIKey: "test-key", ImageSize: size})
			if err == nil {
				t.Fatalf("expected %q to be rejected", size)
			}
			var cfgErr *config.ConfigError
			if !errors.As(err, &cfgErr) {
				t.Fatalf("error type = %T, want *config.ConfigError", err)
			}
			if cfgErr.Field != "ImageSize" {
				t.Errorf("Field = %q, want ImageSize", cfgErr.Field)
			}
			if !strings.Contains(cfgErr.Message, size) {
				t.Errorf("Message %q should mention the rejected size %q", cfgErr.Message, size)
			}
		})
	}
}

func TestMapSizeToRequestyCoversEveryDocumentedDimension(t *testing.T) {
	tiers := GetRequestySupportedImageSizes()
	for ratio, dims := range GetRequestySupportedDimensions() {
		if len(dims) != len(tiers) {
			t.Fatalf("%s has %d dimensions, want one per tier (%d)", ratio, len(dims), len(tiers))
		}
		for i, dim := range dims {
			gotRatio, gotSize, err := mapSizeToRequesty(dim)
			if err != nil {
				t.Fatalf("mapSizeToRequesty(%q) error = %v", dim, err)
			}
			if gotRatio != ratio || gotSize != tiers[i] {
				t.Errorf("mapSizeToRequesty(%q) = %s/%s, want %s/%s", dim, gotRatio, gotSize, ratio, tiers[i])
			}
		}

		gotRatio, gotSize, err := mapSizeToRequesty(ratio)
		if err != nil || gotRatio != ratio || gotSize != "1K" {
			t.Errorf("mapSizeToRequesty(%q) = %s/%s/%v, want %s/1K", ratio, gotRatio, gotSize, err, ratio)
		}
	}
}

func TestMapSizeToRequestyIsCaseAndSpaceTolerant(t *testing.T) {
	ratio, size, err := mapSizeToRequesty(" 2k ")
	if err != nil || ratio != "1:1" || size != "2K" {
		t.Fatalf("mapSizeToRequesty(\" 2k \") = %s/%s/%v, want 1:1/2K", ratio, size, err)
	}
	ratio, size, err = mapSizeToRequesty("1376X768")
	if err != nil || ratio != "16:9" || size != "1K" {
		t.Fatalf("mapSizeToRequesty(\"1376X768\") = %s/%s/%v, want 16:9/1K", ratio, size, err)
	}
}

func TestRequestyProvider_DefaultBaseURL(t *testing.T) {
	p, err := NewRequestyProvider(&config.Config{ImageAPIKey: "test-key"})
	if err != nil {
		t.Fatalf("NewRequestyProvider() error = %v", err)
	}
	if p.baseURL != "https://router.requesty.ai/v1" {
		t.Errorf("baseURL = %v, want https://router.requesty.ai/v1", p.baseURL)
	}
}

func TestRequestyProvider_TrimsTrailingSlashFromBaseURL(t *testing.T) {
	p, err := NewRequestyProvider(&config.Config{
		ImageAPIKey:  "test-key",
		ImageAPIBase: " https://router.eu.requesty.ai/v1/ ",
	})
	if err != nil {
		t.Fatalf("NewRequestyProvider() error = %v", err)
	}
	if p.baseURL != "https://router.eu.requesty.ai/v1" {
		t.Fatalf("baseURL = %q", p.baseURL)
	}

	var gotURL string
	p.client = newMockHTTPClient(func(r *http.Request) (*http.Response, error) {
		gotURL = r.URL.String()
		return jsonResponse(http.StatusOK, requestyImageResponse(requestyPNGDataURL())), nil
	})
	result, err := p.Generate(context.Background(), "prompt")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	_ = os.Remove(result.URL)
	if gotURL != "https://router.eu.requesty.ai/v1/chat/completions" {
		t.Fatalf("request URL = %q", gotURL)
	}
}

func TestRequestyProvider_Generate(t *testing.T) {
	dataURL := requestyPNGDataURL()

	cfg := &config.Config{
		ImageAPIKey:  "test-key",
		ImageAPIBase: "https://mock.local",
		ImageModel:   requestyTestDefaultModel,
		ImageSize:    "16:9",
	}

	p, err := NewRequestyProvider(cfg)
	if err != nil {
		t.Fatalf("NewRequestyProvider() error = %v", err)
	}
	p.client = newMockHTTPClient(func(r *http.Request) (*http.Response, error) {
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

		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		if reqBody["model"] != requestyTestDefaultModel {
			t.Errorf("model = %v, want %v", reqBody["model"], requestyTestDefaultModel)
		}
		if _, present := reqBody["modalities"]; present {
			t.Errorf("request must not carry the OpenRouter modalities field: %v", reqBody["modalities"])
		}
		imageConfig, ok := reqBody["image_config"].(map[string]any)
		if !ok {
			t.Fatalf("image_config missing: %v", reqBody)
		}
		if imageConfig["aspect_ratio"] != "16:9" || imageConfig["image_size"] != "1K" {
			t.Errorf("image_config = %v, want aspect_ratio 16:9 and image_size 1K", imageConfig)
		}
		if len(imageConfig) != 2 {
			t.Errorf("image_config has unexpected keys: %v", imageConfig)
		}

		return jsonResponse(http.StatusOK, requestyImageResponse(dataURL)), nil
	})

	result, err := p.Generate(context.Background(), "a test image prompt")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if result.URL == "" {
		t.Error("URL is empty")
	}
	if !strings.HasSuffix(result.URL, ".png") {
		t.Errorf("saved file %q should keep the png extension from the data URL", result.URL)
	}
	if _, err := os.Stat(result.URL); os.IsNotExist(err) {
		t.Errorf("Generated file does not exist: %s", result.URL)
	} else {
		_ = os.Remove(result.URL)
	}
	if result.Model != requestyTestDefaultModel {
		t.Errorf("Model = %v, want %v", result.Model, requestyTestDefaultModel)
	}
	if result.Size != "16:9" {
		t.Errorf("Size = %v, want 16:9", result.Size)
	}
}

func TestRequestyProvider_Generate_NoImage(t *testing.T) {
	p, _ := NewRequestyProvider(&config.Config{ImageAPIKey: "test-key", ImageAPIBase: "https://mock.local"})
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

func TestRequestyProvider_Generate_MalformedResponses(t *testing.T) {
	tests := []struct {
		name     string
		body     any
		wantCode string
	}{
		{
			name:     "invalid json",
			body:     "{not json",
			wantCode: "decode_error",
		},
		{
			name:     "image url is not a data url",
			body:     requestyImageResponse("https://cdn.example.com/image.png"),
			wantCode: "parse_error",
		},
		{
			name:     "data url with broken base64",
			body:     requestyImageResponse("data:image/png;base64,!!!not-base64!!!"),
			wantCode: "parse_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, _ := NewRequestyProvider(&config.Config{ImageAPIKey: "test-key", ImageAPIBase: "https://mock.local"})
			p.client = newMockHTTPClient(func(r *http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusOK, tt.body), nil
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
		})
	}
}

func TestRequestyProvider_HandleErrorResponse(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		body        string
		wantCode    string
		wantMessage []string
	}{
		{
			name:        "unauthorized keeps the router message",
			statusCode:  http.StatusUnauthorized,
			body:        `{"error":{"origin":"router","message":"Invalid authorization token"}}`,
			wantCode:    "unauthorized",
			wantMessage: []string{"Invalid authorization token"},
		},
		{
			name:        "forbidden is not reported as a balance problem",
			statusCode:  http.StatusForbidden,
			body:        `{"error":{"origin":"router","message":"Model not allowed for this API key"}}`,
			wantCode:    "forbidden",
			wantMessage: []string{"Model not allowed for this API key"},
		},
		{
			name:        "payment required",
			statusCode:  http.StatusPaymentRequired,
			body:        `{"error":{"message":"insufficient balance"}}`,
			wantCode:    "payment_required",
			wantMessage: []string{"insufficient balance"},
		},
		{
			name:        "rate limit",
			statusCode:  http.StatusTooManyRequests,
			body:        `{"error":{"message":"rate limit exceeded"}}`,
			wantCode:    "rate_limit",
			wantMessage: []string{"rate limit exceeded"},
		},
		{
			name:        "unknown model",
			statusCode:  http.StatusNotFound,
			body:        `{"error":{"origin":"router","message":"Model not found: vertex/does-not-exist"}}`,
			wantCode:    "model_not_found",
			wantMessage: []string{"vertex/does-not-exist"},
		},
		{
			name:        "bad request with a string code",
			statusCode:  http.StatusBadRequest,
			body:        `{"error":{"message":"invalid aspect_ratio","type":"invalid_request_error","code":"invalid_value"}}`,
			wantCode:    "bad_request",
			wantMessage: []string{"[invalid_value]", "invalid aspect_ratio"},
		},
		{
			name:        "bad request with a numeric code",
			statusCode:  http.StatusBadRequest,
			body:        `{"error":{"message":"unsupported image_size","code":400}}`,
			wantCode:    "bad_request",
			wantMessage: []string{"[400]", "unsupported image_size"},
		},
		{
			name:        "server error",
			statusCode:  http.StatusInternalServerError,
			body:        `{"error":{"message":"internal error"}}`,
			wantCode:    "unknown",
			wantMessage: []string{"HTTP 500", "internal error"},
		},
		{
			name:        "non json error body still maps by status",
			statusCode:  http.StatusBadGateway,
			body:        `upstream timed out`,
			wantCode:    "unknown",
			wantMessage: []string{"HTTP 502"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, _ := NewRequestyProvider(&config.Config{ImageAPIKey: "test-key", ImageAPIBase: "https://mock.local"})
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
			for _, want := range tt.wantMessage {
				if !strings.Contains(genErr.Message, want) {
					t.Errorf("Message %q should contain %q", genErr.Message, want)
				}
			}
			if genErr.Original == nil || !strings.Contains(genErr.Original.Error(), tt.body) {
				t.Errorf("Original should carry the raw body, got %v", genErr.Original)
			}
		})
	}
}

func TestRequestyProvider_BuildRequest(t *testing.T) {
	p, err := NewRequestyProvider(&config.Config{
		ImageAPIKey: "test-key",
		ImageModel:  "test-model",
		ImageSize:   "3168x1344",
	})
	if err != nil {
		t.Fatalf("NewRequestyProvider() error = %v", err)
	}
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

	if _, present := req["modalities"]; present {
		t.Errorf("modalities must not be sent to Requesty")
	}

	imageConfig, ok := req["image_config"].(map[string]string)
	if !ok {
		t.Fatalf("image_config missing")
	}
	if imageConfig["aspect_ratio"] != "21:9" {
		t.Errorf("aspect_ratio = %v, want 21:9", imageConfig["aspect_ratio"])
	}
	if imageConfig["image_size"] != "2K" {
		t.Errorf("image_size = %v, want 2K", imageConfig["image_size"])
	}
}

func TestNewProviderResolvesRequestyAndAlias(t *testing.T) {
	for _, name := range []string{"requesty", "rq"} {
		t.Run(name, func(t *testing.T) {
			provider, err := NewProvider(&config.Config{ImageProvider: name, ImageAPIKey: "test-key"})
			if err != nil {
				t.Fatalf("NewProvider(%q) error = %v", name, err)
			}
			if _, ok := provider.(*RequestyProvider); !ok {
				t.Fatalf("NewProvider(%q) = %T, want *RequestyProvider", name, provider)
			}
			if provider.Name() != "Requesty" {
				t.Fatalf("Name() = %q", provider.Name())
			}
		})
	}
}

func TestNewProviderRequestyRequiresAPIKey(t *testing.T) {
	_, err := NewProvider(&config.Config{ImageProvider: "rq"})
	if err == nil {
		t.Fatal("expected missing API key to be rejected")
	}
	var cfgErr *config.ConfigError
	if !errors.As(err, &cfgErr) || cfgErr.Field != "ImageAPIKey" {
		t.Fatalf("error = %v, want ConfigError on ImageAPIKey", err)
	}
}

func TestNewProviderRequestyPropagatesSizeError(t *testing.T) {
	_, err := NewProvider(&config.Config{ImageProvider: "requesty", ImageAPIKey: "test-key", ImageSize: "1920x1080"})
	var cfgErr *config.ConfigError
	if !errors.As(err, &cfgErr) || cfgErr.Field != "ImageSize" {
		t.Fatalf("error = %v, want ConfigError on ImageSize", err)
	}
}

func TestLookupProviderMetaSupportsRequestyAlias(t *testing.T) {
	meta, ok := LookupProviderMeta("rq")
	if !ok {
		t.Fatal("expected rq alias to resolve")
	}
	if meta.Name != "requesty" {
		t.Fatalf("Name = %q", meta.Name)
	}
	if meta.DefaultBaseURL != "https://router.requesty.ai/v1" {
		t.Fatalf("DefaultBaseURL = %q", meta.DefaultBaseURL)
	}
	if meta.DefaultModel != requestyTestDefaultModel {
		t.Fatalf("DefaultModel = %q", meta.DefaultModel)
	}
	if !meta.SupportsSize {
		t.Fatal("requesty must advertise size support")
	}
	for _, model := range meta.SupportedModels {
		if strings.HasSuffix(model.Name, "-preview") {
			t.Fatalf("model %q is not served by the Requesty router; use the live catalog ids", model.Name)
		}
	}
}

func TestGetRequestySupportedModels(t *testing.T) {
	models := GetRequestySupportedModels()
	if len(models) == 0 {
		t.Error("No supported models returned")
	}

	found := false
	for _, m := range models {
		if m == requestyTestDefaultModel {
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
	want := []string{"1:1", "2:3", "3:2", "3:4", "4:3", "4:5", "5:4", "9:16", "16:9", "21:9"}
	if len(ratios) != len(want) {
		t.Fatalf("ratios = %v, want %v", ratios, want)
	}
	for i := range want {
		if ratios[i] != want[i] {
			t.Errorf("ratios[%d] = %v, want %v", i, ratios[i], want[i])
		}
	}
}

func TestGetRequestySupportedImageSizes(t *testing.T) {
	sizes := GetRequestySupportedImageSizes()
	want := []string{"1K", "2K", "4K"}
	if len(sizes) != len(want) {
		t.Fatalf("sizes = %v, want %v", sizes, want)
	}
	for i := range want {
		if sizes[i] != want[i] {
			t.Errorf("sizes[%d] = %v, want %v", i, sizes[i], want[i])
		}
	}
}
