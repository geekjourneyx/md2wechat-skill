package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	// 微信公众号配置
	WechatAppID  string `json:"wechat_appid" yaml:"wechat_appid" env:"WECHAT_APPID"`
	WechatSecret string `json:"wechat_secret" yaml:"wechat_secret" env:"WECHAT_SECRET"`

	// md2wechat.cn API 配置
	MD2WechatAPIKey     string `json:"md2wechat_api_key" yaml:"md2wechat_api_key" env:"MD2WECHAT_API_KEY"`
	MD2WechatBaseURL    string `json:"md2wechat_base_url" yaml:"md2wechat_base_url" env:"MD2WECHAT_BASE_URL"`
	DefaultConvertMode  string `json:"default_convert_mode" yaml:"default_convert_mode" env:"CONVERT_MODE"`
	DefaultTheme        string `json:"default_theme" yaml:"default_theme" env:"DEFAULT_THEME"`
	DefaultBackgroundType string `json:"default_background_type" yaml:"default_background_type" env:"DEFAULT_BACKGROUND_TYPE"` // default/grid/none

	// 图片生成 API 配置
	ImageProvider string `json:"image_provider" yaml:"image_provider" env:"IMAGE_PROVIDER"`
	ImageAPIKey   string `json:"image_api_key" yaml:"image_api_key" env:"IMAGE_API_KEY"`
	ImageAPIBase  string `json:"image_api_base" yaml:"image_api_base" env:"IMAGE_API_BASE"`
	ImageModel    string `json:"image_model" yaml:"image_model" env:"IMAGE_MODEL"`
	ImageSize     string `json:"image_size" yaml:"image_size" env:"IMAGE_SIZE"`

	// 图片处理配置
	CompressImages bool  `json:"compress_images" yaml:"compress_images" env:"COMPRESS_IMAGES"`
	MaxImageWidth  int   `json:"max_image_width" yaml:"max_image_width" env:"MAX_IMAGE_WIDTH"`
	MaxImageSize   int64 `json:"max_image_size" yaml:"max_image_size" env:"MAX_IMAGE_SIZE"`

	// 超时配置
	HTTPTimeout int `json:"http_timeout" yaml:"http_timeout" env:"HTTP_TIMEOUT"`

	// 配置文件路径（用于追踪）
	configFile string
}

// ConfigFile 配置文件结构（YAML/JSON）
type configFile struct {
	Wechat struct {
		AppID  string `json:"appid" yaml:"appid"`
		Secret string `json:"secret" yaml:"secret"`
	} `json:"wechat" yaml:"wechat"`

	API struct {
		MD2WechatKey       string `json:"md2wechat_key" yaml:"md2wechat_key"`
		MD2WechatBaseURL  string `json:"md2wechat_base_url" yaml:"md2wechat_base_url"`
		ImageKey          string `json:"image_key" yaml:"image_key"`
		ImageBaseURL      string `json:"image_base_url" yaml:"image_base_url"`
		ImageProvider     string `json:"image_provider" yaml:"image_provider"`
		ImageModel        string `json:"image_model" yaml:"image_model"`
		ImageSize         string `json:"image_size" yaml:"image_size"`
		ConvertMode       string `json:"convert_mode" yaml:"convert_mode"`
		DefaultTheme      string `json:"default_theme" yaml:"default_theme"`
		BackgroundType    string `json:"background_type" yaml:"background_type"`
		HTTPTimeout       int    `json:"http_timeout" yaml:"http_timeout"`
	} `json:"api" yaml:"api"`

	Image struct {
		Compress bool `json:"compress" yaml:"compress"`
		MaxWidth int  `json:"max_width" yaml:"max_width"`
		MaxSize  int  `json:"max_size_mb" yaml:"max_size_mb"`
	} `json:"image" yaml:"image"`
}

// Load 从配置文件和环境变量加载配置
// 优先级：环境变量 > 配置文件 > 默认值
func Load() (*Config, error) {
	return LoadWithDefaults("")
}

