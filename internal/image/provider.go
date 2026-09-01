package image

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
)

type ProviderMeta struct {
	Name                     string              `json:"name"`
	Aliases                  []string            `json:"aliases,omitempty"`
	Description              string              `json:"description"`
	RequiredConfig           []string            `json:"required_config,omitempty"`
	OptionalConfig           []string            `json:"optional_config,omitempty"`
	DefaultBaseURL           string              `json:"default_base_url,omitempty"`
	DefaultModel             string              `json:"default_model,omitempty"`
	SupportedModels          []ProviderModelMeta `json:"supported_models,omitempty"`
	SupportsSize             bool                `json:"supports_size"`
	SupportsSubjectReference bool                `json:"supports_subject_reference,omitempty"`
}

type ProviderModelMeta struct {
	Name                     string `json:"name"`
	Description              string `json:"description,omitempty"`
	Default                  bool   `json:"default,omitempty"`
	SupportsSubjectReference bool   `json:"supports_subject_reference,omitempty"`
}

var providerRegistry = []ProviderMeta{
	{
		Name:           "minimax",
		Description:    "MiniMax image generation provider",
		RequiredConfig: []string{"IMAGE_API_KEY"},
		OptionalConfig: []string{"IMAGE_API_BASE", "IMAGE_MODEL", "IMAGE_SIZE"},
		DefaultBaseURL: "https://api.minimax.io",
		DefaultModel:   "image-01",
		SupportedModels: []ProviderModelMeta{
			{Name: "image-01", Description: "MiniMax image model", Default: true, SupportsSubjectReference: true},
			{Name: "image-01-live", Description: "MiniMax live image model"},
		},
		SupportsSize:             true,
		SupportsSubjectReference: true,
	},
	{
		Name:           "openai",
		Description:    "OpenAI-compatible image generation provider",
		RequiredConfig: []string{"IMAGE_API_KEY"},
		OptionalConfig: []string{"IMAGE_API_BASE", "IMAGE_MODEL", "IMAGE_SIZE"},
		DefaultBaseURL: "https://api.openai.com/v1",
		DefaultModel:   "gpt-image-2",
		SupportedModels: []ProviderModelMeta{
			{Name: "gpt-image-2", Description: "OpenAI current primary image model", Default: true},
			{Name: "gpt-image-1.5", Description: "Previous state-of-the-art OpenAI image model"},
			{Name: "gpt-image-1", Description: "Previous generation general image model"},
			{Name: "gpt-image-1-mini", Description: "Lower-cost OpenAI image model"},
			{Name: "dall-e-3", Description: "Legacy compatibility model"},
			{Name: "dall-e-2", Description: "Legacy compatibility model"},
		},
		SupportsSize: true,
	},
	{
		Name:           "tuzi",
		Description:    "TuZi image generation provider",
		RequiredConfig: []string{"IMAGE_API_KEY", "IMAGE_API_BASE"},
		OptionalConfig: []string{"IMAGE_MODEL", "IMAGE_SIZE"},
		DefaultModel:   "doubao-seedream-4-5-251128",
		SupportedModels: []ProviderModelMeta{
			{Name: "doubao-seedream-4-5-251128", Description: "Doubao Seedream 4.5", Default: true},
			{Name: "gemini-3-pro-image-preview", Description: "Gemini 3 Pro image preview"},
		},
		SupportsSize: true,
	},
	{
		Name:           "modelscope",
		Aliases:        []string{"ms"},
		Description:    "ModelScope image generation provider",
		RequiredConfig: []string{"IMAGE_API_KEY"},
		OptionalConfig: []string{"IMAGE_API_BASE", "IMAGE_MODEL", "IMAGE_SIZE"},
		DefaultBaseURL: "https://api-inference.modelscope.cn",
		DefaultModel:   "Tongyi-MAI/Z-Image-Turbo",
		SupportedModels: []ProviderModelMeta{
			{Name: "Tongyi-MAI/Z-Image-Turbo", Description: "Default verified ModelScope image model", Default: true},
		},
		SupportsSize: true,
	},
	{
		Name:           "openrouter",
		Aliases:        []string{"or"},
		Description:    "OpenRouter image generation provider",
		RequiredConfig: []string{"IMAGE_API_KEY"},
		OptionalConfig: []string{"IMAGE_API_BASE", "IMAGE_MODEL", "IMAGE_SIZE"},
		DefaultBaseURL: "https://openrouter.ai/api/v1",
		DefaultModel:   "google/gemini-3-pro-image-preview",
		SupportedModels: []ProviderModelMeta{
			{Name: "google/gemini-3-pro-image-preview", Description: "Gemini 3 Pro via OpenRouter", Default: true},
			{Name: "google/gemini-2.5-flash-image-preview", Description: "Gemini 2.5 Flash via OpenRouter"},
			{Name: "black-forest-labs/flux.2-pro", Description: "Flux 2 Pro"},
			{Name: "black-forest-labs/flux.2-flex", Description: "Flux 2 Flex"},
			{Name: "sourceful/riverflow-v2-standard-preview", Description: "Riverflow v2 standard"},
			{Name: "sourceful/riverflow-v2-fast", Description: "Riverflow v2 fast"},
			{Name: "sourceful/riverflow-v2-pro", Description: "Riverflow v2 pro"},
		},
		SupportsSize: true,
	},
	{
		Name:           "gemini",
		Aliases:        []string{"google"},
		Description:    "Google Gemini image generation provider",
		RequiredConfig: []string{"IMAGE_API_KEY"},
		OptionalConfig: []string{"IMAGE_MODEL", "IMAGE_SIZE"},
		DefaultModel:   "gemini-3.1-flash-image-preview",
		SupportedModels: []ProviderModelMeta{
			{Name: "gemini-3.1-flash-image-preview", Description: "Gemini 3.1 Flash image preview", Default: true},
			{Name: "gemini-3-pro-image-preview", Description: "Gemini 3 Pro image preview"},
			{Name: "gemini-2.5-flash-image", Description: "Gemini 2.5 Flash image model"},
			{Name: "gemini-2.5-flash-preview-image", Description: "Gemini 2.5 Flash preview compatibility name"},
			{Name: "gemini-2.0-flash-exp-image-generation", Description: "Gemini 2.0 Flash experimental image generation"},
		},
		SupportsSize: true,
	},
	{
		Name:           "volcengine",
		Aliases:        []string{"volc"},
		Description:    "Volcengine Ark image generation provider",
		RequiredConfig: []string{"IMAGE_API_KEY"},
		OptionalConfig: []string{"IMAGE_API_BASE", "IMAGE_MODEL", "IMAGE_SIZE"},
		DefaultBaseURL: "https://ark.cn-beijing.volces.com/api/v3",
		DefaultModel:   "doubao-seedream-5-0-pro-260628",
		SupportedModels: []ProviderModelMeta{
			{Name: "doubao-seedream-5-0-pro-260628", Description: "Doubao Seedream 5.0 Pro", Default: true},
			{Name: "doubao-seedream-5-0-260128", Description: "Doubao Seedream 5.0"},
			{Name: "doubao-seedream-5-0-lite-260128", Description: "Doubao Seedream 5.0 Lite"},
		},
		SupportsSize: true,
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

func DefaultProviderModel(name string) string {
	meta, ok := LookupProviderMeta(name)
	if !ok {
		return ""
	}
	return meta.DefaultModel
}

func DefaultProviderBaseURL(name string) string {
	meta, ok := LookupProviderMeta(name)
	if !ok {
		return ""
	}
	return meta.DefaultBaseURL
}

func ProviderSupportedModels(name string) []ProviderModelMeta {
	meta, ok := LookupProviderMeta(name)
	if !ok {
		return nil
	}
	result := make([]ProviderModelMeta, len(meta.SupportedModels))
	copy(result, meta.SupportedModels)
	return result
}

func ProviderSupportedModelNames(name string) []string {
	models := ProviderSupportedModels(name)
	result := make([]string, 0, len(models))
	for _, model := range models {
		result = append(result, model.Name)
	}
	return result
}

// ProviderSupportsSubjectReference reports whether the provider accepts a
// subject reference for image-to-image generation.
func ProviderSupportsSubjectReference(name string) bool {
	meta, ok := LookupProviderMeta(name)
	if !ok {
		return false
	}
	return meta.SupportsSubjectReference
}

// SubjectReferenceProviderNames lists the providers that accept a subject
// reference for image-to-image generation.
func SubjectReferenceProviderNames() []string {
	result := make([]string, 0, len(providerRegistry))
	for _, meta := range SupportedProviders() {
		if meta.SupportsSubjectReference {
			result = append(result, meta.Name)
		}
	}
	return result
}

// SubjectReferenceModelNames lists the provider models that accept a subject
// reference.
func SubjectReferenceModelNames(provider string) []string {
	meta, ok := LookupProviderMeta(provider)
	if !ok || !meta.SupportsSubjectReference {
		return nil
	}
	result := make([]string, 0, len(meta.SupportedModels))
	for _, model := range meta.SupportedModels {
		if model.SupportsSubjectReference {
			result = append(result, model.Name)
		}
	}
	return result
}

// ModelSupportsSubjectReference reports whether the provider and model pair
// accepts a subject reference. An empty model resolves to the provider default.
func ModelSupportsSubjectReference(provider, model string) bool {
	meta, ok := LookupProviderMeta(provider)
	if !ok || !meta.SupportsSubjectReference {
		return false
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = meta.DefaultModel
	}
	for _, candidate := range meta.SupportedModels {
		if strings.EqualFold(candidate.Name, model) {
			return candidate.SupportsSubjectReference
		}
	}
	return false
}

// ValidateSubjectReferenceURL checks that a subject reference is a public
// HTTP(S) image URL, which is the only form the current contract supports.
func ValidateSubjectReferenceURL(reference string) error {
	trimmed := strings.TrimSpace(reference)
	if trimmed == "" {
		return fmt.Errorf("subject reference is required")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("subject reference must be a valid URL: %w", err)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if (scheme != "http" && scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("subject reference must be a public http:// or https:// image URL; inline data URLs and local paths are not supported")
	}
	return nil
}

// SubjectReferenceModelsHint describes which models accept a subject reference.
func SubjectReferenceModelsHint(provider string) string {
	models := SubjectReferenceModelNames(provider)
	if len(models) == 0 {
		return ""
	}
	return "models supporting subject references: " + strings.Join(models, ", ")
}

func ProviderSupportedModelsHint(name string) string {
	models := ProviderSupportedModelNames(name)
	if len(models) == 0 {
		return ""
	}
	return "支持的模型: " + strings.Join(models, ", ")
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

// SubjectReferenceProvider supports image generation guided by a portrait reference.
type SubjectReferenceProvider interface {
	Provider
	GenerateWithSubject(ctx context.Context, prompt, subjectReference string) (*GenerateResult, error)
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
	providerName := strings.TrimSpace(cfg.ImageProvider)
	if providerName == "" {
		providerName = "openai"
	}
	if meta, ok := LookupProviderMeta(providerName); ok {
		providerName = meta.Name
	}

	switch providerName {
	case "minimax":
		if err := validateMiniMaxConfig(cfg); err != nil {
			return nil, err
		}
		return NewMiniMaxProvider(cfg)
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
	case "volcengine", "volc":
		if err := validateVolcengineConfig(cfg); err != nil {
			return nil, err
		}
		return NewVolcengineProvider(cfg)
	case "openai", "":
		if err := validateOpenAIConfig(cfg); err != nil {
			return nil, err
		}
		return NewOpenAIProvider(cfg)
	default:
		return nil, &config.ConfigError{
			Field:   "ImageProvider",
			Message: fmt.Sprintf("未知的图片服务提供者: %s", cfg.ImageProvider),
			Hint:    "支持的提供者: openai, minimax, tuzi, modelscope (或 ms), openrouter (或 or), gemini (或 google), volcengine (或 volc)",
		}
	}
}

func validateMiniMaxConfig(cfg *config.Config) error {
	if cfg.ImageAPIKey == "" {
		return &config.ConfigError{
			Field:   "ImageAPIKey",
			Message: "MiniMax image generation requires an API key",
			Hint:    "Set api.image_key in the config file or use IMAGE_API_KEY",
		}
	}
	return nil
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
			Hint:    "在配置文件中设置 api.image_key 或环境变量 IMAGE_API_KEY，前往 https://aistudio.google.com/apikey 获取",
		}
	}
	return nil
}

func validateVolcengineConfig(cfg *config.Config) error {
	if cfg.ImageAPIKey == "" {
		return &config.ConfigError{
			Field:   "ImageAPIKey",
			Message: "使用火山引擎图片服务需要配置 API Key",
			Hint:    "在配置文件中设置 api.image_key 或环境变量 IMAGE_API_KEY，前往火山引擎 Ark 控制台获取",
		}
	}
	return nil
}
