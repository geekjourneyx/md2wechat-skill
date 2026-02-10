package image

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
)

// OpenRouterProvider OpenRouter 图片生成服务提供者
// OpenRouter 提供统一的 API 接口，支持多种图片生成模型（如 Gemini、Flux 等）
type OpenRouterProvider struct {
	apiKey      string
	baseURL     string
	model       string
	aspectRatio string       // OpenRouter 使用 aspect_ratio 而非 WIDTHxHEIGHT
	imageSize   string       // 1K/2K/4K
	client      *http.Client
}

// NewOpenRouterProvider 创建 OpenRouter Provider
func NewOpenRouterProvider(cfg *config.Config) (*OpenRouterProvider, error) {
	model := cfg.ImageModel
	if model == "" {
		model = "google/gemini-3-pro-image-preview" // 默认模型
	}

	// 将 IMAGE_SIZE (WIDTHxHEIGHT) 映射到 OpenRouter 的 aspect_ratio 和 image_size
	aspectRatio, imageSize := mapSizeToOpenRouter(cfg.ImageSize)

	baseURL := cfg.ImageAPIBase
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1" // 默认 API 地址
	}

	return &OpenRouterProvider{
		apiKey:      cfg.ImageAPIKey,
		baseURL:     baseURL,
		model:       model,
		aspectRatio: aspectRatio,
		imageSize:   imageSize,
		client: &http.Client{
			Timeout: 120 * time.Second, // 图片生成可能需要较长时间
		},
	}, nil
}

// Name 返回提供者名称
func (p *OpenRouterProvider) Name() string {
	return "OpenRouter"
}

// Generate 生成图片
// OpenRouter 返回 base64 编码的图片，此方法将其保存为临时文件并返回文件路径
func (p *OpenRouterProvider) Generate(ctx context.Context, prompt string) (*GenerateResult, error) {
	// 构造请求体（Chat Completions 格式）
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

	// 创建 HTTP 请求
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

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("HTTP-Referer", "https://md2wechat.cn")
	req.Header.Set("X-Title", "md2wechat")

	// 发送请求
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
	defer resp.Body.Close()

	// 处理错误响应
	if resp.StatusCode != http.StatusOK {
		return nil, p.handleErrorResponse(resp)
	}

	// 解析响应并保存图片到临时文件
	filePath, err := p.parseResponseAndSave(resp.Body)
	if err != nil {
		return nil, err
	}

	return &GenerateResult{
		URL:   filePath, // 返回本地文件路径
		Model: p.model,
		Size:  p.aspectRatio,
	}, nil
}

// buildRequest 构建 OpenRouter 请求体（Chat Completions 格式）
func (p *OpenRouterProvider) buildRequest(prompt string) map[string]any {
	req := map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"modalities": []string{"image"}, // 仅生成图片
	}

	// 添加 image_config（如果有 aspect_ratio 或 image_size）
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

// openRouterResponse OpenRouter API 响应结构
type openRouterResponse struct {
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

// parseResponseAndSave 解析响应并将 base64 图片保存到临时文件
func (p *OpenRouterProvider) parseResponseAndSave(body io.Reader) (string, error) {
	var result openRouterResponse
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return "", &GenerateError{
			Provider: p.Name(),
			Code:     "decode_error",
			Message:  "响应解析失败",
			Original: err,
		}
	}

	// 检查是否有图片返回
	if len(result.Choices) == 0 || len(result.Choices[0].Message.Images) == 0 {
		return "", &GenerateError{
			Provider: p.Name(),
			Code:     "no_image",
			Message:  "未生成图片",
			Hint:     "提示词可能不符合内容政策，请尝试修改提示词",
		}
	}

	// 获取 base64 data URL
	dataURL := result.Choices[0].Message.Images[0].ImageURL.URL

	// 解析 data URL 并解码 base64
	imageData, ext, err := parseDataURL(dataURL)
	if err != nil {
		return "", &GenerateError{
			Provider: p.Name(),
			Code:     "parse_error",
			Message:  "图片数据解析失败",
			Original: err,
		}
	}

	// 保存到临时文件
	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("md2wechat_openrouter_%d%s", time.Now().UnixNano(), ext))
	if err := os.WriteFile(tmpPath, imageData, 0644); err != nil {
		return "", &GenerateError{
			Provider: p.Name(),
			Code:     "write_error",
			Message:  "图片保存失败",
			Original: err,
		}
	}

	return tmpPath, nil
}

// parseDataURL 解析 data URL 并返回解码后的字节和文件扩展名
// 格式: data:image/png;base64,iVBORw0KGgo...
func parseDataURL(dataURL string) ([]byte, string, error) {
	if !strings.HasPrefix(dataURL, "data:") {
		return nil, "", fmt.Errorf("invalid data URL format: missing 'data:' prefix")
	}

	// 查找逗号分隔符
	commaIdx := strings.Index(dataURL, ",")
	if commaIdx == -1 {
		return nil, "", fmt.Errorf("invalid data URL: no comma separator found")
	}

	// 解析元数据 (e.g., "image/png;base64")
	metadata := dataURL[5:commaIdx] // 跳过 "data:"
	base64Data := dataURL[commaIdx+1:]

	// 确定文件扩展名
	ext := ".png" // 默认 PNG
	if strings.Contains(metadata, "image/jpeg") || strings.Contains(metadata, "image/jpg") {
		ext = ".jpg"
	} else if strings.Contains(metadata, "image/gif") {
		ext = ".gif"
	} else if strings.Contains(metadata, "image/webp") {
		ext = ".webp"
	}

	// 解码 base64
	imageData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, "", fmt.Errorf("base64 decode failed: %w", err)
	}

	return imageData, ext, nil
}

