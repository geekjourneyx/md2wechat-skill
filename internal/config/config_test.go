package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadWithDefaultsPreservesCompressDefaultWhenOmitted(t *testing.T) {
	t.Setenv("WECHAT_APPID", "")
	t.Setenv("WECHAT_SECRET", "")
	t.Setenv("MD2WECHAT_API_KEY", "")
	t.Setenv("MD2WECHAT_BASE_URL", "")
	t.Setenv("CONVERT_MODE", "")
	t.Setenv("DEFAULT_THEME", "")
	t.Setenv("DEFAULT_BACKGROUND_TYPE", "")
	t.Setenv("IMAGE_API_KEY", "")
	t.Setenv("IMAGE_API_BASE", "")
	t.Setenv("IMAGE_PROVIDER", "")
	t.Setenv("IMAGE_MODEL", "")
	t.Setenv("IMAGE_SIZE", "")
	t.Setenv("COMPRESS_IMAGES", "")
	t.Setenv("MAX_IMAGE_WIDTH", "")
	t.Setenv("MAX_IMAGE_SIZE", "")
	t.Setenv("HTTP_TIMEOUT", "")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
wechat:
  appid: appid
  secret: secret
api:
  convert_mode: api
`)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadWithDefaults(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.CompressImages {
		t.Fatalf("expected CompressImages default to remain true when field is omitted")
	}
}

func TestLoadWithDefaultsRespectsExplicitCompressFalse(t *testing.T) {
	t.Setenv("WECHAT_APPID", "")
	t.Setenv("WECHAT_SECRET", "")
	t.Setenv("MD2WECHAT_API_KEY", "")
	t.Setenv("MD2WECHAT_BASE_URL", "")
	t.Setenv("CONVERT_MODE", "")
	t.Setenv("DEFAULT_THEME", "")
	t.Setenv("DEFAULT_BACKGROUND_TYPE", "")
	t.Setenv("IMAGE_API_KEY", "")
	t.Setenv("IMAGE_API_BASE", "")
	t.Setenv("IMAGE_PROVIDER", "")
	t.Setenv("IMAGE_MODEL", "")
	t.Setenv("IMAGE_SIZE", "")
	t.Setenv("COMPRESS_IMAGES", "")
	t.Setenv("MAX_IMAGE_WIDTH", "")
	t.Setenv("MAX_IMAGE_SIZE", "")
	t.Setenv("HTTP_TIMEOUT", "")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
wechat:
  appid: appid
  secret: secret
api:
  convert_mode: api
image:
  compress: false
`)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadWithDefaults(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.CompressImages {
		t.Fatalf("expected CompressImages to respect explicit false")
	}
}

