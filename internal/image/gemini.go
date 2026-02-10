package image

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"google.golang.org/genai"
)

// GeminiProvider Google Gemini 图片生成服务提供者
// 直接调用 Google Gemini API，使用官方 Go SDK
type GeminiProvider struct {
	apiKey      string
	model       string
	aspectRatio string
	client      *genai.Client
}

// NewGeminiProvider 创建 Gemini Provider
func NewGeminiProvider(cfg *config.Config) (*GeminiProvider, error) {
	model := cfg.ImageModel
	if model == "" {
		model = "gemini-3-pro-image-preview" // 默认模型
	}

	// 处理宽高比配置
	aspectRatio := mapSizeToGeminiAspectRatio(cfg.ImageSize)

	// 创建 Gemini 客户端
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.ImageAPIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, &GenerateError{
			Provider: "Gemini",
			Code:     "client_error",
			Message:  "创建 Gemini 客户端失败",
			Hint:     "请检查 API Key 是否正确",
			Original: err,
		}
	}

	return &GeminiProvider{
		apiKey:      cfg.ImageAPIKey,
		model:       model,
		aspectRatio: aspectRatio,
		client:      client,
	}, nil
}

// Name 返回提供者名称
func (p *GeminiProvider) Name() string {
	return "Gemini"
}

// Generate 生成图片
func (p *GeminiProvider) Generate(ctx context.Context, prompt string) (*GenerateResult, error) {
	// 构建请求内容
	contents := []*genai.Content{
		{
			Parts: []*genai.Part{
				genai.NewPartFromText(prompt),
			},
			Role: "user",
		},
	}

	// 配置生成参数
	config := &genai.GenerateContentConfig{
		ResponseModalities: []string{"TEXT", "IMAGE"},
	}

	// 如果有宽高比配置，添加到配置中
	// 注意：Gemini API 可能通过不同方式支持宽高比，这里先用基础配置

	// 调用 Gemini API
	resp, err := p.client.Models.GenerateContent(ctx, p.model, contents, config)
	if err != nil {
		return nil, p.handleError(err)
	}

	// 解析响应，提取图片
	filePath, err := p.extractAndSaveImage(resp)
	if err != nil {
		return nil, err
	}

	return &GenerateResult{
		URL:   filePath, // 返回本地文件路径
		Model: p.model,
		Size:  p.aspectRatio,
	}, nil
}

// extractAndSaveImage 从响应中提取图片并保存到临时文件
func (p *GeminiProvider) extractAndSaveImage(resp *genai.GenerateContentResponse) (string, error) {
	if resp == nil || len(resp.Candidates) == 0 {
		return "", &GenerateError{
			Provider: p.Name(),
			Code:     "no_response",
			Message:  "未收到响应",
			Hint:     "请稍后重试",
		}
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return "", &GenerateError{
			Provider: p.Name(),
			Code:     "no_content",
			Message:  "响应中没有内容",
			Hint:     "提示词可能不符合内容政策，请尝试修改提示词",
		}
	}

	// 遍历响应部分，查找图片
	for _, part := range candidate.Content.Parts {
		// 检查是否是内联数据（图片）
		if part.InlineData != nil {
			return p.saveInlineData(part.InlineData)
		}
	}

	return "", &GenerateError{
		Provider: p.Name(),
		Code:     "no_image",
		Message:  "响应中没有图片",
		Hint:     "模型可能只返回了文本，请确保使用支持图片生成的模型",
	}
}

// saveInlineData 保存内联数据到临时文件
func (p *GeminiProvider) saveInlineData(data *genai.Blob) (string, error) {
	if data == nil || len(data.Data) == 0 {
		return "", &GenerateError{
			Provider: p.Name(),
			Code:     "empty_data",
			Message:  "图片数据为空",
		}
	}

	// 确定文件扩展名
	ext := ".png" // 默认 PNG
	mimeType := data.MIMEType
	if strings.Contains(mimeType, "jpeg") || strings.Contains(mimeType, "jpg") {
		ext = ".jpg"
	} else if strings.Contains(mimeType, "gif") {
		ext = ".gif"
	} else if strings.Contains(mimeType, "webp") {
		ext = ".webp"
	}

	// 保存到临时文件
	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("md2wechat_gemini_%d%s", time.Now().UnixNano(), ext))

	// data.Data 已经是解码后的字节，直接写入
	if err := os.WriteFile(tmpPath, data.Data, 0644); err != nil {
		return "", &GenerateError{
			Provider: p.Name(),
			Code:     "write_error",
			Message:  "图片保存失败",
			Original: err,
		}
	}

	return tmpPath, nil
}

