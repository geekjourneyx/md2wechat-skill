package main

import (
	"encoding/json"
	"testing"

	"github.com/geekjourneyx/md2wechat-skill/internal/action"
	"github.com/spf13/cobra"
)

func TestFactsExportContract(t *testing.T) {
	oldJSON, oldVersion := jsonOutput, Version
	oldSourceCommit, oldCfg := SourceCommit, cfg
	t.Cleanup(func() {
		jsonOutput = oldJSON
		Version = oldVersion
		SourceCommit = oldSourceCommit
		cfg = oldCfg
	})

	Version = "3.4.0"
	SourceCommit = "07fdea284e71ddaf5c6b5311238d7e9c2df3b8af"
	cfg = nil
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MD2WECHAT_THEMES_DIR", "")
	t.Setenv("MD2WECHAT_PROMPTS_DIR", "")
	t.Setenv("MD2WECHAT_API_KEY", "")
	t.Setenv("WECHAT_APPID", "")
	t.Setenv("WECHAT_SECRET", "")

	root := &cobra.Command{Use: "md2wechat", SilenceErrors: true, SilenceUsage: true}
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Emit machine-readable JSON output")
	root.AddCommand(newFactsCommand())
	root.SetArgs([]string{"facts", "export", "--json"})

	stdout := captureStdout(t, func() {
		if err := root.Execute(); err != nil {
			t.Fatalf("facts export: %v", err)
		}
	})

	var response struct {
		Success       bool          `json:"success"`
		Code          string        `json:"code"`
		SchemaVersion string        `json:"schema_version"`
		Status        action.Status `json:"status"`
		Data          struct {
			SchemaVersion string `json:"schema_version"`
			Version       string `json:"version"`
			SourceCommit  string `json:"source_commit"`
			Counts        struct {
				APIThemes            int `json:"api_themes"`
				RecommendedScenarios int `json:"recommended_scenarios"`
				RecommendedSyntax    int `json:"recommended_syntax"`
				RenderSyntax         int `json:"render_syntax"`
			} `json:"counts"`
			SideEffects struct {
				Convert     bool `json:"convert"`
				CreateDraft bool `json:"create_draft"`
			} `json:"side_effects"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}

	if !response.Success || response.Code != "FACTS_EXPORTED" {
		t.Fatalf("unexpected response status: %#v", response)
	}
	if response.SchemaVersion != action.SchemaVersion || response.Status != action.StatusCompleted {
		t.Fatalf("unexpected envelope: %#v", response)
	}
	if response.Data.SchemaVersion != "v1" {
		t.Fatalf("schema_version = %q, want v1", response.Data.SchemaVersion)
	}
	if response.Data.Version != "3.4.0" {
		t.Fatalf("version = %q, want 3.4.0", response.Data.Version)
	}
	if response.Data.SourceCommit != "07fdea284e71ddaf5c6b5311238d7e9c2df3b8af" {
		t.Fatalf("source_commit = %q", response.Data.SourceCommit)
	}
	if response.Data.Counts.APIThemes != 48 {
		t.Fatalf("api_themes = %d, want 48", response.Data.Counts.APIThemes)
	}
	if response.Data.Counts.RecommendedScenarios != 77 {
		t.Fatalf("recommended_scenarios = %d, want 77", response.Data.Counts.RecommendedScenarios)
	}
	if response.Data.Counts.RecommendedSyntax != 56 {
		t.Fatalf("recommended_syntax = %d, want 56", response.Data.Counts.RecommendedSyntax)
	}
	if response.Data.Counts.RenderSyntax != 63 {
		t.Fatalf("render_syntax = %d, want 63", response.Data.Counts.RenderSyntax)
	}
	if response.Data.SideEffects.Convert {
		t.Fatal("convert side effect must be false")
	}
	if !response.Data.SideEffects.CreateDraft {
		t.Fatal("create_draft side effect must be true")
	}
}
