package image

import (
	"context"
	"fmt"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
)

// Provider 图片生成服务提供者接口
type Provider interface {
	// Name 返回提供者名称
	Name() string

	// Generate 生成图片，返回图片 URL
	// ctx: 上下文，用于超时控制
	// prompt: 图片生成提示词
	Generate(ctx context.Context, prompt string) (*GenerateResult, error)
}

// GenerateResult 图片生成结果
type GenerateResult struct {
	URL           string // 生成的图片 URL
	RevisedPrompt string // 优化后的提示词（某些提供者会返回）
	Model         string // 实际使用的模型
	Size          string // 实际尺寸
}

// GenerateError 图片生成错误
type GenerateError struct {
	Provider string // 提供者名称
	Code     string // 错误码
	Message  string // 用户友好的错误信息
	Hint     string // 解决提示
	Original error  // 原始错误
}

func (e *GenerateError) Error() string {
	msg := fmt.Sprintf("[%s] %s", e.Provider, e.Message)
	if e.Hint != "" {
		msg += fmt.Sprintf("\n提示: %s", e.Hint)
	}
	return msg
}

func (e *GenerateError) Unwrap() error {
	return e.Original
}

// NewProvider 根据配置创建对应的 Provider
func NewProvider(cfg *config.Config) (Provider, error) {
	switch cfg.ImageProvider {
	case "tuzi":
		if err := validateTuZiConfig(cfg); err != nil {
			return nil, err
		}
		return NewTuZiProvider(cfg)
	case "modelscope", "ms":
		if err := validateModelScopeConfig(cfg); err != nil {
			return nil, err
		}
		return NewModelScopeProvider(cfg)
	case "openrouter", "or":
		if err := validateOpenRouterConfig(cfg); err != nil {
			return nil, err
		}
		return NewOpenRouterProvider(cfg)
	case "gemini", "google":
		if err := validateGeminiConfig(cfg); err != nil {
			return nil, err
		}
		return NewGeminiProvider(cfg)
	case "openai", "":
		if err := validateOpenAIConfig(cfg); err != nil {
			return nil, err
		}
		return NewOpenAIProvider(cfg)
	default:
		return nil, &config.ConfigError{
			Field:   "ImageProvider",
			Message: fmt.Sprintf("未知的图片服务提供者: %s", cfg.ImageProvider),
			Hint:    "支持的提供者: openai, tuzi, modelscope (或 ms), openrouter (或 or), gemini (或 google)",
		}
	}
}

// validateOpenAIConfig 验证 OpenAI 配置
func validateOpenAIConfig(cfg *config.Config) error {
	if cfg.ImageAPIKey == "" {
		return &config.ConfigError{
			Field:   "ImageAPIKey",
			Message: "使用 OpenAI 图片服务需要配置 API Key",
			Hint:    "在配置文件中设置 api.image_key 或环境变量 IMAGE_API_KEY",
		}
	}
	if cfg.ImageAPIBase == "" {
		return &config.ConfigError{
			Field:   "ImageAPIBase",
			Message: "需要配置 API Base URL",
			Hint:    "在配置文件中设置 api.image_base_url 或使用默认值",
		}
	}
	return nil
}

// validateTuZiConfig 验证 TuZi 配置
func validateTuZiConfig(cfg *config.Config) error {
	if cfg.ImageAPIKey == "" {
		return &config.ConfigError{
			Field:   "ImageAPIKey",
			Message: "使用 TuZi 图片服务需要配置 API Key",
			Hint:    "在配置文件中设置 api.image_key 或环境变量 IMAGE_API_KEY",
		}
	}
	if cfg.ImageAPIBase == "" {
		return &config.ConfigError{
			Field:   "ImageAPIBase",
			Message: "需要配置 TuZi API Base URL",
			Hint:    "在配置文件中设置 api.image_base_url，通常为 https://api.tu-zi.com/v1",
		}
	}
	return nil
}

// validateModelScopeConfig 验证 ModelScope 配置
func validateModelScopeConfig(cfg *config.Config) error {
	if cfg.ImageAPIKey == "" {
		return &config.ConfigError{
			Field:   "ImageAPIKey",
			Message: "使用 ModelScope 图片服务需要配置 API Key",
			Hint:    "在配置文件中设置 api.image_key 或环境变量 IMAGE_API_KEY，前往 https://modelscope.cn/my/myaccesstoken 获取",
		}
	}
	// ModelScope API Base 有默认值，可以为空
	return nil
}

// validateOpenRouterConfig 验证 OpenRouter 配置
func validateOpenRouterConfig(cfg *config.Config) error {
	if cfg.ImageAPIKey == "" {
		return &config.ConfigError{
			Field:   "ImageAPIKey",
			Message: "使用 OpenRouter 图片服务需要配置 API Key",
			Hint:    "在配置文件中设置 api.image_key 或环境变量 IMAGE_API_KEY，前往 openrouter.ai 获取",
		}
	}
	// OpenRouter API Base 有默认值，可以为空
	return nil
}

// validateGeminiConfig 验证 Google Gemini 配置
func validateGeminiConfig(cfg *config.Config) error {
	if cfg.ImageAPIKey == "" {
		return &config.ConfigError{
			Field:   "ImageAPIKey",
			Message: "使用 Google Gemini 图片服务需要配置 API Key",
			Hint:    "在配置文件中设置 api.image_key 或环境变量 IMAGE_API_KEY (或 GOOGLE_API_KEY)，前往 https://aistudio.google.com/apikey 获取",
		}
	}
	return nil
}
