package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/layoutcatalog"
)

func TestDoctorJSONReportsBlockedWhenAPIKeyMissing(t *testing.T) {
	chdirToRepoRoot(t)
	isolateDoctorEnv(t)
	layoutcatalog.ResetDefaultCatalogForTests()
	t.Cleanup(layoutcatalog.ResetDefaultCatalogForTests)

	oldJSON := jsonOutput
	t.Cleanup(func() {
		jsonOutput = oldJSON
	})
	jsonOutput = true

	stdout := captureStdout(t, func() {
		doctorCmd.SetArgs(nil)
		if err := doctorCmd.Execute(); err != nil {
			t.Fatalf("doctorCmd.Execute() error = %v", err)
		}
	})

	response := decodeDoctorResponse(t, stdout)
	if response["success"] != true || response["code"] != codeDoctorCompleted {
		t.Fatalf("unexpected response: %#v", response)
	}
	data := response["data"].(map[string]any)
	if data["overall"] != "blocked" || data["live"] != false {
		t.Fatalf("unexpected doctor data: %#v", data)
	}
	readiness := data["readiness"].(map[string]any)
	if readiness["format_api"] != false || readiness["advanced_layout"] != false {
		t.Fatalf("unexpected readiness: %#v", readiness)
	}
	check := findDoctorCheck(t, data, "api.config")
	if check["status"] != "fail" || check["blocking"] != true {
		t.Fatalf("unexpected api check: %#v", check)
	}
}

func TestDoctorJSONReadyWithLocalConfigPresent(t *testing.T) {
	chdirToRepoRoot(t)
	isolateDoctorEnv(t)
	layoutcatalog.ResetDefaultCatalogForTests()
	t.Cleanup(layoutcatalog.ResetDefaultCatalogForTests)

	oldJSON := jsonOutput
	t.Cleanup(func() {
		jsonOutput = oldJSON
	})
	jsonOutput = true
	t.Setenv("MD2WECHAT_API_KEY", "test-api-key")
	t.Setenv("WECHAT_APPID", "wx-appid")
	t.Setenv("WECHAT_SECRET", "wx-secret")

	stdout := captureStdout(t, func() {
		doctorCmd.SetArgs(nil)
		if err := doctorCmd.Execute(); err != nil {
			t.Fatalf("doctorCmd.Execute() error = %v", err)
		}
	})

	response := decodeDoctorResponse(t, stdout)
	data := response["data"].(map[string]any)
	if data["overall"] != "ready" {
		t.Fatalf("unexpected doctor data: %#v", data)
	}
	readiness := data["readiness"].(map[string]any)
	if readiness["format_api"] != true || readiness["advanced_layout"] != true || readiness["draft"] != true {
		t.Fatalf("unexpected readiness: %#v", readiness)
	}
	if check := findDoctorCheck(t, data, "layout.catalog"); check["status"] != "pass" {
		t.Fatalf("unexpected layout check: %#v", check)
	}
	if check := findDoctorCheck(t, data, "theme.catalog"); check["status"] != "pass" {
		t.Fatalf("unexpected theme check: %#v", check)
	}
}

func TestDoctorTextOutputDoesNotExposeSecrets(t *testing.T) {
	chdirToRepoRoot(t)
	isolateDoctorEnv(t)
	layoutcatalog.ResetDefaultCatalogForTests()
	t.Cleanup(layoutcatalog.ResetDefaultCatalogForTests)

	oldJSON := jsonOutput
	t.Cleanup(func() {
		jsonOutput = oldJSON
	})
	jsonOutput = false
	t.Setenv("MD2WECHAT_API_KEY", "secret-api-key")
	t.Setenv("WECHAT_APPID", "wx-appid")
	t.Setenv("WECHAT_SECRET", "secret-wechat")

	stdout := captureStdout(t, func() {
		doctorCmd.SetArgs(nil)
		if err := doctorCmd.Execute(); err != nil {
			t.Fatalf("doctorCmd.Execute() error = %v", err)
		}
	})

	output := string(stdout)
	if !strings.Contains(output, "Overall: ready") || !strings.Contains(output, "layout.catalog") {
		t.Fatalf("unexpected text output: %s", output)
	}
	if strings.Contains(output, "secret-api-key") || strings.Contains(output, "secret-wechat") {
		t.Fatalf("doctor output exposed secrets: %s", output)
	}
}

func isolateDoctorEnv(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("WECHAT_APPID", "")
	t.Setenv("WECHAT_SECRET", "")
	t.Setenv("MD2WECHAT_API_KEY", "")
	t.Setenv("MD2WECHAT_BASE_URL", "")
	t.Setenv("CONVERT_MODE", "")
	t.Setenv("DEFAULT_THEME", "")
	t.Setenv("MD2WECHAT_THEMES_DIR", "")
}

func decodeDoctorResponse(t *testing.T, stdout []byte) map[string]any {
	t.Helper()
	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	data, ok := response["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object: %#v", response)
	}
	if _, ok := data["readiness"].(map[string]any); !ok {
		t.Fatalf("expected readiness object: %#v", data)
	}
	return response
}

func findDoctorCheck(t *testing.T, data map[string]any, id string) map[string]any {
	t.Helper()
	checks, ok := data["checks"].([]any)
	if !ok {
		t.Fatalf("expected checks array: %#v", data)
	}
	for _, raw := range checks {
		check, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("expected check object: %#v", raw)
		}
		if check["id"] == id {
			return check
		}
	}
	t.Fatalf("missing check %q in %#v", id, checks)
	return nil
}