// handleError 处理 Gemini API 错误
func (p *GeminiProvider) handleError(err error) error {
	errStr := err.Error()

	// 根据错误信息判断错误类型
	if strings.Contains(errStr, "PERMISSION_DENIED") || strings.Contains(errStr, "401") || strings.Contains(errStr, "403") {
		return &GenerateError{
			Provider: p.Name(),
			Code:     "unauthorized",
			Message:  "Google API Key 无效或权限不足",
			Hint:     "请检查 GOOGLE_API_KEY 或 IMAGE_API_KEY 是否正确，前往 https://aistudio.google.com/apikey 获取",
			Original: err,
		}
	}

	if strings.Contains(errStr, "RESOURCE_EXHAUSTED") || strings.Contains(errStr, "429") {
		return &GenerateError{
			Provider: p.Name(),
			Code:     "rate_limit",
			Message:  "请求过于频繁或配额已用尽",
			Hint:     "请等待一段时间后再试，或检查 Google AI Studio 中的配额使用情况",
			Original: err,
		}
	}

	if strings.Contains(errStr, "INVALID_ARGUMENT") || strings.Contains(errStr, "400") {
		return &GenerateError{
			Provider: p.Name(),
			Code:     "bad_request",
			Message:  "请求参数错误",
			Hint:     "请检查模型名称是否正确。支持的模型: gemini-3-pro-image-preview, gemini-2.5-flash-preview-image",
			Original: err,
		}
	}

	if strings.Contains(errStr, "NOT_FOUND") || strings.Contains(errStr, "404") {
		return &GenerateError{
			Provider: p.Name(),
			Code:     "not_found",
			Message:  "模型不存在",
			Hint:     "请检查模型名称是否正确: gemini-3-pro-image-preview",
			Original: err,
		}
	}

	if strings.Contains(errStr, "SAFETY") || strings.Contains(errStr, "blocked") {
		return &GenerateError{
			Provider: p.Name(),
			Code:     "safety_blocked",
			Message:  "内容被安全过滤器阻止",
			Hint:     "提示词可能包含敏感内容，请修改提示词后重试",
			Original: err,
		}
	}

	// 默认错误
	return &GenerateError{
		Provider: p.Name(),
		Code:     "unknown",
		Message:  fmt.Sprintf("Gemini API 错误: %s", errStr),
		Hint:     "请稍后重试，或查看 https://ai.google.dev/gemini-api/docs 了解更多信息",
		Original: err,
	}
}

// mapSizeToGeminiAspectRatio 将尺寸配置映射到 Gemini 支持的宽高比
func mapSizeToGeminiAspectRatio(size string) string {
	if size == "" {
		return "1:1" // 默认正方形
	}

	// 如果已经是宽高比格式，直接返回
	validRatios := map[string]bool{
		"1:1": true, "16:9": true, "9:16": true,
		"4:3": true, "3:4": true, "3:2": true, "2:3": true,
		"4:5": true, "5:4": true, "21:9": true,
	}
	if validRatios[size] {
		return size
	}

	// Gemini 3 Pro 官方支持的尺寸（按 1K/2K/4K 分组）
	sizeMap := map[string]string{
		// 1:1 正方形
		"1024x1024": "1:1", // 1K
		"2048x2048": "1:1", // 2K
		"4096x4096": "1:1", // 4K
		// 2:3 竖版
		"848x1264":  "2:3", // 1K
		"1696x2528": "2:3", // 2K
		"3392x5056": "2:3", // 4K
		// 3:2 横版
		"1264x848":  "3:2", // 1K
		"2528x1696": "3:2", // 2K
		"5056x3392": "3:2", // 4K
		// 3:4 竖版
		"896x1200":  "3:4", // 1K
		"1792x2400": "3:4", // 2K
		"3584x4800": "3:4", // 4K
		// 4:3 横版
		"1200x896":  "4:3", // 1K
		"2400x1792": "4:3", // 2K
		"4800x3584": "4:3", // 4K
		// 4:5 竖版
		"928x1152":  "4:5", // 1K
		"1856x2304": "4:5", // 2K
		"3712x4608": "4:5", // 4K
		// 5:4 横版
		"1152x928":  "5:4", // 1K
		"2304x1856": "5:4", // 2K
		"4608x3712": "5:4", // 4K
		// 9:16 竖版
		"768x1376":  "9:16", // 1K
		"1536x2752": "9:16", // 2K
		"3072x5504": "9:16", // 4K
		// 16:9 横版
		"1376x768":  "16:9", // 1K
		"2752x1536": "16:9", // 2K
		"5504x3072": "16:9", // 4K
		// 21:9 超宽横版
		"1584x672":  "21:9", // 1K
		"3168x1344": "21:9", // 2K
		"6336x2688": "21:9", // 4K
	}

	if ratio, ok := sizeMap[size]; ok {
		return ratio
	}

	return "1:1" // 默认
}

// Close 关闭客户端连接
func (p *GeminiProvider) Close() error {
	// genai.Client 目前不需要显式关闭
	return nil
}

// GetGeminiSupportedModels 返回 Gemini 支持的图片生成模型列表
func GetGeminiSupportedModels() []string {
	return []string{
		"gemini-3-pro-image-preview",        // Gemini 3 Pro 图片预览版（推荐）
		"gemini-2.5-flash-preview-image",    // Gemini 2.5 Flash 图片版
		"gemini-2.0-flash-exp-image-generation", // Gemini 2.0 Flash 实验版
	}
}

// GetGeminiSupportedAspectRatios 返回 Gemini 支持的宽高比列表
func GetGeminiSupportedAspectRatios() []string {
	return []string{
		"1:1",  // 正方形
		"2:3",  // 竖版照片
		"3:2",  // 横版照片
		"3:4",  // 标准竖版
		"4:3",  // 标准横版
		"4:5",  // 竖版
		"5:4",  // 横版
		"9:16", // 竖版
		"16:9", // 横版
		"21:9", // 超宽横版
	}
}

