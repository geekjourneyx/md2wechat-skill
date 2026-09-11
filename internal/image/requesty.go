package image

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
)

// RequestyProvider is the Requesty image generation provider.
// Requesty exposes image models through its OpenAI compatible chat completions endpoint.
// Contract: https://docs.requesty.ai/features/image-generation
type RequestyProvider struct {
	apiKey      string
	baseURL     string
	model       string
	aspectRatio string // Requesty image_config.aspect_ratio, for example "16:9"
	imageSize   string // Requesty image_config.image_size: 1K, 2K or 4K
	client      *http.Client
}

// requestyAspectRatios lists the aspect ratios accepted by Requesty's image_config, with the
// pixel dimensions produced at each size tier (1K, 2K, 4K). The table mirrors the Requesty docs.
var requestyAspectRatios = []struct {
	ratio      string
	dimensions [3]string // 1K, 2K, 4K
}{
	{"1:1", [3]string{"1024x1024", "2048x2048", "4096x4096"}},
	{"2:3", [3]string{"848x1264", "1696x2528", "3392x5056"}},
	{"3:2", [3]string{"1264x848", "2528x1696", "5056x3392"}},
	{"3:4", [3]string{"896x1200", "1792x2400", "3584x4800"}},
	{"4:3", [3]string{"1200x896", "2400x1792", "4800x3584"}},
	{"4:5", [3]string{"928x1152", "1856x2304", "3712x4608"}},
	{"5:4", [3]string{"1152x928", "2304x1856", "4608x3712"}},
	{"9:16", [3]string{"768x1376", "1536x2752", "3072x5504"}},
	{"16:9", [3]string{"1376x768", "2752x1536", "5504x3072"}},
	{"21:9", [3]string{"1584x672", "3168x1344", "6336x2688"}},
}

// requestyImageSizes lists the size tiers accepted by Requesty's image_config.image_size.
// 1K is what Requesty applies when image_size is omitted.
var requestyImageSizes = []string{"1K", "2K", "4K"}

const (
	requestyDefaultAspectRatio = "1:1"
	requestyDefaultImageSize   = "1K"
)

// NewRequestyProvider creates a Requesty Provider.
func NewRequestyProvider(cfg *config.Config) (*RequestyProvider, error) {
	model := cfg.ImageModel
	if model == "" {
		model = DefaultProviderModel("requesty")
	}

	aspectRatio, imageSize, err := mapSizeToRequesty(cfg.ImageSize)
	if err != nil {
		return nil, err
	}

	baseURL := strings.TrimSpace(cfg.ImageAPIBase)
	if baseURL == "" {
		baseURL = DefaultProviderBaseURL("requesty")
	}
	baseURL = strings.TrimRight(baseURL, "/")

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
// Requesty does not use the OpenRouter "modalities" field: an image model returns images from a
// plain chat completion, and image_config carries the aspect ratio and size tier.
func (p *RequestyProvider) buildRequest(prompt string) map[string]any {
	return map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"image_config": map[string]string{
			"aspect_ratio": p.aspectRatio,
			"image_size":   p.imageSize,
		},
	}
}

// requestyResponse is the Requesty API response structure.
type requestyResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content,omitempty"`
			Images  []struct {
				Type     string `json:"type,omitempty"`
				ImageURL struct {
					URL string `json:"url"`
				} `json:"image_url"`
			} `json:"images,omitempty"`
		} `json:"message"`
	} `json:"choices"`
	Error *requestyError `json:"error,omitempty"`
}

// requestyError is the error envelope returned by the Requesty router.
// Router-originated errors carry "origin":"router" and no code; errors forwarded from the upstream
// model provider may carry a code, which can be a string or a number depending on the provider.
type requestyError struct {
	Message string          `json:"message"`
	Type    string          `json:"type,omitempty"`
	Origin  string          `json:"origin,omitempty"`
	Code    json.RawMessage `json:"code,omitempty"`
}

