package image

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
)

type ProviderMeta struct {
	Name           string   `json:"name"`
	Aliases        []string `json:"aliases,omitempty"`
	Description    string   `json:"description"`
	RequiredConfig []string `json:"required_config,omitempty"`
	OptionalConfig []string `json:"optional_config,omitempty"`
	DefaultBaseURL string   `json:"default_base_url,omitempty"`
	DefaultModel   string   `json:"default_model,omitempty"`
	SupportsSize   bool     `json:"supports_size"`
}

var providerRegistry = []ProviderMeta{
	{
		Name:           "openai",
		Description:    "OpenAI-compatible image generation provider",
		RequiredConfig: []string{"IMAGE_API_KEY"},
		OptionalConfig: []string{"IMAGE_API_BASE", "IMAGE_MODEL", "IMAGE_SIZE"},
		DefaultBaseURL: "https://api.openai.com/v1",
		DefaultModel:   "gpt-image-1.5",
		SupportsSize:   true,
	},
	{
		Name:           "bailian",
		Aliases:        []string{"aliyun"},
		Description:    "Aliyun Bailian (DashScope) image generation provider",
		RequiredConfig: []string{"IMAGE_API_KEY"},
		OptionalConfig: []string{"IMAGE_API_BASE", "IMAGE_MODEL", "IMAGE_SIZE"},
		DefaultBaseURL: "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation",
		DefaultModel:   "qwen-image-2.0",
		SupportsSize:   true,
	},
	{
		Name:           "tuzi",
		Description:    "TuZi image generation provider",
		RequiredConfig: []string{"IMAGE_API_KEY", "IMAGE_API_BASE"},
		OptionalConfig: []string{"IMAGE_MODEL", "IMAGE_SIZE"},
		DefaultModel:   "gpt-image-1",
		SupportsSize:   true,
	},
	{
		Name:           "ark",
		Aliases:        []string{"volcengine", "volces"},
		Description:    "Volcengine Ark image generation provider",
		RequiredConfig: []string{"IMAGE_API_KEY"},
		OptionalConfig: []string{"IMAGE_API_BASE", "IMAGE_MODEL", "IMAGE_SIZE"},
		DefaultBaseURL: "https://ark.cn-beijing.volces.com/api/v3",
		DefaultModel:   "doubao-seedream-5.0-lite",
		SupportsSize:   true,
	},
	{
		Name:           "modelscope",
		Aliases:        []string{"ms"},
		Description:    "ModelScope image generation provider",
		RequiredConfig: []string{"IMAGE_API_KEY"},
		OptionalConfig: []string{"IMAGE_API_BASE", "IMAGE_MODEL", "IMAGE_SIZE"},
		DefaultBaseURL: "https://api-inference.modelscope.cn",
		DefaultModel:   "Tongyi-MAI/Z-Image-Turbo",
		SupportsSize:   true,
	},
	{
		Name:           "openrouter",
		Aliases:        []string{"or"},
		Description:    "OpenRouter image generation provider",
		RequiredConfig: []string{"IMAGE_API_KEY"},
		OptionalConfig: []string{"IMAGE_API_BASE", "IMAGE_MODEL", "IMAGE_SIZE"},
		DefaultBaseURL: "https://openrouter.ai/api/v1",
		DefaultModel:   "google/gemini-3-pro-image-preview",
		SupportsSize:   true,
	},
	{
		Name:           "gemini",
		Aliases:        []string{"google"},
		Description:    "Google Gemini image generation provider",
		RequiredConfig: []string{"IMAGE_API_KEY"},
		OptionalConfig: []string{"IMAGE_MODEL", "IMAGE_SIZE"},
		DefaultModel:   "gemini-3.1-flash-image-preview",
		SupportsSize:   true,
	},
}

func SupportedProviders() []ProviderMeta {
	result := make([]ProviderMeta, len(providerRegistry))
	copy(result, providerRegistry)
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func LookupProviderMeta(name string) (ProviderMeta, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, meta := range providerRegistry {
		if meta.Name == name {
			return meta, true
		}
		for _, alias := range meta.Aliases {
			if alias == name {
				return meta, true
			}
		}
	}
	return ProviderMeta{}, false
}

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
	case "bailian", "aliyun":
		if err := validateBailianConfig(cfg); err != nil {
			return nil, err
		}
		return NewBailianProvider(cfg)
	case "ark", "volcengine", "volces":
		if err := validateArkConfig(cfg); err != nil {
			return nil, err
		}
		return NewArkProvider(cfg)
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
			Hint:    "支持的提供者: openai, ark (或 volcengine / volces), tuzi, modelscope (或 ms), openrouter (或 or), gemini (或 google), bailian (或 aliyun)",
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

// validateArkConfig 验证 Ark 配置
func validateArkConfig(cfg *config.Config) error {
	if cfg.ImageAPIKey == "" {
		return &config.ConfigError{
			Field:   "ImageAPIKey",
			Message: "使用 Ark 图片服务需要配置 API Key",
			Hint:    "在配置文件中设置 api.image_key 或环境变量 IMAGE_API_KEY，前往火山引擎 Ark 控制台获取",
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

// validateBailianConfig 验证 Bailian 配置
func validateBailianConfig(cfg *config.Config) error {
	if cfg.ImageAPIKey == "" {
		return &config.ConfigError{
			Field:   "ImageAPIKey",
			Message: "使用 bailian provider 时，API Key 不能为空",
			Hint:    "请在配置文件中设置 api.image_key，或通过环境变量 IMAGE_API_KEY 设置",
		}
	}
	return nil
}
