package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/action"
)

func TestPrintJSONUsesCompactMachineOutput(t *testing.T) {
	got := captureStdout(t, func() {
		printJSON(map[string]any{
			"alpha":  "one",
			"nested": map[string]any{"beta": true},
		})
	})

	trimmed := bytes.TrimSpace(got)
	if bytes.Contains(trimmed, []byte("\n")) {
		t.Fatalf("printJSON() emitted indented JSON: %q", got)
	}
	var decoded map[string]any
	if err := json.Unmarshal(trimmed, &decoded); err != nil {
		t.Fatalf("printJSON() emitted invalid JSON: %v", err)
	}
}

func TestRunVersionOutputsJSONEnvelopeWhenRequested(t *testing.T) {
	oldJSON, oldVersion := jsonOutput, Version
	t.Cleanup(func() {
		jsonOutput = oldJSON
		Version = oldVersion
	})

	jsonOutput = true
	Version = "1.2.3-test"

	stdout := captureStdout(t, func() {
		runVersion()
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["success"] != true || response["code"] != "VERSION_SHOWN" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response["schema_version"] != action.SchemaVersion || response["status"] != string(action.StatusCompleted) {
		t.Fatalf("unexpected envelope: %#v", response)
	}
	data, _ := response["data"].(map[string]any)
	if data["version"] != "1.2.3-test" {
		t.Fatalf("unexpected version data: %#v", data)
	}
}

func TestRunVersionJSONStaysMinimal(t *testing.T) {
	oldJSON, oldVersion := jsonOutput, Version
	t.Cleanup(func() {
		jsonOutput = oldJSON
		Version = oldVersion
	})

	jsonOutput = true
	Version = "2.0.0-test"

	stdout := captureStdout(t, func() {
		runVersion()
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	data, _ := response["data"].(map[string]any)
	if data["version"] != "2.0.0-test" {
		t.Fatalf("version = %#v, want 2.0.0-test", data["version"])
	}
	for _, key := range []string{"commit", "built_at", "layout_catalog_version", "layout_module_count"} {
		if _, ok := data[key]; ok {
			t.Fatalf("version data should not include %s: %#v", key, data)
		}
	}
}

func TestResponseErrorUsesStableEnvelope(t *testing.T) {
	oldExit := exitFunc
	t.Cleanup(func() {
		exitFunc = oldExit
	})

	exitCode := 0
	exitFunc = func(code int) {
		exitCode = code
	}

	stdout := captureStdout(t, func() {
		responseErrorWith("CONFIG_INVALID", errors.New("bad config"))
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["success"] != false || response["code"] != "CONFIG_INVALID" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response["schema_version"] != action.SchemaVersion || response["status"] != string(action.StatusFailed) || response["retryable"] != false {
		t.Fatalf("unexpected envelope: %#v", response)
	}
	if response["message"] != "bad config" || response["error"] != "bad config" {
		t.Fatalf("unexpected error payload: %#v", response)
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
}

func TestResponseErrorExtractsCLIErrorCode(t *testing.T) {
	oldExit := exitFunc
	t.Cleanup(func() {
		exitFunc = oldExit
	})

	exitCode := 0
	exitFunc = func(code int) {
		exitCode = code
	}

	stdout := captureStdout(t, func() {
		responseError(newCLIError(codeConfigInvalid, "broken config"))
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["code"] != codeConfigInvalid || response["message"] != "broken config" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response["schema_version"] != action.SchemaVersion || response["status"] != string(action.StatusFailed) {
		t.Fatalf("unexpected envelope: %#v", response)
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
}

func TestResponseErrorIncludesDetailsAndNextActions(t *testing.T) {
	oldExit := exitFunc
	t.Cleanup(func() {
		exitFunc = oldExit
	})

	exitCode := 0
	exitFunc = func(code int) {
		exitCode = code
	}

	message := "Theme autumn-warm is not available for api mode."
	stdout := captureStdout(t, func() {
		err := newCLIErrorWithDetails(
			"THEME_MODE_MISMATCH",
			message,
			map[string]any{
				"theme":         "autumn-warm",
				"mode":          "api",
				"allowed_types": []string{"ai"},
			},
			[]string{"Choose a theme that supports api mode."},
		)
		responseError(err)
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["success"] != false || response["code"] != "THEME_MODE_MISMATCH" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response["message"] != message || response["error"] != message {
		t.Fatalf("unexpected error payload: %#v", response)
	}
	errorDetails, _ := response["error_details"].(map[string]any)
	if errorDetails["theme"] != "autumn-warm" || errorDetails["mode"] != "api" {
		t.Fatalf("unexpected error details: %#v", response["error_details"])
	}
	nextActions, _ := response["next_actions"].([]any)
	if len(nextActions) != 1 {
		t.Fatalf("next actions length = %d, want 1: %#v", len(nextActions), response["next_actions"])
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
}

func TestResponseSuccessUsesStableEnvelope(t *testing.T) {
	stdout := captureStdout(t, func() {
		responseSuccessWith("TEST_OK", "ready", map[string]any{"ok": true})
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["success"] != true || response["code"] != "TEST_OK" || response["message"] != "ready" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response["schema_version"] != action.SchemaVersion || response["status"] != string(action.StatusCompleted) || response["retryable"] != false {
		t.Fatalf("unexpected envelope: %#v", response)
	}
	data, _ := response["data"].(map[string]any)
	if data["ok"] != true {
		t.Fatalf("unexpected data payload: %#v", data)
	}
}

func TestResponseActionRequiredUsesStableEnvelope(t *testing.T) {
	stdout := captureStdout(t, func() {
		responseActionRequiredWith("TEST_ACTION", "need input", map[string]any{"prompt": "run this"})
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["success"] != true || response["code"] != "TEST_ACTION" || response["status"] != string(action.StatusActionRequired) {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response["schema_version"] != action.SchemaVersion || response["retryable"] != false {
		t.Fatalf("unexpected envelope: %#v", response)
	}
}

func TestRunVersionOutputsPlainTextByDefault(t *testing.T) {
	oldJSON, oldVersion := jsonOutput, Version
	t.Cleanup(func() {
		jsonOutput = oldJSON
		Version = oldVersion
	})

	jsonOutput = false
	Version = "9.9.9"

	stdout := captureStdout(t, func() {
		runVersion()
	})

	if strings.TrimSpace(string(stdout)) != "9.9.9" {
		t.Fatalf("unexpected plain version output: %q", string(stdout))
	}
}
