package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func captureStderr(t *testing.T, fn func()) []byte {
	t.Helper()

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	t.Cleanup(func() {
		os.Stderr = oldStderr
	})

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	return buf.Bytes()
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
	if !strings.Contains(output, "wechat:") || strings.Contains(output, "\"success\"") {
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

func TestConfigInitWritesSampleAPIBaseURLAndImageSettings(t *testing.T) {
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
		"image_provider: openai",
		"image_model: gpt-image-1.5",
		"image_size: 1024x1024",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected generated config to contain %q, got:\n%s", snippet, content)
		}
	}
}