// handleErrorResponse 处理错误响应
func (p *OpenRouterProvider) handleErrorResponse(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var errResp openRouterResponse
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
			Message:  "OpenRouter API Key 无效或已过期",
			Hint:     "请检查配置文件中的 api.image_key 是否正确，或前往 openrouter.ai 获取新的 API Key",
			Original: fmt.Errorf("status 401: %s", string(body)),
		}
	case http.StatusTooManyRequests:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "rate_limit",
			Message:  "请求过于频繁，请稍后重试",
			Hint:     "OpenRouter API 有速率限制，请等待一段时间后再试",
			Original: fmt.Errorf("status 429: %s", string(body)),
		}
	case http.StatusBadRequest:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "bad_request",
			Message:  fmt.Sprintf("请求参数错误: %s", errMsg),
			Hint:     "请检查模型名称、aspect_ratio 等参数是否正确。支持的模型: google/gemini-3-pro-image-preview",
			Original: fmt.Errorf("status 400: %s", string(body)),
		}
	case http.StatusPaymentRequired, http.StatusForbidden:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "payment_required",
			Message:  "OpenRouter 账户余额不足或访问受限",
			Hint:     "请前往 openrouter.ai 检查账户余额和 API 使用权限",
			Original: fmt.Errorf("status %d: %s", resp.StatusCode, string(body)),
		}
	default:
		return &GenerateError{
			Provider: p.Name(),
			Code:     "unknown",
			Message:  fmt.Sprintf("OpenRouter API 返回错误 (HTTP %d)", resp.StatusCode),
			Hint:     "请稍后重试，或访问 openrouter.ai 查看服务状态",
			Original: fmt.Errorf("status %d: %s", resp.StatusCode, string(body)),
		}
	}
}

// mapSizeToOpenRouter 将 WIDTHxHEIGHT 格式映射到 OpenRouter 的 aspect_ratio 和 image_size
func mapSizeToOpenRouter(size string) (aspectRatio, imageSize string) {
	if size == "" {
		return "1:1", "2K" // 默认值
	}

	// 映射表：常见尺寸 → (aspect_ratio, image_size)
	sizeMap := map[string]struct{ ratio, size string }{
		// 1:1 正方形
		"1024x1024": {"1:1", "1K"},
		"2048x2048": {"1:1", "2K"},
		"4096x4096": {"1:1", "4K"},
		// 16:9 横版
		"1344x768":  {"16:9", "1K"},
		"1920x1080": {"16:9", "2K"},
		"2560x1440": {"16:9", "2K"},
		"3840x2160": {"16:9", "4K"},
		// 9:16 竖版
		"768x1344":  {"9:16", "1K"},
		"1080x1920": {"9:16", "2K"},
		"1440x2560": {"9:16", "2K"},
		"2160x3840": {"9:16", "4K"},
		// 4:3 横版
		"1184x864":  {"4:3", "1K"},
		"1600x1200": {"4:3", "2K"},
		"2048x1536": {"4:3", "2K"},
		// 3:4 竖版
		"864x1184":  {"3:4", "1K"},
		"1200x1600": {"3:4", "2K"},
		"1536x2048": {"3:4", "2K"},
		// 3:2 横版
		"1248x832":  {"3:2", "1K"},
		"1800x1200": {"3:2", "2K"},
		"3072x2048": {"3:2", "4K"},
		// 2:3 竖版
		"832x1248":  {"2:3", "1K"},
		"1200x1800": {"2:3", "2K"},
		"2048x3072": {"2:3", "4K"},
		// 5:4 横版
		"1152x896": {"5:4", "1K"},
		// 4:5 竖版
		"896x1152": {"4:5", "1K"},
		// 21:9 超宽
		"1536x672": {"21:9", "1K"},
	}

	if mapped, ok := sizeMap[size]; ok {
		return mapped.ratio, mapped.size
	}

	// 检查是否已经是 aspect_ratio 格式
	validRatios := map[string]bool{
		"1:1": true, "2:3": true, "3:2": true, "3:4": true, "4:3": true,
		"4:5": true, "5:4": true, "9:16": true, "16:9": true, "21:9": true,
	}
	if validRatios[size] {
		return size, "2K" // 使用传入的比例，默认 2K 分辨率
	}

	// 默认回退
	return "1:1", "2K"
}

// GetOpenRouterSupportedModels 返回 OpenRouter 支持的图片生成模型列表
func GetOpenRouterSupportedModels() []string {
	return []string{
		"google/gemini-3-pro-image-preview",
		"google/gemini-2.5-flash-image-preview",
		"black-forest-labs/flux.2-pro",
		"black-forest-labs/flux.2-flex",
		"sourceful/riverflow-v2-standard-preview",
		"sourceful/riverflow-v2-fast",
		"sourceful/riverflow-v2-pro",
	}
}

// GetOpenRouterSupportedAspectRatios 返回 OpenRouter 支持的宽高比列表
func GetOpenRouterSupportedAspectRatios() []string {
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

// GetOpenRouterSupportedImageSizes 返回 OpenRouter 支持的图片尺寸等级
func GetOpenRouterSupportedImageSizes() []string {
	return []string{
		"1K", // 标准分辨率
		"2K", // 较高分辨率（默认）
		"4K", // 最高分辨率
	}
}
