package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestSyncCommandRejectsUnknownSubcommands(t *testing.T) {
	for _, name := range []string{"draft", "auth", "record", "status", "typo"} {
		t.Run(name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "md2wechat"}
			cmd.AddCommand(newSyncCommand())
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{"sync", name})
			if err := cmd.Execute(); err == nil {
				t.Fatal("unknown sync subcommand returned success")
			}
		})
	}
}

func TestSyncPrepareEnvelope(t *testing.T) {
	old := jsonOutput
	jsonOutput = true
	t.Cleanup(func() { jsonOutput = old })
	dir := t.TempDir()
	article := filepath.Join(dir, "article.md")
	if err := os.WriteFile(article, []byte("# Title\n\nBody"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := newSyncPrepareCommand()
	cmd.SetArgs([]string{article, "--output", filepath.Join(dir, "out")})
	stdout := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
	})
	resp := decodeResponse(t, stdout)
	if resp["code"] != "SYNC_PREPARED" || resp["status"] != "action_required" || resp["success"] != true {
		t.Fatalf("%s", stdout)
	}
	data := responseData(t, resp)
	if data["execution_owner"] != "host_agent" || data["title"] != "Title" {
		t.Fatalf("%v", data)
	}
	if _, err := os.Stat(data["body_html"].(string)); err != nil {
		t.Fatal(err)
	}
}

func TestSyncPrepareRejectsInput(t *testing.T) {
	dir := t.TempDir()
	article := filepath.Join(dir, "article.md")
	if err := os.WriteFile(article, []byte("# Title"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := newSyncPrepareCommand()
	out := filepath.Join(dir, "out")
	cmd.SetArgs([]string{article, "--output", out})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("accepted title-only source")
	}
	cliErr, ok := extractCLIError(err)
	if !ok || cliErr.Code != "SYNC_PREPARE_FAILED" {
		t.Fatal(err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("created rejected output")
	}
}