// LoadWithDefaults 使用指定配置文件路径加载配置
func LoadWithDefaults(configPath string) (*Config, error) {
	cfg := &Config{
		DefaultConvertMode:  "api",
		DefaultTheme:        "default",
		DefaultBackgroundType: "default",
		MD2WechatBaseURL:   "https://www.md2wechat.cn",
		CompressImages:     true,
		MaxImageWidth:      1920,
		MaxImageSize:       5 * 1024 * 1024, // 5MB
		HTTPTimeout:        30,
		ImageProvider:      "openai",
		ImageAPIBase:       "https://api.openai.com/v1",
		ImageModel:         "dall-e-3",
		ImageSize:          "1024x1024",
	}

	// 1. 尝试从配置文件加载
	if configPath == "" {
		configPath = findConfigFile()
	}
	if configPath != "" {
		if err := loadFromFile(cfg, configPath); err != nil {
			// 配置文件加载失败不是致命错误，继续使用环境变量和默认值
			fmt.Fprintf(os.Stderr, "⚠️  警告: 配置文件加载失败 (%v)，将使用环境变量或默认值\n", err)
		} else {
			cfg.configFile = configPath
			// 显示正在使用的配置文件
			relPath := getRelativePath(configPath)
			fmt.Fprintf(os.Stderr, "✅ 使用配置文件: %s\n", relPath)
		}
	}

	// 2. 环境变量覆盖配置文件
	loadFromEnv(cfg)

	// 3. 验证必需配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// 4. 处理 MaxImageSize (配置文件中是 MB)
	if cfg.configFile != "" && cfg.MaxImageSize < 1024*1024 {
		// 如果值小于 1MB，可能是配置文件使用了 MB 单位
		cfg.MaxImageSize = cfg.MaxImageSize * 1024 * 1024
	}

	return cfg, nil
}

// findConfigFile 查找配置文件
// 优先级：用户目录（全局配置） > 当前目录（项目配置）
func findConfigFile() string {
	// 优先使用用户主目录的配置文件（全局配置，一次配置所有项目通用）
	homeDir, _ := os.UserHomeDir()
	userPaths := []string{
		filepath.Join(homeDir, ".config", "md2wechat", "config.yaml"),
		filepath.Join(homeDir, ".md2wechat.yaml"),
		filepath.Join(homeDir, ".md2wechat.yml"),
	}

	// 当前工作目录的配置文件（项目级配置，可选）
	cwdPaths := []string{
		"md2wechat.yaml",
		"md2wechat.yml",
		"md2wechat.json",
		".md2wechat.yaml",
		".md2wechat.yml",
		".md2wechat.json",
	}

	// 先查找用户目录配置
	for _, path := range userPaths {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}

	// 再查找当前目录配置
	for _, path := range cwdPaths {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}

	return ""
}

// loadFromFile 从文件加载配置
func loadFromFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))

	if ext == ".json" {
		return loadFromJSON(cfg, data)
	}
	// 默认使用 YAML
	return loadFromYAML(cfg, data)
}

// loadFromYAML 从 YAML 加载
func loadFromYAML(cfg *Config, data []byte) error {
	var cf configFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return fmt.Errorf("parse yaml: %w", err)
	}

	// 映射到 Config
	if cf.Wechat.AppID != "" {
		cfg.WechatAppID = cf.Wechat.AppID
	}
	if cf.Wechat.Secret != "" {
		cfg.WechatSecret = cf.Wechat.Secret
	}
	if cf.API.MD2WechatKey != "" {
		cfg.MD2WechatAPIKey = cf.API.MD2WechatKey
	}
	if cf.API.MD2WechatBaseURL != "" {
		cfg.MD2WechatBaseURL = cf.API.MD2WechatBaseURL
	}
	if cf.API.ImageKey != "" {
		cfg.ImageAPIKey = cf.API.ImageKey
	}
	if cf.API.ImageBaseURL != "" {
		cfg.ImageAPIBase = cf.API.ImageBaseURL
	}
	if cf.API.ImageProvider != "" {
		cfg.ImageProvider = cf.API.ImageProvider
	}
	if cf.API.ImageModel != "" {
		cfg.ImageModel = cf.API.ImageModel
	}
	if cf.API.ImageSize != "" {
		cfg.ImageSize = cf.API.ImageSize
	}
	if cf.API.ConvertMode != "" {
		cfg.DefaultConvertMode = cf.API.ConvertMode
	}
	if cf.API.DefaultTheme != "" {
		cfg.DefaultTheme = cf.API.DefaultTheme
	}
	if cf.API.BackgroundType != "" {
		cfg.DefaultBackgroundType = cf.API.BackgroundType
	}
	if cf.API.HTTPTimeout > 0 {
		cfg.HTTPTimeout = cf.API.HTTPTimeout
	}
	cfg.CompressImages = cf.Image.Compress
	if cf.Image.MaxWidth > 0 {
		cfg.MaxImageWidth = cf.Image.MaxWidth
	}
	if cf.Image.MaxSize > 0 {
		cfg.MaxImageSize = int64(cf.Image.MaxSize) * 1024 * 1024
	}

	return nil
}

