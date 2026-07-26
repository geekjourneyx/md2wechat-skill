package config

import (
	"io"
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
		MD2WechatAPIKey:    "api-key-value",
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
	if result["wechat_secret"] == "secret-value" || result["md2wechat_api_key"] == "api-key-value" || result["image_api_key"] == "image-key-value" {
		t.Fatalf("expected secrets to be masked: %#v", result)
	}
}

func TestSaveConfigAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := &Config{
		WechatAppID:           "appid",
		WechatSecret:          "secret",
		MD2WechatAPIKey:       "api-key",
		MD2WechatBaseURL:      "https://example.com",
		DefaultConvertMode:    "api",
		DefaultTheme:          "default",
		DefaultBackgroundType: "none",
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

func TestLoadWithDefaultsParsesWechatProxyURL(t *testing.T) {
	t.Setenv("WECHAT_PROXY_URL", "")

	path := writeTempConfig(t, `
wechat:
  proxy_url: https://proxy.example.com:8443
api:
  convert_mode: api
`)

	cfg, err := LoadWithDefaults(path)
	if err != nil {
		t.Fatalf("LoadWithDefaults() error = %v", err)
	}
	if cfg.WechatProxyURL != "https://proxy.example.com:8443" {
		t.Fatalf("WechatProxyURL = %q", cfg.WechatProxyURL)
	}
}

func TestLoadWithDefaultsEnvOverridesWechatProxyURL(t *testing.T) {
	t.Setenv("WECHAT_PROXY_URL", "http://env-user:env-pass@env-proxy.example.com:8080")

	path := writeTempConfig(t, `
wechat:
  proxy_url: https://file-proxy.example.com
api:
  convert_mode: api
`)

	cfg, err := LoadWithDefaults(path)
	if err != nil {
		t.Fatalf("LoadWithDefaults() error = %v", err)
	}
	if cfg.WechatProxyURL != "http://env-user:env-pass@env-proxy.example.com:8080" {
		t.Fatalf("WechatProxyURL = %q", cfg.WechatProxyURL)
	}
}

func TestSaveConfigAndLoadRoundTripPreservesWechatProxyURL(t *testing.T) {
	t.Setenv("WECHAT_PROXY_URL", "")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := &Config{
		WechatProxyURL:     "https://user:pass@proxy.example.com:8443",
		DefaultConvertMode: "api",
		DefaultTheme:       "default",
		CompressImages:     true,
		MaxImageWidth:      1920,
		MaxImageSize:       5 * 1024 * 1024,
		HTTPTimeout:        30,
		MD2WechatBaseURL:   "https://www.md2wechat.cn",
		ImageProvider:      "openai",
		ImageAPIBase:       "https://api.openai.com/v1",
		ImageModel:         "gpt-image-2",
		ImageSize:          "auto",
	}

	if err := SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadWithDefaults(path)
	if err != nil {
		t.Fatalf("LoadWithDefaults() error = %v", err)
	}
	if loaded.WechatProxyURL != cfg.WechatProxyURL {
		t.Fatalf("loaded WechatProxyURL = %q", loaded.WechatProxyURL)
	}
}

func TestToMapMasksWechatProxyURLPassword(t *testing.T) {
	tests := []struct {
		name       string
		proxyURL   string
		wantMasked string
	}{
		{
			name:       "password",
			proxyURL:   "https://user:secret-password@proxy.example.com:8443",
			wantMasked: "https://user:***@proxy.example.com:8443",
		},
		{
			name:       "username-only-token",
			proxyURL:   "http://proxy-token@proxy.example.com:18443",
			wantMasked: "http://***@proxy.example.com:18443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				WechatProxyURL: tt.proxyURL,
			}

			masked := cfg.ToMap(true)
			if got := masked["wechat_proxy_url"]; got != tt.wantMasked {
				t.Fatalf("masked wechat_proxy_url = %q, want %q", got, tt.wantMasked)
			}

			unmasked := cfg.ToMap(false)
			if got := unmasked["wechat_proxy_url"]; got != cfg.WechatProxyURL {
				t.Fatalf("unmasked wechat_proxy_url = %q", got)
			}
		})
	}
}

