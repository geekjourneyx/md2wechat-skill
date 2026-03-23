package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// configCmd config 命令
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long: `Manage md2wechat configuration.

Configuration Priority:
  1. Environment variables (highest)
  2. Config file
  3. Default values (lowest)

Config file search order:
  1. ~/.config/md2wechat/config.yaml  (global config, recommended)
  2. ~/.md2wechat.yaml                (global config)
  3. ./md2wechat.yaml                  (project config)

💡 Tip: Use global config (~/.md2wechat.yaml) for all your projects.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 默认显示配置
		return runConfigShow(false)
	},
}

var (
	configShowSecret bool
	configFormat     string
)

func init() {
	// show 子命令
	var showCmd = &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigShow(configShowSecret)
		},
	}
	showCmd.Flags().BoolVar(&configShowSecret, "show-secret", false, "Show secret values")
	showCmd.Flags().StringVarP(&configFormat, "format", "f", "json", "Output format: json, yaml")
	configCmd.AddCommand(showCmd)

	// validate 子命令
	var validateCmd = &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runConfigValidate(); err != nil {
				return err
			}
			responseSuccessWith(codeConfigValidated, "Configuration is valid", map[string]any{
				"valid":   true,
				"message": "Configuration is valid",
			})
			return nil
		},
	}
	configCmd.AddCommand(validateCmd)

	// init 子命令
	var initCmd = &cobra.Command{
		Use:   "init [output_file]",
		Short: "Create a sample config file",
		Long: `Create a sample config file.

If no output file is specified, the config will be created in:
  ~/.config/md2wechat/config.yaml

This is the global config location, used by all your projects.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var outputFile string
			if len(args) > 0 {
				outputFile = args[0]
			} else {
				// 默认使用用户目录
				homeDir, _ := os.UserHomeDir()
				configDir := homeDir + "/.config/md2wechat"
				outputFile = configDir + "/config.yaml"
			}

			if err := runConfigInit(outputFile); err != nil {
				return err
			}
			relPath := normalizeConfigOutputPath(outputFile)
			if !jsonOutput {
				fmt.Fprintf(os.Stderr, "\n✅ 配置文件已创建: %s\n", relPath)
				fmt.Fprintf(os.Stderr, "📝 下一步: 编辑配置文件，填入你的微信公众号 AppID 和 Secret\n")
				fmt.Fprintf(os.Stderr, "📍 获取方式: 微信公众平台 > 设置与开发 > 基本配置\n\n")
			}

			responseSuccessWith(codeConfigInitialized, "Config file created. Please edit it with your credentials.", map[string]any{
				"file":    relPath,
				"message": "Config file created. Please edit it with your credentials.",
			})
			return nil
		},
	}
	configCmd.AddCommand(initCmd)
}

// runConfigShow 显示配置
func runConfigShow(showSecret bool) error {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		// 如果加载失败，可能是缺少必需配置，尝试创建一个用于显示
		if os.Getenv("WECHAT_APPID") == "" && os.Getenv("WECHAT_SECRET") == "" {
			return newCLIError(codeConfigNotFound, "no configuration found. Set environment variables or create a config file with 'md2wechat config init'")
		}
		return wrapCLIError(codeConfigInvalid, err, err.Error())
	}

	configData := cfg.ToMap(!showSecret)
	if jsonOutput || configFormat == "json" {
		responseSuccessWith(codeConfigShown, "Configuration loaded", map[string]any{
			"config": configData,
		})
	} else {
		// YAML 格式输出（简化版）
		printYAMLConfig(cfg, !showSecret)
	}

	return nil
}

// runConfigValidate 验证配置
func runConfigValidate() error {
	cfg, err := config.Load()
	if err != nil {
		return wrapCLIError(codeConfigInvalid, err, err.Error())
	}

	// 基本验证已在 Load 中完成
	// 这里可以添加更多验证

	logger := log
	if logger == nil {
		logger = zap.NewNop()
	}
	logger.Info("configuration validated",
		zap.String("config_file", cfg.GetConfigFile()),
		zap.String("convert_mode", cfg.DefaultConvertMode),
		zap.String("default_theme", cfg.DefaultTheme))

	return nil
}

func runConfigInit(outputFile string) error {
	return initConfigFile(outputFile)
}

// initConfigFile 创建示例配置文件
func initConfigFile(outputFile string) error {
	// 检查文件是否已存在
	if _, err := os.Stat(outputFile); err == nil {
		return newCLIError(codeConfigWriteFailed, fmt.Sprintf("config file already exists: %s", outputFile))
	}

	// 创建示例配置
	cfg := &config.Config{
		WechatAppID:           "your_wechat_appid",
		WechatSecret:          "your_wechat_secret",
		ImageProvider:         "openai",
		ImageAPIKey:           "your_image_api_key",
		ImageAPIBase:          "https://api.openai.com/v1",
		ImageModel:            "gpt-image-1.5",
		ImageSize:             "1024x1024",
		DefaultConvertMode:    "api",
		DefaultTheme:          "default",
		CompressImages:        true,
		MaxImageWidth:         1920,
		MaxImageSize:          5 * 1024 * 1024,
		HTTPTimeout:           30,
	}

	if err := config.SaveConfig(outputFile, cfg); err != nil {
		return wrapCLIError(codeConfigWriteFailed, err, err.Error())
	}

	return nil
}

// printYAMLConfig 打印 YAML 格式配置
func printYAMLConfig(cfg *config.Config, maskSecret bool) {
	fmt.Println("# md2wechat Configuration")
	fmt.Printf("# Config file: %s\n\n", cfg.GetConfigFile())

	fmt.Println("wechat:")
	fmt.Printf("  appid: %s\n", cfg.WechatAppID)
	secret := cfg.WechatSecret
	if maskSecret && secret != "" && secret != "your_wechat_secret" {
		if len(secret) > 4 {
			secret = secret[:2] + "***" + secret[len(secret)-2:]
		} else {
			secret = "***"
		}
	}
	fmt.Printf("  secret: %s\n\n", secret)

	fmt.Println("api:")
	fmt.Printf("  image_key: %s\n", maskAPIKey(cfg.ImageAPIKey, maskSecret))
	fmt.Printf("  image_provider: %s\n", cfg.ImageProvider)
	fmt.Printf("  image_base_url: %s\n", cfg.ImageAPIBase)
	fmt.Printf("  image_model: %s\n", cfg.ImageModel)
	fmt.Printf("  image_size: %s\n", cfg.ImageSize)
	fmt.Printf("  convert_mode: %s\n", cfg.DefaultConvertMode)
	fmt.Printf("  default_theme: %s\n", cfg.DefaultTheme)
	fmt.Printf("  http_timeout: %d\n\n", cfg.HTTPTimeout)

	fmt.Println("image:")
	fmt.Printf("  compress: %v\n", cfg.CompressImages)
	fmt.Printf("  max_width: %d\n", cfg.MaxImageWidth)
	fmt.Printf("  max_size_mb: %d\n", cfg.MaxImageSize/1024/1024)
}

func maskAPIKey(key string, mask bool) string {
	if !mask || key == "" || key == "your_image_api_key" {
		return key
	}
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "***" + key[len(key)-4:]
}

func normalizeConfigOutputPath(outputFile string) string {
	relPath := outputFile
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" && strings.HasPrefix(outputFile, homeDir) {
		rel := strings.TrimPrefix(outputFile, homeDir)
		if strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, "\\") {
			rel = rel[1:]
		}
		relPath = "~/" + rel
	}
	return relPath
}