// loadFromJSON 从 JSON 加载
func loadFromJSON(cfg *Config, data []byte) error {
	var cf configFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return fmt.Errorf("parse json: %w", err)
	}

	// 映射到 Config（与 loadFromYAML 相同的逻辑）
	if cf.Wechat.AppID != "" {
		cfg.WechatAppID = cf.Wechat.AppID
	}
	if cf.Wechat.Secret != "" {
		cfg.WechatSecret = cf.Wechat.Secret
	}
	if cf.API.MD2WechatKey != "" {
		cfg.MD2WechatAPIKey = cf.API.MD2WechatKey
	}
	if cf.API.MD2WechatBaseURL != "" {
		cfg.MD2WechatBaseURL = cf.API.MD2WechatBaseURL
	}
	if cf.API.ImageKey != "" {
		cfg.ImageAPIKey = cf.API.ImageKey
	}
	if cf.API.ImageBaseURL != "" {
		cfg.ImageAPIBase = cf.API.ImageBaseURL
	}
	if cf.API.ImageProvider != "" {
		cfg.ImageProvider = cf.API.ImageProvider
	}
	if cf.API.ImageModel != "" {
		cfg.ImageModel = cf.API.ImageModel
	}
	if cf.API.ImageSize != "" {
		cfg.ImageSize = cf.API.ImageSize
	}
	if cf.API.ConvertMode != "" {
		cfg.DefaultConvertMode = cf.API.ConvertMode
	}
	if cf.API.DefaultTheme != "" {
		cfg.DefaultTheme = cf.API.DefaultTheme
	}
	if cf.API.BackgroundType != "" {
		cfg.DefaultBackgroundType = cf.API.BackgroundType
	}
	if cf.API.HTTPTimeout > 0 {
		cfg.HTTPTimeout = cf.API.HTTPTimeout
	}
	cfg.CompressImages = cf.Image.Compress
	if cf.Image.MaxWidth > 0 {
		cfg.MaxImageWidth = cf.Image.MaxWidth
	}
	if cf.Image.MaxSize > 0 {
		cfg.MaxImageSize = int64(cf.Image.MaxSize) * 1024 * 1024
	}

	return nil
}