func TestLoadWithDefaultsRejectsInvalidWechatProxyURLs(t *testing.T) {
	cases := []string{
		"://proxy.example.com",
		"http://",
		"http://:8080",
		"https://",
		"ftp://proxy.example.com",
		"socks5://proxy.example.com",
		"http://proxy.example.com:",
		"http://proxy.example.com:0",
		"http://proxy.example.com:-1",
		"http://proxy.example.com:abc",
		"http://proxy.example.com:65536",
		"http://proxy.example.com:99999",
		"http://[::1]:",
		"http://[::1]:99999",
	}

	for _, proxyURL := range cases {
		t.Run(proxyURL, func(t *testing.T) {
			t.Setenv("WECHAT_PROXY_URL", proxyURL)
			path := writeTempConfig(t, `
api:
  convert_mode: api
`)

			_, err := LoadWithDefaults(path)
			if err == nil || !strings.Contains(err.Error(), "WechatProxyURL") {
				t.Fatalf("LoadWithDefaults() error = %v, want WechatProxyURL validation error", err)
			}
		})
	}
}

func TestLoadWithDefaultsAcceptsValidWechatProxyPorts(t *testing.T) {
	cases := []string{
		"http://proxy.example.com",
		"http://proxy.example.com:1",
		"http://proxy.example.com:65535",
		"http://[::1]:18443",
	}

	for _, proxyURL := range cases {
		t.Run(proxyURL, func(t *testing.T) {
			t.Setenv("WECHAT_PROXY_URL", proxyURL)
			path := writeTempConfig(t, `
api:
  convert_mode: api
`)

			cfg, err := LoadWithDefaults(path)
			if err != nil {
				t.Fatalf("LoadWithDefaults() error = %v", err)
			}
			if cfg.WechatProxyURL != proxyURL {
				t.Fatalf("WechatProxyURL = %q, want %q", cfg.WechatProxyURL, proxyURL)
			}
		})
	}
}