// codeString returns the error code as text, whatever JSON type Requesty used for it.
func (e *requestyError) codeString() string {
	if e == nil || len(e.Code) == 0 {
		return ""
	}
	var asString string
	if err := json.Unmarshal(e.Code, &asString); err == nil {
		return asString
	}
	return strings.TrimSpace(string(e.Code))
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
		hint := "提示词可能不符合内容政策，请尝试修改提示词"
		if modelsHint := ProviderSupportedModelsHint("requesty"); modelsHint != "" {
			hint += "；也请确认模型支持图片生成。" + modelsHint
		}
		return "", &GenerateError{
			Provider: p.Name(),
			Code:     "no_image",
			Message:  "未生成图片",
			Hint:     hint,
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
// The HTTP status picks the error class; the Requesty message (and code, when present) is kept in
// the user facing message so the real cause is not hidden behind a generic text.
func (p *RequestyProvider) handleErrorResponse(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var errResp requestyResponse
	_ = json.Unmarshal(body, &errResp)

	detail := ""
	if errResp.Error != nil {
		detail = strings.TrimSpace(errResp.Error.Message)
		if code := errResp.Error.codeString(); code != "" {
			detail = fmt.Sprintf("[%s] %s", code, detail)
		}
	}
	withDetail := func(msg string) string {
		if detail == "" {
			return msg
		}
		return msg + ": " + detail
	}
	original := fmt.Errorf("status %d: %s", resp.StatusCode, string(body))

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "unauthorized",
			Message:  withDetail("Requesty API Key 无效或已过期"),
			Hint:     "请检查配置文件中的 api.image_key 是否正确，或前往 https://app.requesty.ai/api-keys 获取新的 API Key",
			Original: original,
		}
	case http.StatusForbidden:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "forbidden",
			Message:  withDetail("Requesty 拒绝了本次请求"),
			Hint:     "请确认 API Key 有效，且该 Key 允许访问所选模型（在 app.requesty.ai 检查 Key 的模型和策略限制）",
			Original: original,
		}
	case http.StatusPaymentRequired:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "payment_required",
			Message:  withDetail("Requesty 账户余额不足"),
			Hint:     "请前往 app.requesty.ai 充值或检查账户余额",
			Original: original,
		}
	case http.StatusTooManyRequests:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "rate_limit",
			Message:  withDetail("请求过于频繁，请稍后重试"),
			Hint:     "Requesty API 有速率限制，请等待一段时间后再试",
			Original: original,
		}
	case http.StatusNotFound:
		hint := "请检查模型名称是否正确"
		if modelsHint := ProviderSupportedModelsHint("requesty"); modelsHint != "" {
			hint += "。" + modelsHint
		}
		return &GenerateError{
			Provider: p.Name(),
			Code:     "model_not_found",
			Message:  withDetail("Requesty 未找到所选模型"),
			Hint:     hint,
			Original: original,
		}
	case http.StatusBadRequest:
		hint := "请检查模型名称、aspect_ratio、image_size 等参数是否正确"
		if modelsHint := ProviderSupportedModelsHint("requesty"); modelsHint != "" {
			hint += "。" + modelsHint
		}
		return &GenerateError{
			Provider: p.Name(),
			Code:     "bad_request",
			Message:  withDetail("请求参数错误"),
			Hint:     hint,
			Original: original,
		}
	default:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "unknown",
			Message:  withDetail(fmt.Sprintf("Requesty API 返回错误 (HTTP %d)", resp.StatusCode)),
			Hint:     "请稍后重试，或访问 status.requesty.ai 查看服务状态",
			Original: original,
		}
	}
}

// mapSizeToRequesty maps IMAGE_SIZE to Requesty's image_config aspect_ratio and image_size.
//
// Accepted forms:
//   - empty: Requesty's own default, 1:1 at 1K
//   - an aspect ratio such as "16:9": that ratio at 1K
//   - a size tier such as "2K": 1:1 at that tier
//   - exact WIDTHxHEIGHT from the Requesty dimension table, such as "2752x1536" (16:9 at 2K)
//
// Anything else is rejected instead of being silently replaced by a different size.
func mapSizeToRequesty(size string) (aspectRatio, imageSize string, err error) {
	size = strings.TrimSpace(size)
	if size == "" {
		return requestyDefaultAspectRatio, requestyDefaultImageSize, nil
	}

	for _, tier := range requestyImageSizes {
		if strings.EqualFold(size, tier) {
			return requestyDefaultAspectRatio, tier, nil
		}
	}

	for _, entry := range requestyAspectRatios {
		if size == entry.ratio {
			return entry.ratio, requestyDefaultImageSize, nil
		}
		for i, dims := range entry.dimensions {
			if strings.EqualFold(size, dims) {
				return entry.ratio, requestyImageSizes[i], nil
			}
		}
	}

	return "", "", &config.ConfigError{
		Field:   "ImageSize",
		Message: fmt.Sprintf("Requesty 不支持的图片尺寸: %q", size),
		Hint: "可填写宽高比（" + strings.Join(GetRequestySupportedAspectRatios(), ", ") +
			"）、分辨率等级（" + strings.Join(GetRequestySupportedImageSizes(), ", ") +
			"）或对应的精确尺寸，例如 16:9、2K、2752x1536；详见 docs/IMAGE_PROVISIONERS.md 的 Requesty 尺寸表",
	}
}

// GetRequestySupportedModels returns the list of image generation models supported by Requesty.
func GetRequestySupportedModels() []string {
	return ProviderSupportedModelNames("requesty")
}

// GetRequestySupportedAspectRatios returns the aspect ratios accepted by Requesty's image_config.
func GetRequestySupportedAspectRatios() []string {
	ratios := make([]string, 0, len(requestyAspectRatios))
	for _, entry := range requestyAspectRatios {
		ratios = append(ratios, entry.ratio)
	}
	return ratios
}

// GetRequestySupportedImageSizes returns the size tiers accepted by Requesty's image_config.
func GetRequestySupportedImageSizes() []string {
	sizes := make([]string, len(requestyImageSizes))
	copy(sizes, requestyImageSizes)
	return sizes
}

// GetRequestySupportedDimensions returns every exact WIDTHxHEIGHT accepted for IMAGE_SIZE,
// grouped by aspect ratio in 1K, 2K, 4K order.
func GetRequestySupportedDimensions() map[string][]string {
	dims := make(map[string][]string, len(requestyAspectRatios))
	for _, entry := range requestyAspectRatios {
		dims[entry.ratio] = append([]string(nil), entry.dimensions[:]...)
	}
	return dims
}
