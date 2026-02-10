package image

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
)

func TestNewOpenRouterProvider(t *testing.T) {
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
			wantModel:       "google/gemini-3-pro-image-preview",
			wantAspectRatio: "1:1",
			wantImageSize:   "2K",
		},
		{
			name: "custom model and size",
			cfg: &config.Config{
				ImageAPIKey: "test-key",
				ImageModel:  "black-forest-labs/flux.2-pro",
				ImageSize:   "1920x1080",
			},
			wantModel:       "black-forest-labs/flux.2-pro",
			wantAspectRatio: "16:9",
			wantImageSize:   "2K",
		},
		{
			name: "aspect ratio format",
			cfg: &config.Config{
				ImageAPIKey: "test-key",
				ImageSize:   "16:9",
			},
			wantModel:       "google/gemini-3-pro-image-preview",
			wantAspectRatio: "16:9",
			wantImageSize:   "2K",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewOpenRouterProvider(tt.cfg)
			if err != nil {
				t.Fatalf("NewOpenRouterProvider() error = %v", err)
			}

			if p.Name() != "OpenRouter" {
				t.Errorf("Name() = %v, want OpenRouter", p.Name())
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

func TestMapSizeToOpenRouter(t *testing.T) {
	tests := []struct {
		input     string
		wantRatio string
		wantSize  string
	}{
		// 1:1 正方形
		{"1024x1024", "1:1", "1K"},
		{"2048x2048", "1:1", "2K"},
		{"4096x4096", "1:1", "4K"},
		// 16:9 横版
		{"1920x1080", "16:9", "2K"},
		{"2560x1440", "16:9", "2K"},
		{"3840x2160", "16:9", "4K"},
		// 9:16 竖版
		{"1080x1920", "9:16", "2K"},
		{"1440x2560", "9:16", "2K"},
		// 直接使用比例格式
		{"1:1", "1:1", "2K"},
		{"16:9", "16:9", "2K"},
		{"9:16", "9:16", "2K"},
		{"21:9", "21:9", "2K"},
		// 默认值
		{"", "1:1", "2K"},
		// 未知格式回退到默认
		{"unknown", "1:1", "2K"},
		{"123x456", "1:1", "2K"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ratio, size := mapSizeToOpenRouter(tt.input)
			if ratio != tt.wantRatio || size != tt.wantSize {
				t.Errorf("mapSizeToOpenRouter(%q) = (%q, %q), want (%q, %q)",
					tt.input, ratio, size, tt.wantRatio, tt.wantSize)
			}
		})
	}
}

func TestParseDataURL(t *testing.T) {
	tests := []struct {
		name       string
		dataURL    string
		wantExt    string
		wantLen    int
		wantErr    bool
	}{
		{
			name:    "valid PNG",
			dataURL: "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte{0x89, 0x50, 0x4E, 0x47}),
			wantExt: ".png",
			wantLen: 4,
			wantErr: false,
		},
		{
			name:    "valid JPEG",
			dataURL: "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString([]byte{0xFF, 0xD8, 0xFF}),
			wantExt: ".jpg",
			wantLen: 3,
			wantErr: false,
		},
		{
			name:    "valid WebP",
			dataURL: "data:image/webp;base64," + base64.StdEncoding.EncodeToString([]byte{0x52, 0x49, 0x46, 0x46}),
			wantExt: ".webp",
			wantLen: 4,
			wantErr: false,
		},
		{
			name:    "invalid prefix",
			dataURL: "http://example.com/image.png",
			wantErr: true,
		},
		{
			name:    "missing comma",
			dataURL: "data:image/png;base64ABC",
			wantErr: true,
		},
		{
			name:    "invalid base64",
			dataURL: "data:image/png;base64,!!!invalid!!!",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, ext, err := parseDataURL(tt.dataURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDataURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if ext != tt.wantExt {
				t.Errorf("ext = %v, want %v", ext, tt.wantExt)
			}
			if len(data) != tt.wantLen {
				t.Errorf("data length = %v, want %v", len(data), tt.wantLen)
			}
		})
	}
}

func TestOpenRouterProvider_Generate(t *testing.T) {
	// 创建测试用的 PNG 数据 (最小的有效 PNG)
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
	}
	b64 := base64.StdEncoding.EncodeToString(pngData)
	dataURL := "data:image/png;base64," + b64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求
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

		// 验证请求体
		var reqBody map[string]any
		json.NewDecoder(r.Body).Decode(&reqBody)

		if reqBody["model"] != "google/gemini-3-pro-image-preview" {
			t.Errorf("model = %v, want google/gemini-3-pro-image-preview", reqBody["model"])
		}

		modalities, ok := reqBody["modalities"].([]any)
		if !ok || len(modalities) != 1 || modalities[0] != "image" {
			t.Errorf("modalities = %v, want [image]", reqBody["modalities"])
		}

		// 返回模拟响应
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

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{
		ImageAPIKey:  "test-key",
		ImageAPIBase: server.URL,
		ImageModel:   "google/gemini-3-pro-image-preview",
	}

	p, err := NewOpenRouterProvider(cfg)
	if err != nil {
		t.Fatalf("NewOpenRouterProvider() error = %v", err)
	}

	result, err := p.Generate(context.Background(), "a test image prompt")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// 验证结果
	if result.URL == "" {
		t.Error("URL is empty")
	}

	// 验证文件存在
	if _, err := os.Stat(result.URL); os.IsNotExist(err) {
		t.Errorf("Generated file does not exist: %s", result.URL)
	} else {
		os.Remove(result.URL) // 清理临时文件
	}

	if result.Model != "google/gemini-3-pro-image-preview" {
		t.Errorf("Model = %v, want google/gemini-3-pro-image-preview", result.Model)
	}
}

func TestOpenRouterProvider_Generate_NoImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 返回没有图片的响应
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

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{
		ImageAPIKey:  "test-key",
		ImageAPIBase: server.URL,
	}

	p, _ := NewOpenRouterProvider(cfg)
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

func TestOpenRouterProvider_HandleErrorResponse(t *testing.T) {
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
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			cfg := &config.Config{
				ImageAPIKey:  "test-key",
				ImageAPIBase: server.URL,
			}

			p, _ := NewOpenRouterProvider(cfg)
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

			if genErr.Provider != "OpenRouter" {
				t.Errorf("Provider = %v, want OpenRouter", genErr.Provider)
			}
		})
	}
}

func TestOpenRouterProvider_BuildRequest(t *testing.T) {
	cfg := &config.Config{
		ImageAPIKey: "test-key",
		ImageModel:  "test-model",
		ImageSize:   "16:9",
	}

	p, _ := NewOpenRouterProvider(cfg)
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

func TestGetOpenRouterSupportedModels(t *testing.T) {
	models := GetOpenRouterSupportedModels()
	if len(models) == 0 {
		t.Error("No supported models returned")
	}

	// 检查默认模型在列表中
	found := false
	for _, m := range models {
		if m == "google/gemini-3-pro-image-preview" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Default model not in supported list")
	}
}

func TestGetOpenRouterSupportedAspectRatios(t *testing.T) {
	ratios := GetOpenRouterSupportedAspectRatios()
	if len(ratios) == 0 {
		t.Error("No supported aspect ratios returned")
	}

	// 检查常用比例
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

func TestGetOpenRouterSupportedImageSizes(t *testing.T) {
	sizes := GetOpenRouterSupportedImageSizes()
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
