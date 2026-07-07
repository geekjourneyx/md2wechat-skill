package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdviseCommandRequiresJSON(t *testing.T) {
	oldJSON := jsonOutput
	t.Cleanup(func() { jsonOutput = oldJSON })

	jsonOutput = false

	err := adviseCmd.RunE(adviseCmd, []string{"article.md"})
	if err == nil {
		t.Fatal("expected --json requirement error")
	}
	cliErr, ok := err.(*cliError)
	if !ok {
		t.Fatalf("error type = %T, want *cliError", err)
	}
	if cliErr.Code != codeConfigInvalid {
		t.Fatalf("code = %q, want %q", cliErr.Code, codeConfigInvalid)
	}
	if !strings.Contains(cliErr.Message, "--json") {
		t.Fatalf("message = %q, want --json hint", cliErr.Message)
	}
}

func TestAdviseCommandMissingFileUsesAdviseReadCode(t *testing.T) {
	oldJSON := jsonOutput
	t.Cleanup(func() { jsonOutput = oldJSON })

	jsonOutput = true

	err := adviseCmd.RunE(adviseCmd, []string{filepath.Join(t.TempDir(), "missing.md")})
	if err == nil {
		t.Fatal("expected missing file error")
	}
	cliErr, ok := err.(*cliError)
	if !ok {
		t.Fatalf("error type = %T, want *cliError", err)
	}
	if cliErr.Code != codeAdviseReadFailed {
		t.Fatalf("code = %q, want %q", cliErr.Code, codeAdviseReadFailed)
	}
}

func TestAdviseCommandJSONContract(t *testing.T) {
	oldJSON := jsonOutput
	t.Cleanup(func() { jsonOutput = oldJSON })

	jsonOutput = true

	articlePath := filepath.Join(t.TempDir(), "article.md")
	markdown := strings.Join([]string{
		"# Launch Notes",
		"",
		"## Context",
		"Paragraph one with 42% metric.",
		"",
		"## Steps",
		"1. Draft the article.",
		"2. Review the output.",
		"",
		"> Quote ready line.",
	}, "\n")
	if err := os.WriteFile(articlePath, []byte(markdown), 0600); err != nil {
		t.Fatalf("write article: %v", err)
	}

	stdout := captureStdout(t, func() {
		if err := adviseCmd.RunE(adviseCmd, []string{articlePath}); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})

	var response struct {
		Success bool   `json:"success"`
		Code    string `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
		Data    struct {
			SchemaVersion string `json:"schema_version"`
			Command       string `json:"command"`
			Source        string `json:"source"`
			Actions       []struct {
				Tool string `json:"tool"`
			} `json:"actions"`
			Layout struct {
				MaxModules int `json:"max_modules"`
			} `json:"layout"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if !response.Success || response.Code != codeAdviseCompleted || response.Message != "Article advice completed" {
		t.Fatalf("unexpected envelope: %#v", response)
	}
	if response.Status != "completed" {
		t.Fatalf("status = %q, want completed", response.Status)
	}
	if response.Data.SchemaVersion != "v1" || response.Data.Command != "advise" {
		t.Fatalf("unexpected advise data: %#v", response.Data)
	}
	if response.Data.Source != articlePath {
		t.Fatalf("source = %q, want %q", response.Data.Source, articlePath)
	}
	if response.Data.Layout.MaxModules != 3 {
		t.Fatalf("layout.max_modules = %d, want 3", response.Data.Layout.MaxModules)
	}
}