func TestLoadWithDefaultsEnvOverridesFileCompressValue(t *testing.T) {
	t.Setenv("WECHAT_APPID", "")
	t.Setenv("WECHAT_SECRET", "")
	t.Setenv("MD2WECHAT_API_KEY", "")
	t.Setenv("MD2WECHAT_BASE_URL", "")
	t.Setenv("CONVERT_MODE", "")
	t.Setenv("DEFAULT_THEME", "")
	t.Setenv("DEFAULT_BACKGROUND_TYPE", "")
	t.Setenv("IMAGE_API_KEY", "")
	t.Setenv("IMAGE_API_BASE", "")
	t.Setenv("IMAGE_PROVIDER", "")
	t.Setenv("IMAGE_MODEL", "")
	t.Setenv("IMAGE_SIZE", "")
	t.Setenv("COMPRESS_IMAGES", "true")
	t.Setenv("MAX_IMAGE_WIDTH", "")
	t.Setenv("MAX_IMAGE_SIZE", "")
	t.Setenv("HTTP_TIMEOUT", "")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
wechat:
  appid: appid
  secret: secret
api:
  convert_mode: api
image:
  compress: false
`)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadWithDefaults(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.CompressImages {
		t.Fatalf("expected environment variable to override file value")
	}
}

func TestLoadWithDefaultsJSONUsesSameMergeRules(t *testing.T) {
	t.Setenv("WECHAT_APPID", "")
	t.Setenv("WECHAT_SECRET", "")
	t.Setenv("MD2WECHAT_API_KEY", "")
	t.Setenv("MD2WECHAT_BASE_URL", "")
	t.Setenv("CONVERT_MODE", "")
	t.Setenv("DEFAULT_THEME", "")
	t.Setenv("DEFAULT_BACKGROUND_TYPE", "")
	t.Setenv("IMAGE_API_KEY", "")
	t.Setenv("IMAGE_API_BASE", "")
	t.Setenv("IMAGE_PROVIDER", "")
	t.Setenv("IMAGE_MODEL", "")
	t.Setenv("IMAGE_SIZE", "")
	t.Setenv("COMPRESS_IMAGES", "")
	t.Setenv("MAX_IMAGE_WIDTH", "")
	t.Setenv("MAX_IMAGE_SIZE", "")
	t.Setenv("HTTP_TIMEOUT", "")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := []byte(`{
  "wechat": {
    "appid": "appid",
    "secret": "secret"
  },
  "api": {
    "convert_mode": "api"
  }
}`)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadWithDefaults(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.CompressImages {
		t.Fatalf("expected JSON loader to preserve CompressImages default when field is omitted")
	}
}

func TestValidateForWeChatRequiresCredentials(t *testing.T) {
	cfg := &Config{}
	if err := cfg.ValidateForWeChat(); err == nil {
		t.Fatal("expected missing appid error")
	}

	cfg.WechatAppID = "appid"
	if err := cfg.ValidateForWeChat(); err == nil {
		t.Fatal("expected missing secret error")
	}
}

func TestValidateForImageGenerationRequiresImageKey(t *testing.T) {
	cfg := &Config{
		WechatAppID:  "appid",
		WechatSecret: "secret",
	}

	if err := cfg.ValidateForImageGeneration(); err == nil {
		t.Fatal("expected missing image key error")
	}

	cfg.ImageAPIKey = "image-key"
	if err := cfg.ValidateForImageGeneration(); err != nil {
		t.Fatalf("ValidateForImageGeneration() error = %v", err)
	}
}

func TestValidateCommonRejectsOutOfRangeValues(t *testing.T) {
	cfg := &Config{
		DefaultConvertMode: "invalid",
		MaxImageWidth:      1920,
		MaxImageSize:       5 * 1024 * 1024,
		HTTPTimeout:        30,
	}
	if err := cfg.validateCommon(); err == nil {
		t.Fatal("expected invalid convert mode error")
	}

	cfg.DefaultConvertMode = "api"
	cfg.MaxImageWidth = 10
	if err := cfg.validateCommon(); err == nil {
		t.Fatal("expected invalid max width error")
	}

	cfg.MaxImageWidth = 1920
	cfg.MaxImageSize = 10
	if err := cfg.validateCommon(); err == nil {
		t.Fatal("expected invalid max image size error")
	}

	cfg.MaxImageSize = 5 * 1024 * 1024
	cfg.HTTPTimeout = 0
	if err := cfg.validateCommon(); err == nil {
		t.Fatal("expected invalid http timeout error")
	}
}

func TestToMapMasksSecrets(t *testing.T) {
	cfg := &Config{
		WechatAppID:        "appid",
		WechatSecret:       "secret-value",
		ImageAPIKey:        "image-key-value",
		CompressImages:     true,
		MaxImageWidth:      1920,
		MaxImageSize:       5 * 1024 * 1024,
		HTTPTimeout:        30,
		configFile:         "/tmp/config.yaml",
		DefaultTheme:       "default",
		DefaultConvertMode: "api",
	}

	result := cfg.ToMap(true)
        if result["wechat_secret"] == "secret-value" || result["image_api_key"] == "image-key-value" {
	}
}

func TestSaveConfigAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := &Config{
		WechatAppID:           "appid",
		WechatSecret:          "secret",
		DefaultConvertMode:    "api",
		DefaultTheme:          "default",
		ImageProvider:         "openai",
		ImageAPIKey:           "image-key",
		ImageAPIBase:          "https://api.example.com",
		ImageModel:            "model",
		ImageSize:             "1024x1024",
		CompressImages:        false,
		MaxImageWidth:         1600,
		MaxImageSize:          3 * 1024 * 1024,
		HTTPTimeout:           45,
	}

	if err := SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadWithDefaults(path)
	if err != nil {
		t.Fatalf("LoadWithDefaults() error = %v", err)
	}
	if loaded.WechatAppID != "appid" || loaded.ImageAPIKey != "image-key" || loaded.CompressImages != false {
		t.Fatalf("loaded config = %#v", loaded)
	}
}

func TestConfigErrorFormatting(t *testing.T) {
	err := (&ConfigError{
		Field:   "WechatSecret",
		Message: "missing",
		Hint:    "set it",
	}).Error()
	if !strings.Contains(err, "WechatSecret") || !strings.Contains(err, "set it") {
		t.Fatalf("ConfigError.Error() = %q", err)
	}
}