// loadFromEnv 从环境变量加载
func loadFromEnv(cfg *Config) {
	if v := os.Getenv("WECHAT_APPID"); v != "" {
		cfg.WechatAppID = v
	}
	if v := os.Getenv("WECHAT_SECRET"); v != "" {
		cfg.WechatSecret = v
	}
	if v := os.Getenv("MD2WECHAT_API_KEY"); v != "" {
		cfg.MD2WechatAPIKey = v
	}
	if v := os.Getenv("MD2WECHAT_BASE_URL"); v != "" {
		cfg.MD2WechatBaseURL = v
	}
	if v := os.Getenv("CONVERT_MODE"); v != "" {
		cfg.DefaultConvertMode = v
	}
	if v := os.Getenv("DEFAULT_THEME"); v != "" {
		cfg.DefaultTheme = v
	}
	if v := os.Getenv("DEFAULT_BACKGROUND_TYPE"); v != "" {
		cfg.DefaultBackgroundType = v
	}
	if v := os.Getenv("IMAGE_API_KEY"); v != "" {
		cfg.ImageAPIKey = v
	}
	if v := os.Getenv("IMAGE_API_BASE"); v != "" {
		cfg.ImageAPIBase = v
	}
	if v := os.Getenv("IMAGE_PROVIDER"); v != "" {
		cfg.ImageProvider = v
	}
	if v := os.Getenv("IMAGE_MODEL"); v != "" {
		cfg.ImageModel = v
	}
	if v := os.Getenv("IMAGE_SIZE"); v != "" {
		cfg.ImageSize = v
	}
	if v := os.Getenv("COMPRESS_IMAGES"); v != "" {
		cfg.CompressImages = getEnvBool("COMPRESS_IMAGES", true)
	}
	if v := os.Getenv("MAX_IMAGE_WIDTH"); v != "" {
		cfg.MaxImageWidth = getEnvInt("MAX_IMAGE_WIDTH", cfg.MaxImageWidth)
	}
	if v := os.Getenv("MAX_IMAGE_SIZE"); v != "" {
		cfg.MaxImageSize = int64(getEnvInt("MAX_IMAGE_SIZE", int(cfg.MaxImageSize)))
	}
	if v := os.Getenv("HTTP_TIMEOUT"); v != "" {
		cfg.HTTPTimeout = getEnvInt("HTTP_TIMEOUT", cfg.HTTPTimeout)
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.WechatAppID == "" {
		return &ConfigError{
			Field:   "WechatAppID",
			Message: "微信公众号 AppID 未配置",
			Hint:    "运行 'md2wechat config init' 生成配置文件，然后填入 AppID",
		}
	}
	if c.WechatSecret == "" {
		return &ConfigError{
			Field:   "WechatSecret",
			Message: "微信公众号 Secret 未配置",
			Hint:    "登录微信公众平台 > 设置与开发 > 基本配置 > 获取 Secret",
		}
	}

	// 验证转换模式
	if c.DefaultConvertMode != "api" && c.DefaultConvertMode != "ai" {
		return &ConfigError{
			Field:   "ConvertMode",
			Message: "转换模式必须是 'api' 或 'ai'",
			Hint:    "配置文件中设置 api.convert_mode: api",
		}
	}

	// 验证数值范围
	if c.MaxImageWidth < 100 || c.MaxImageWidth > 10000 {
		return &ConfigError{
			Field:   "MaxImageWidth",
			Message: "图片最大宽度必须在 100 到 10000 之间",
			Hint:    "配置文件中设置 image.max_width: 1920",
		}
	}
	if c.MaxImageSize < 1024*100 { // 最小 100KB
		return &ConfigError{
			Field:   "MaxImageSize",
			Message: "图片最大大小不能小于 100KB",
			Hint:    "配置文件中设置 image.max_size_mb: 5",
		}
	}
	if c.HTTPTimeout < 1 || c.HTTPTimeout > 300 {
		return &ConfigError{
			Field:   "HTTPTimeout",
			Message: "超时时间必须在 1 到 300 秒之间",
			Hint:    "配置文件中设置 api.http_timeout: 30",
		}
	}

	return nil
}

// ValidateForImageGeneration 验证图片生成配置
func (c *Config) ValidateForImageGeneration() error {
	if c.ImageAPIKey == "" {
		return &ConfigError{Field: "ImageAPIKey", Message: "IMAGE_API_KEY is required for image generation"}
	}
	return nil
}

// ValidateForAPIConversion 验证 API 转换配置
func (c *Config) ValidateForAPIConversion() error {
	if c.MD2WechatAPIKey == "" && c.DefaultConvertMode == "api" {
		return &ConfigError{Field: "MD2WechatAPIKey", Message: "MD2WECHAT_API_KEY is required for API mode"}
	}
	return nil
}

// GetConfigFile 获取配置文件路径
func (c *Config) GetConfigFile() string {
	return c.configFile
}

// ToMap 转换为 map 用于显示
func (c *Config) ToMap(maskSecret bool) map[string]any {
	result := map[string]any{
		"wechat_appid":         c.WechatAppID,
		"wechat_secret":        maskIf(c.WechatSecret, maskSecret),
		"default_convert_mode": c.DefaultConvertMode,
		"default_theme":        c.DefaultTheme,
		"default_background_type": c.DefaultBackgroundType,
		"md2wechat_api_key":    maskIf(c.MD2WechatAPIKey, maskSecret),
		"md2wechat_base_url":  c.MD2WechatBaseURL,
		"image_provider":       c.ImageProvider,
		"image_api_key":        maskIf(c.ImageAPIKey, maskSecret),
		"image_api_base":       c.ImageAPIBase,
		"image_model":          c.ImageModel,
		"image_size":           c.ImageSize,
		"compress_images":      c.CompressImages,
		"max_image_width":      c.MaxImageWidth,
		"max_image_size_mb":    c.MaxImageSize / 1024 / 1024,
		"http_timeout":         c.HTTPTimeout,
		"config_file":          c.configFile,
	}
	return result
}

// SaveConfig 保存配置到文件
func SaveConfig(path string, cfg *Config) error {
	ext := strings.ToLower(filepath.Ext(path))

	cf := configFile{}
	cf.Wechat.AppID = cfg.WechatAppID
	cf.Wechat.Secret = cfg.WechatSecret
	cf.API.MD2WechatKey = cfg.MD2WechatAPIKey
	cf.API.MD2WechatBaseURL = cfg.MD2WechatBaseURL
	cf.API.ImageKey = cfg.ImageAPIKey
	cf.API.ImageBaseURL = cfg.ImageAPIBase
	cf.API.ImageProvider = cfg.ImageProvider
	cf.API.ImageModel = cfg.ImageModel
	cf.API.ImageSize = cfg.ImageSize
	cf.API.ConvertMode = cfg.DefaultConvertMode
	cf.API.DefaultTheme = cfg.DefaultTheme
	cf.API.BackgroundType = cfg.DefaultBackgroundType
	cf.API.HTTPTimeout = cfg.HTTPTimeout
	cf.Image.Compress = cfg.CompressImages
	cf.Image.MaxWidth = cfg.MaxImageWidth
	cf.Image.MaxSize = int(cfg.MaxImageSize / 1024 / 1024)

	var data []byte
	var err error

	if ext == ".json" {
		data, err = json.MarshalIndent(cf, "", "  ")
	} else {
		data, err = yaml.Marshal(cf)
	}

	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	// 确保目录存在
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory: %w", err)
		}
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// ConfigError 配置错误
type ConfigError struct {
	Field   string
	Message string
	Hint    string // 配置提示
}

func (e *ConfigError) Error() string {
	msg := fmt.Sprintf("配置错误 [%s]: %s", e.Field, e.Message)
	if e.Hint != "" {
		msg += fmt.Sprintf("\n💡 提示: %s", e.Hint)
	}
	return msg
}

// getEnvBool 获取布尔型环境变量
func getEnvBool(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val == "true" || val == "1"
}

// getEnvInt 获取整型环境变量
func getEnvInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return i
}

// getEnvString 获取字符串型环境变量
func getEnvString(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// maskIf 掩码处理
func maskIf(value string, mask bool) string {
	if !mask || value == "" {
		return value
	}
	if len(value) <= 4 {
		return "***"
	}
	return value[:2] + "***" + value[len(value)-2:]
}

// getRelativePath 获取相对路径（用于更友好的显示）
func getRelativePath(fullPath string) string {
	// 如果是用户目录，显示为 ~/.md2wechat.yaml
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" && strings.HasPrefix(fullPath, homeDir) {
		rel := strings.TrimPrefix(fullPath, homeDir)
		if strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, "\\") {
			rel = rel[1:]
		}
		return "~/" + rel
	}

	// 如果是当前目录，直接显示文件名
	if cwd, err := os.Getwd(); err == nil {
		if strings.HasPrefix(fullPath, cwd) {
			rel := strings.TrimPrefix(fullPath, cwd)
			if strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, "\\") {
				rel = rel[1:]
			}
			return "./" + rel
		}
	}

	// 其他情况返回完整路径
	return fullPath
}
