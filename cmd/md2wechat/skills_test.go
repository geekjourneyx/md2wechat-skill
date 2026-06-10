package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSkillsListCommandExposesEmbeddedSkillMetadata(t *testing.T) {
	oldJSON, oldVersion := jsonOutput, Version
	t.Cleanup(func() {
		jsonOutput = oldJSON
		Version = oldVersion
	})

	jsonOutput = true
	Version = "1.2.3"

	stdout := captureStdout(t, func() {
		if err := skillsListCmd.RunE(skillsListCmd, nil); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})

	var response struct {
		Success bool   `json:"success"`
		Code    string `json:"code"`
		Data    struct {
			Skills []skillSummary `json:"skills"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if !response.Success || response.Code != codeSkillsShown {
		t.Fatalf("unexpected response: %#v", response)
	}

	var found *skillSummary
	for i := range response.Data.Skills {
		if response.Data.Skills[i].Name == "md2wechat" {
			found = &response.Data.Skills[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected md2wechat skill: %#v", response.Data.Skills)
	}
	if found.Version != "1.2.3" || found.Source != "embedded" || found.RiskLevel != "side-effect-aware" {
		t.Fatalf("unexpected skill metadata: %#v", found)
	}
	if found.ReadCommand != "md2wechat skills read md2wechat --json" {
		t.Fatalf("ReadCommand = %q", found.ReadCommand)
	}
	if !strings.HasPrefix(found.ContentHash, "sha256:") || len(found.ContentHash) != len("sha256:")+64 {
		t.Fatalf("ContentHash = %q", found.ContentHash)
	}
	if len(found.SideEffects) == 0 || len(found.Commands) == 0 {
		t.Fatalf("expected side effects and command hints: %#v", found)
	}
}

func TestSkillsReadCommandReturnsEmbeddedSkillContent(t *testing.T) {
	oldJSON, oldVersion := jsonOutput, Version
	t.Cleanup(func() {
		jsonOutput = oldJSON
		Version = oldVersion
	})

	jsonOutput = true
	Version = "2.0.0"

	stdout := captureStdout(t, func() {
		if err := skillsReadCmd.RunE(skillsReadCmd, []string{"md2wechat"}); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})

	var response struct {
		Success bool   `json:"success"`
		Code    string `json:"code"`
		Data    struct {
			Skill skillDocument `json:"skill"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if !response.Success || response.Code != codeSkillsShown {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response.Data.Skill.Metadata.Name != "md2wechat" || response.Data.Skill.Metadata.Version != "2.0.0" {
		t.Fatalf("unexpected skill metadata: %#v", response.Data.Skill.Metadata)
	}
	if !strings.Contains(response.Data.Skill.Content, "# md2wechat CLI Skill") {
		t.Fatalf("missing embedded skill title:\n%s", response.Data.Skill.Content)
	}
	if !strings.Contains(response.Data.Skill.Content, "md2wechat skills read md2wechat --json") {
		t.Fatalf("missing self-read guidance:\n%s", response.Data.Skill.Content)
	}
}

func TestReadEmbeddedSkillRejectsUnknownSkill(t *testing.T) {
	_, err := readEmbeddedSkill("missing")
	if err == nil {
		t.Fatal("expected error")
	}
	cliErr, ok := extractCLIError(err)
	if !ok {
		t.Fatalf("expected cliError, got %T: %v", err, err)
	}
	if cliErr.Code != codeSkillNotFound {
		t.Fatalf("Code = %q, want %q", cliErr.Code, codeSkillNotFound)
	}
	if cliErr.Details["requested"] != "missing" {
		t.Fatalf("Details = %#v", cliErr.Details)
	}
}