func TestLoadWithDefaultsAppliesVolcengineImageDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := strings.TrimSpace(`
api:
  image_provider: "volcengine"
`)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadWithDefaults(path)
	if err != nil {
		t.Fatalf("LoadWithDefaults() error = %v", err)
	}
	if cfg.ImageAPIBase != "https://ark.cn-beijing.volces.com/api/v3" {
		t.Fatalf("ImageAPIBase = %q", cfg.ImageAPIBase)
	}
	if cfg.ImageModel != "doubao-seedream-5-0-pro-260628" {
		t.Fatalf("ImageModel = %q", cfg.ImageModel)
	}
	if cfg.ImageSize != "2K" {
		t.Fatalf("ImageSize = %q", cfg.ImageSize)
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

func TestLoadWithDefaultsParsesWechatAccounts(t *testing.T) {
	t.Setenv("WECHAT_APPID", "")
	t.Setenv("WECHAT_SECRET", "")
	t.Setenv("WECHAT_ACCOUNT", "")

	path := writeTempConfig(t, `
wechat:
  appid: legacy-appid
  secret: legacy-secret
  default_account: main
  accounts:
    main:
      appid: named-appid
      secret: named-secret
    client-a:
      appid: client-appid
      secret: client-secret
api:
  convert_mode: api
`)

	cfg, err := LoadWithDefaults(path)
	if err != nil {
		t.Fatalf("LoadWithDefaults() error = %v", err)
	}
	if cfg.WechatAppID != "named-appid" || cfg.WechatSecret != "named-secret" {
		t.Fatalf("effective WeChat credentials = %q/%q", cfg.WechatAppID, cfg.WechatSecret)
	}
	if cfg.WechatAccount != "main" || !cfg.WechatAccountNamed {
		t.Fatalf("selected account = %q named=%v", cfg.WechatAccount, cfg.WechatAccountNamed)
	}
}

func TestLoadWithDefaultsPreservesDirectCredentialsWhenDirectWins(t *testing.T) {
	t.Setenv("WECHAT_APPID", "")
	t.Setenv("WECHAT_SECRET", "")
	t.Setenv("WECHAT_ACCOUNT", "")

	path := writeTempConfig(t, `
wechat:
  appid: legacy-appid
  secret: legacy-secret
  accounts:
    client-a:
      appid: client-appid
      secret: client-secret
api:
  convert_mode: api
`)

	cfg, err := LoadWithDefaults(path)
	if err != nil {
		t.Fatalf("LoadWithDefaults() error = %v", err)
	}
	if cfg.WechatAppID != "legacy-appid" || cfg.WechatSecret != "legacy-secret" {
		t.Fatalf("effective direct credentials = %q/%q", cfg.WechatAppID, cfg.WechatSecret)
	}
	if cfg.WechatAccount != "" || cfg.WechatAccountNamed {
		t.Fatalf("direct path should not be named: %q named=%v", cfg.WechatAccount, cfg.WechatAccountNamed)
	}
}

func TestLoadWithDefaultsEnvWechatAccountDoesNotUseDirectEnvCredentials(t *testing.T) {
	t.Setenv("WECHAT_APPID", "env-direct-appid")
	t.Setenv("WECHAT_SECRET", "env-direct-secret")
	t.Setenv("WECHAT_ACCOUNT", "client-a")

	path := writeTempConfig(t, `
wechat:
  appid: file-direct-appid
  secret: file-direct-secret
  accounts:
    client-a:
      appid: client-appid
      secret: client-secret
api:
  convert_mode: api
`)

	cfg, err := LoadWithDefaults(path)
	if err != nil {
		t.Fatalf("LoadWithDefaults() error = %v", err)
	}
	if cfg.WechatAppID != "client-appid" || cfg.WechatSecret != "client-secret" {
		t.Fatalf("named credentials should win over direct env: %#v", cfg)
	}
}

func TestLoadWithDefaultsRejectsInvalidWechatAccountConfig(t *testing.T) {
	cases := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name: "invalid name",
			yaml: `
wechat:
  accounts:
    Client.A:
      appid: appid
      secret: secret
`,
			wantErr: "WECHAT_ACCOUNT_INVALID",
		},
		{
			name: "missing secret",
			yaml: `
wechat:
  accounts:
    main:
      appid: appid
`,
			wantErr: "WechatAccounts.main.secret",
		},
		{
			name: "missing default",
			yaml: `
wechat:
  default_account: missing
  accounts:
    main:
      appid: appid
      secret: secret
`,
			wantErr: "WECHAT_ACCOUNT_NOT_FOUND",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("WECHAT_APPID", "")
			t.Setenv("WECHAT_SECRET", "")
			t.Setenv("WECHAT_ACCOUNT", "")
			path := writeTempConfig(t, tc.yaml)
			_, err := LoadWithDefaults(path)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("LoadWithDefaults() error = %v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestResolveWeChatAccountRejectsAmbiguousNamedAccounts(t *testing.T) {
	t.Setenv("WECHAT_APPID", "")
	t.Setenv("WECHAT_SECRET", "")
	t.Setenv("WECHAT_ACCOUNT", "")

	path := writeTempConfig(t, `
wechat:
  accounts:
    a:
      appid: a
      secret: a
    b:
      appid: b
      secret: b
`)

	cfg, err := LoadWithDefaults(path)
	if err != nil {
		t.Fatalf("LoadWithDefaults() error = %v", err)
	}
	err = cfg.ResolveWeChatAccount("")
	if err == nil || !IsWechatAccountAmbiguous(err) {
		t.Fatalf("ResolveWeChatAccount() error = %v, want ambiguous", err)
	}
}

func TestLoadStrictRejectsMalformedConfigWhileLoadFallsBack(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "md2wechat")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(configDir, "config.yaml"),
		[]byte("wechat: ["),
		0600,
	); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	oldQuiet, oldWriter := quietOutput, statusWriter
	SetQuiet(true)
	SetStatusWriter(io.Discard)
	t.Cleanup(func() {
		quietOutput, statusWriter = oldQuiet, oldWriter
	})

	if _, err := LoadStrict(); err == nil ||
		!strings.Contains(err.Error(), "parse yaml") {
		t.Fatalf("LoadStrict() error = %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() fallback error = %v", err)
	}
	if cfg.DefaultConvertMode != "api" {
		t.Fatalf("fallback config = %#v", cfg)
	}
}

func TestApplyImageProviderDefaultsAtlasCloudAlias(t *testing.T) {
	cfg := &Config{ImageProvider: "atlas"}
	applyImageProviderDefaults(cfg)

	if cfg.ImageAPIBase != "https://api.atlascloud.ai/api/v1/model" {
		t.Fatalf("ImageAPIBase = %q", cfg.ImageAPIBase)
	}
	if cfg.ImageModel != "openai/gpt-image-2/text-to-image" {
		t.Fatalf("ImageModel = %q", cfg.ImageModel)
	}
	if cfg.ImageSize != "1024x1024" {
		t.Fatalf("ImageSize = %q", cfg.ImageSize)
	}
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
