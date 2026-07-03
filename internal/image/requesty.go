package image

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
)

// RequestyProvider is the Requesty image generation provider.
// Requesty offers a unified OpenAI-compatible API supporting multiple image models.
type RequestyProvider struct {
	apiKey      string
	baseURL     string
	model       string
	aspectRatio string // Requesty uses aspect_ratio rather than WIDTHxHEIGHT
	imageSize   string // 1K/2K/4K
	client      *http.Client
}

// NewRequestyProvider creates a Requesty Provider.
func NewRequestyProvider(cfg *config.Config) (*RequestyProvider, error) {
	model := cfg.ImageModel
	if model == "" {
		model = DefaultProviderModel("requesty")
	}

	// Map IMAGE_SIZE (WIDTHxHEIGHT) to Requesty's aspect_ratio and image_size.
	aspectRatio, imageSize := mapSizeToOpenRouter(cfg.ImageSize)

	baseURL := cfg.ImageAPIBase
	if baseURL == "" {
		baseURL = DefaultProviderBaseURL("requesty")
	}

	return &RequestyProvider{
		apiKey:      cfg.ImageAPIKey,
		baseURL:     baseURL,
		model:       model,
		aspectRatio: aspectRatio,
		imageSize:   imageSize,
		client: &http.Client{
			Timeout: 120 * time.Second, // image generation can take a while
		},
	}, nil
}

// Name returns the provider name.
func (p *RequestyProvider) Name() string {
	return "Requesty"
}

// Generate generates an image.
// Requesty returns a base64-encoded image; this method saves it to a temp file and returns the file path.
func (p *RequestyProvider) Generate(ctx context.Context, prompt string) (*GenerateResult, error) {
	// Build request body (Chat Completions format).
	reqBody := p.buildRequest(prompt)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, &GenerateError{
			Provider: p.Name(),
			Code:     "marshal_error",
			Message:  "请求构造失败",
			Original: err,
		}
	}

	// Create HTTP request.
	url := p.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, &GenerateError{
			Provider: p.Name(),
			Code:     "request_error",
			Message:  "创建请求失败",
			Original: err,
		}
	}

	// Set request headers.
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("HTTP-Referer", "https://md2wechat.cn")
	req.Header.Set("X-Title", "md2wechat")

	// Send request.
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, &GenerateError{
			Provider: p.Name(),
			Code:     "network_error",
			Message:  "网络请求失败，请检查网络连接",
			Hint:     "确认网络连接正常，API 地址正确",
			Original: err,
		}
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Handle error responses.
	if resp.StatusCode != http.StatusOK {
		return nil, p.handleErrorResponse(resp)
	}

	// Parse response and save image to temp file.
	filePath, err := p.parseResponseAndSave(resp.Body)
	if err != nil {
		return nil, err
	}

	return &GenerateResult{
		URL:   filePath, // return local file path
		Model: p.model,
		Size:  p.aspectRatio,
	}, nil
}

// buildRequest builds the Requesty request body (Chat Completions format).
func (p *RequestyProvider) buildRequest(prompt string) map[string]any {
	req := map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"modalities": []string{"image"}, // image only
	}

	// Add image_config (if aspect_ratio or image_size is set).
	imageConfig := map[string]string{}
	if p.aspectRatio != "" {
		imageConfig["aspect_ratio"] = p.aspectRatio
	}
	if p.imageSize != "" {
		imageConfig["image_size"] = p.imageSize
	}
	if len(imageConfig) > 0 {
		req["image_config"] = imageConfig
	}

	return req
}

