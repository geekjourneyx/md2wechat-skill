package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
)

func captureStderr(t *testing.T, fn func()) []byte {
	t.Helper()

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	defer func() {
		os.Stderr = oldStderr
		_ = w.Close()
	}()
	done := readPipeAsync(r)

	fn()

	os.Stderr = oldStderr
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	result := <-done
	if err := r.Close(); err != nil {
		t.Fatalf("close reader: %v", err)
	}
	if result.err != nil {
		t.Fatalf("read stderr: %v", result.err)
	}
	return result.data
}

func TestConfigShowJSONEnvelope(t *testing.T) {
	oldFormat, oldShowSecret := configFormat, configShowSecret
	oldJSON := jsonOutput
	t.Cleanup(func() {
		configFormat, configShowSecret = oldFormat, oldShowSecret
		jsonOutput = oldJSON
	})

	t.Setenv("WECHAT_APPID", "wx-appid")
	t.Setenv("WECHAT_SECRET", "wx-secret")
	configFormat = "json"
	configShowSecret = false
	jsonOutput = true

	stdout := captureStdout(t, func() {
		configCmd.SetArgs([]string{"show"})
		if err := configCmd.Execute(); err != nil {
			t.Fatalf("configCmd.Execute() error = %v", err)
		}
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["success"] != true || response["code"] != "CONFIG_SHOWN" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response["schema_version"] != "v1" || response["status"] != "completed" || response["retryable"] != false {
		t.Fatalf("unexpected envelope: %#v", response)
	}
	data, ok := response["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data block: %#v", response)
	}
	if _, ok := data["config"].(map[string]any); !ok {
		t.Fatalf("expected config map: %#v", data)
	}
}

func TestConfigShowYAMLOutput(t *testing.T) {
	oldFormat, oldShowSecret := configFormat, configShowSecret
	oldJSON := jsonOutput
	t.Cleanup(func() {
		configFormat, configShowSecret = oldFormat, oldShowSecret
		jsonOutput = oldJSON
	})

	t.Setenv("WECHAT_APPID", "wx-appid")
	t.Setenv("WECHAT_SECRET", "wx-secret")
	configFormat = "yaml"
	configShowSecret = false
	jsonOutput = false

	stdout := captureStdout(t, func() {
		configCmd.SetArgs([]string{"show"})
		if err := configCmd.Execute(); err != nil {
			t.Fatalf("configCmd.Execute() error = %v", err)
		}
	})

	output := string(stdout)
	if !strings.Contains(output, "wechat:") || !strings.Contains(output, "md2wechat_base_url: https://www.md2wechat.cn") || strings.Contains(output, "\"success\"") {
		t.Fatalf("unexpected yaml output: %s", output)
	}
}

func TestPrintYAMLConfigMasksWechatProxyURLPassword(t *testing.T) {
	oldFormat, oldShowSecret := configFormat, configShowSecret
	oldJSON := jsonOutput
	t.Cleanup(func() {
		configFormat, configShowSecret = oldFormat, oldShowSecret
		jsonOutput = oldJSON
	})

	t.Setenv("WECHAT_APPID", "wx-appid")
	t.Setenv("WECHAT_SECRET", "wx-secret")
	t.Setenv("WECHAT_PROXY_URL", "http://account:credential-value@proxy.example.com:18443")
	configFormat = "yaml"
	configShowSecret = false
	jsonOutput = false

	stdout := captureStdout(t, func() {
		configCmd.SetArgs([]string{"show"})
		if err := configCmd.Execute(); err != nil {
			t.Fatalf("configCmd.Execute() error = %v", err)
		}
	})

	output := string(stdout)
	if !strings.Contains(output, "proxy_url: http://account:***@proxy.example.com:18443") {
		t.Fatalf("expected masked proxy_url in yaml output, got:\n%s", output)
	}
	if strings.Contains(output, "credential-value") {
		t.Fatalf("proxy password leaked in yaml output:\n%s", output)
	}
}

func TestPrintYAMLConfigMasksWechatProxyURLUsernameOnlyToken(t *testing.T) {
	oldFormat, oldShowSecret := configFormat, configShowSecret
	oldJSON := jsonOutput
	t.Cleanup(func() {
		configFormat, configShowSecret = oldFormat, oldShowSecret
		jsonOutput = oldJSON
	})

	t.Setenv("WECHAT_APPID", "wx-appid")
	t.Setenv("WECHAT_SECRET", "wx-secret")
	t.Setenv("WECHAT_PROXY_URL", "http://proxy-token@proxy.example.com:18443")
	configFormat = "yaml"
	configShowSecret = false
	jsonOutput = false

	stdout := captureStdout(t, func() {
		configCmd.SetArgs([]string{"show"})
		if err := configCmd.Execute(); err != nil {
			t.Fatalf("configCmd.Execute() error = %v", err)
		}
	})

	output := string(stdout)
	if !strings.Contains(output, "proxy_url: http://***@proxy.example.com:18443") {
		t.Fatalf("expected masked proxy_url in yaml output, got:\n%s", output)
	}
	if strings.Contains(output, "proxy-token") {
		t.Fatalf("proxy token leaked in yaml output:\n%s", output)
	}
}

func TestPrintYAMLConfigShowSecretKeepsWechatProxyURLUsernameOnlyToken(t *testing.T) {
	oldFormat, oldShowSecret := configFormat, configShowSecret
	oldJSON := jsonOutput
	t.Cleanup(func() {
		configFormat, configShowSecret = oldFormat, oldShowSecret
		jsonOutput = oldJSON
	})

	t.Setenv("WECHAT_APPID", "wx-appid")
	t.Setenv("WECHAT_SECRET", "wx-secret")
	t.Setenv("WECHAT_PROXY_URL", "http://proxy-token@proxy.example.com:18443")
	configFormat = "yaml"
	configShowSecret = true
	jsonOutput = false

	stdout := captureStdout(t, func() {
		configCmd.SetArgs([]string{"show", "--show-secret"})
		if err := configCmd.Execute(); err != nil {
			t.Fatalf("configCmd.Execute() error = %v", err)
		}
	})

	output := string(stdout)
	if !strings.Contains(output, "proxy_url: http://proxy-token@proxy.example.com:18443") {
		t.Fatalf("expected unmasked proxy_url in yaml output, got:\n%s", output)
	}
}

func TestConfigValidateJSONEnvelope(t *testing.T) {
	oldJSON := jsonOutput
	t.Cleanup(func() {
		jsonOutput = oldJSON
	})

	t.Setenv("WECHAT_APPID", "wx-appid")
	t.Setenv("WECHAT_SECRET", "wx-secret")
	jsonOutput = true

	stdout := captureStdout(t, func() {
		configCmd.SetArgs([]string{"validate"})
		if err := configCmd.Execute(); err != nil {
			t.Fatalf("configCmd.Execute() error = %v", err)
		}
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["success"] != true || response["code"] != "CONFIG_VALIDATED" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response["schema_version"] != "v1" || response["status"] != "completed" || response["retryable"] != false {
		t.Fatalf("unexpected envelope: %#v", response)
	}
}

func TestConfigInitJSONEnvelopeSuppressesHumanStderr(t *testing.T) {
	oldJSON := jsonOutput
	t.Cleanup(func() {
		jsonOutput = oldJSON
	})

	jsonOutput = true
	outputFile := filepath.Join(t.TempDir(), "config.yaml")

	var stdout []byte
	stderr := captureStderr(t, func() {
		stdout = captureStdout(t, func() {
			configCmd.SetArgs([]string{"init", outputFile})
			if err := configCmd.Execute(); err != nil {
				t.Fatalf("configCmd.Execute() error = %v", err)
			}
		})
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["success"] != true || response["code"] != "CONFIG_INITIALIZED" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response["schema_version"] != "v1" || response["status"] != "completed" || response["retryable"] != false {
		t.Fatalf("unexpected envelope: %#v", response)
	}
	if strings.TrimSpace(string(stderr)) != "" {
		t.Fatalf("expected no stderr in json mode, got %q", string(stderr))
	}
	if _, err := os.Stat(outputFile); err != nil {
		t.Fatalf("expected config file to be created: %v", err)
	}
}

func TestConfigInitWritesSampleVolcengineImageSettings(t *testing.T) {
	oldJSON := jsonOutput
	t.Cleanup(func() {
		jsonOutput = oldJSON
	})

	jsonOutput = true
	outputFile := filepath.Join(t.TempDir(), "config.yaml")

	captureStdout(t, func() {
		configCmd.SetArgs([]string{"init", outputFile})
		if err := configCmd.Execute(); err != nil {
			t.Fatalf("configCmd.Execute() error = %v", err)
		}
	})

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}

	content := string(data)
	expectedSnippets := []string{
		"md2wechat_base_url: https://www.md2wechat.cn",
		"image_provider: volcengine",
		"image_base_url: https://ark.cn-beijing.volces.com/api/v3",
		"image_model: doubao-seedream-5-0-pro-260628",
		"image_size: 2K",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected generated config to contain %q, got:\n%s", snippet, content)
		}
	}
	for _, forbidden := range []string{"default_account:", "accounts:"} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("generated single-account config should not contain %q:\n%s", forbidden, content)
		}
	}
}

func TestConfigWechatAccountsJSONDirectOnly(t *testing.T) {
	oldJSON := jsonOutput
	t.Cleanup(func() { jsonOutput = oldJSON })

	t.Setenv("WECHAT_APPID", "wx-direct")
	t.Setenv("WECHAT_SECRET", "direct-secret")
	t.Setenv("WECHAT_ACCOUNT", "")
	jsonOutput = true

	stdout := captureStdout(t, func() {
		configCmd.SetArgs([]string{"wechat-accounts"})
		if err := configCmd.Execute(); err != nil {
			t.Fatalf("configCmd.Execute() error = %v", err)
		}
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["code"] != "WECHAT_ACCOUNTS_SHOWN" {
		t.Fatalf("unexpected response: %#v", response)
	}
	data := response["data"].(map[string]any)
	if len(data["accounts"].([]any)) != 0 {
		t.Fatalf("expected no named accounts: %#v", data)
	}
	current := data["current"].(map[string]any)
	if current["name"] != "" || current["appid"] != "wx-direct" {
		t.Fatalf("unexpected current: %#v", current)
	}
}

func TestConfigWechatAccountsDoesNotExposeSecrets(t *testing.T) {
	data := buildWechatAccountsData(&config.Config{
		WechatDefaultAccount: "main",
		WechatAccounts: map[string]config.WechatAccount{
			"main": {AppID: "wx-main", Secret: "secret-main"},
		},
		WechatAccount:      "main",
		WechatAccountNamed: true,
		WechatAppID:        "wx-main",
		WechatSecret:       "secret-main",
	})
	encoded, _ := json.Marshal(data)
	if strings.Contains(string(encoded), "secret") {
		t.Fatalf("secrets leaked: %s", encoded)
	}
}
