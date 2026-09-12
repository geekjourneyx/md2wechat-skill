package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSkillsListUsesStableJSONEnvelope(t *testing.T) {
	stdout := captureStdout(t, func() {
		if err := skillsListCmd.RunE(skillsListCmd, nil); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["success"] != true || response["code"] != codeSkillsShown {
		t.Fatalf("unexpected response: %#v", response)
	}
	data, _ := response["data"].(map[string]any)
	if data["count"].(float64) < 1 {
		t.Fatalf("expected at least one embedded skill: %#v", data)
	}
	skills, _ := data["skills"].([]any)
	found := false
	for _, item := range skills {
		skill, _ := item.(map[string]any)
		if skill["name"] == "md2wechat" && skill["description"] != "" {
			found = true
		}
	}
	if !found {
		t.Fatalf("embedded md2wechat skill not found: %#v", skills)
	}
}

func TestSkillsReadRawPrintsCurrentSkillMarkdown(t *testing.T) {
	oldJSON := skillsReadJSON
	t.Cleanup(func() { skillsReadJSON = oldJSON })
	skillsReadJSON = false

	stdout := captureStdout(t, func() {
		if err := skillsReadCmd.RunE(skillsReadCmd, []string{"md2wechat"}); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})

	if !strings.HasPrefix(string(stdout), "---\nname: md2wechat") {
		t.Fatalf("raw skill output = %q", string(stdout))
	}
	if strings.Contains(string(stdout), `"success"`) {
		t.Fatalf("raw skill output must not be wrapped in JSON: %s", stdout)
	}
}

func TestSkillsReadJSONWrapsMarkdownAndGuidance(t *testing.T) {
	oldJSON := skillsReadJSON
	t.Cleanup(func() { skillsReadJSON = oldJSON })
	skillsReadJSON = true

	stdout := captureStdout(t, func() {
		if err := skillsReadCmd.RunE(skillsReadCmd, []string{"md2wechat"}); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})

	var response map[string]any
	if err := json.Unmarshal(stdout, &response); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, stdout)
	}
	if response["success"] != true || response["code"] != codeSkillsRead {
		t.Fatalf("unexpected response: %#v", response)
	}
	data, _ := response["data"].(map[string]any)
	if data["skill"] != "md2wechat" || data["path"] != "SKILL.md" {
		t.Fatalf("unexpected data: %#v", data)
	}
	if !strings.Contains(data["content"].(string), "Discovery First") {
		t.Fatalf("content missing current SOP: %#v", data)
	}
	if !strings.Contains(data["guidance"].(string), "md2wechat skills read md2wechat") {
		t.Fatalf("guidance = %#v", data["guidance"])
	}
}

func TestSkillsReadRejectsTraversal(t *testing.T) {
	if err := skillsReadCmd.RunE(skillsReadCmd, []string{"md2wechat", "../../etc/passwd"}); err == nil || !strings.Contains(err.Error(), "invalid path") {
		t.Fatalf("RunE traversal error = %v", err)
	}
}

// A binary-only host must be able to follow the entire draft workflow offline.
func TestSkillsReadEmbeddedSyncWorkflow(t *testing.T) {
	oldJSON := skillsReadJSON
	t.Cleanup(func() { skillsReadJSON = oldJSON })
	skillsReadJSON = true
	for _, name := range []string{"workflow", "zhihu", "csdn", "toutiao"} {
		t.Run(name, func(t *testing.T) {
			path := "references/sync/" + name + ".md"
			stdout := captureStdout(t, func() {
				if err := skillsReadCmd.RunE(skillsReadCmd, []string{"md2wechat", path}); err != nil {
					t.Fatal(err)
				}
			})
			var response struct {
				Success bool `json:"success"`
				Data    struct {
					Path    string `json:"path"`
					Content string `json:"content"`
				} `json:"data"`
			}
			if err := json.Unmarshal(stdout, &response); err != nil {
				t.Fatal(err)
			}
			if !response.Success || response.Data.Path != path || !strings.HasPrefix(response.Data.Content, "# ") {
				t.Fatalf("embedded workflow unavailable: %s", path)
			}
		})
	}
}