// requestyResponse is the Requesty API response structure.
type requestyResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content,omitempty"`
			Images  []struct {
				ImageURL struct {
					URL string `json:"url"`
				} `json:"image_url"`
			} `json:"images,omitempty"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// parseResponseAndSave parses the response and saves the base64 image to a temp file.
func (p *RequestyProvider) parseResponseAndSave(body io.Reader) (string, error) {
	var result requestyResponse
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return "", &GenerateError{
			Provider: p.Name(),
			Code:     "decode_error",
			Message:  "响应解析失败",
			Original: err,
		}
	}

	// Check whether an image was returned.
	if len(result.Choices) == 0 || len(result.Choices[0].Message.Images) == 0 {
		return "", &GenerateError{
			Provider: p.Name(),
			Code:     "no_image",
			Message:  "未生成图片",
			Hint:     "提示词可能不符合内容政策，请尝试修改提示词",
		}
	}

	// Get the base64 data URL.
	dataURL := result.Choices[0].Message.Images[0].ImageURL.URL

	// Parse the data URL and decode base64.
	imageData, ext, err := parseDataURL(dataURL)
	if err != nil {
		return "", &GenerateError{
			Provider: p.Name(),
			Code:     "parse_error",
			Message:  "图片数据解析失败",
			Original: err,
		}
	}

	// Save to a temp file.
	tmpFile, err := os.CreateTemp("", "md2wechat-requesty-*"+ext)
	if err != nil {
		return "", &GenerateError{
			Provider: p.Name(),
			Code:     "write_error",
			Message:  "图片保存失败",
			Original: err,
		}
	}
	tmpPath := tmpFile.Name()
	if _, err := tmpFile.Write(imageData); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return "", &GenerateError{
			Provider: p.Name(),
			Code:     "write_error",
			Message:  "图片保存失败",
			Original: err,
		}
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", &GenerateError{
			Provider: p.Name(),
			Code:     "write_error",
			Message:  "图片保存失败",
			Original: err,
		}
	}

	return tmpPath, nil
}

// handleErrorResponse handles error responses.
func (p *RequestyProvider) handleErrorResponse(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var errResp requestyResponse
	_ = json.Unmarshal(body, &errResp)

	errMsg := ""
	if errResp.Error != nil {
		errMsg = errResp.Error.Message
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "unauthorized",
			Message:  "Requesty API Key 无效或已过期",
			Hint:     "请检查配置文件中的 api.image_key 是否正确，或前往 requesty.ai 获取新的 API Key",
			Original: fmt.Errorf("status 401: %s", string(body)),
		}
	case http.StatusTooManyRequests:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "rate_limit",
			Message:  "请求过于频繁，请稍后重试",
			Hint:     "Requesty API 有速率限制，请等待一段时间后再试",
			Original: fmt.Errorf("status 429: %s", string(body)),
		}
	case http.StatusBadRequest:
		hint := "请检查模型名称、aspect_ratio 等参数是否正确"
		if modelsHint := ProviderSupportedModelsHint("requesty"); modelsHint != "" {
			hint += "。" + modelsHint
		}
		return &GenerateError{
			Provider: p.Name(),
			Code:     "bad_request",
			Message:  fmt.Sprintf("请求参数错误: %s", errMsg),
			Hint:     hint,
			Original: fmt.Errorf("status 400: %s", string(body)),
		}
	case http.StatusPaymentRequired, http.StatusForbidden:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "payment_required",
			Message:  "Requesty 账户余额不足或访问受限",
			Hint:     "请前往 requesty.ai 检查账户余额和 API 使用权限",
			Original: fmt.Errorf("status %d: %s", resp.StatusCode, string(body)),
		}
	default:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "unknown",
			Message:  fmt.Sprintf("Requesty API 返回错误 (HTTP %d)", resp.StatusCode),
			Hint:     "请稍后重试，或访问 requesty.ai 查看服务状态",
			Original: fmt.Errorf("status %d: %s", resp.StatusCode, string(body)),
		}
	}
}

// GetRequestySupportedModels returns the list of image generation models supported by Requesty.
func GetRequestySupportedModels() []string {
	return ProviderSupportedModelNames("requesty")
}

// GetRequestySupportedAspectRatios returns the list of aspect ratios supported by Requesty.
func GetRequestySupportedAspectRatios() []string {
	return []string{
		"1:1",  // 1024x1024
		"2:3",  // 832x1248
		"3:2",  // 1248x832
		"3:4",  // 864x1184
		"4:3",  // 1184x864
		"4:5",  // 896x1152
		"5:4",  // 1152x896
		"9:16", // 768x1344
		"16:9", // 1344x768
		"21:9", // 1536x672
	}
}

// GetRequestySupportedImageSizes returns the image size tiers supported by Requesty.
func GetRequestySupportedImageSizes() []string {
	return []string{
		"1K", // standard resolution
		"2K", // higher resolution (default)
		"4K", // highest resolution
	}
}
