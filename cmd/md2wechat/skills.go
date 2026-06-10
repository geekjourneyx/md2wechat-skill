package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/geekjourneyx/md2wechat-skill/internal/assets"
	"github.com/spf13/cobra"
)

const (
	codeSkillsShown   = "SKILLS_SHOWN"
	codeSkillNotFound = "SKILL_NOT_FOUND"
)

type skillSummary struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	RiskLevel   string   `json:"risk_level"`
	SideEffects []string `json:"side_effects"`
	Source      string   `json:"source"`
	ContentHash string   `json:"content_hash"`
	ReadCommand string   `json:"read_command"`
	Commands    []string `json:"commands"`
}

type skillDocument struct {
	Metadata skillSummary `json:"metadata"`
	Content  string       `json:"content"`
}

var embeddedSkillDefaults = map[string]skillSummary{
	"md2wechat": {
		Name:        "md2wechat",
		Description: "Convert Markdown to WeChat Official Account HTML and drafts with discovery-first CLI commands.",
		RiskLevel:   "side-effect-aware",
		SideEffects: []string{
			"read-only discovery for version, capabilities, skills, providers, themes, prompts, layout, doctor, and inspect",
			"local file writes only when output, preview, conversion, writing, humanize, or image generation commands request artifacts",
			"network or WeChat side effects only for image providers, uploads, draft creation, or explicit publish flags",
		},
		Source: "embedded",
		Commands: []string{
			"skills list",
			"skills read md2wechat",
			"capabilities",
			"inspect",
			"preview",
			"convert",
		},
	},
}

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Inspect embedded agent skills",
}

var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List embedded agent skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		skills, err := listEmbeddedSkillSummaries()
		if err != nil {
			return wrapCLIError(codeError, err, err.Error())
		}
		responseSuccessWith(codeSkillsShown, "Skills shown", map[string]any{"skills": skills})
		return nil
	},
}

var skillsReadCmd = &cobra.Command{
	Use:   "read <name>",
	Short: "Read an embedded agent skill",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		skill, err := readEmbeddedSkill(args[0])
		if err != nil {
			return err
		}
		responseSuccessWith(codeSkillsShown, "Skill shown", map[string]any{"skill": skill})
		return nil
	},
}

func init() {
	skillsCmd.AddCommand(skillsListCmd, skillsReadCmd)
}

func listEmbeddedSkillSummaries() ([]skillSummary, error) {
	names, err := assets.ListBuiltinSkills()
	if err != nil {
		return nil, err
	}

	skills := make([]skillSummary, 0, len(names))
	for _, name := range names {
		content, err := assets.ReadBuiltinSkill(name)
		if err != nil {
			return nil, err
		}
		skills = append(skills, buildSkillSummary(name, content))
	}
	return skills, nil
}

func readEmbeddedSkill(name string) (skillDocument, error) {
	resolved, err := resolveEmbeddedSkillName(name)
	if err != nil {
		return skillDocument{}, err
	}

	content, err := assets.ReadBuiltinSkill(resolved)
	if err != nil {
		return skillDocument{}, wrapCLIError(codeError, err, err.Error())
	}

	return skillDocument{
		Metadata: buildSkillSummary(resolved, content),
		Content:  string(content),
	}, nil
}

func resolveEmbeddedSkillName(name string) (string, error) {
	names, err := assets.ListBuiltinSkills()
	if err != nil {
		return "", wrapCLIError(codeError, err, err.Error())
	}
	for _, candidate := range names {
		if strings.EqualFold(candidate, name) {
			return candidate, nil
		}
	}
	return "", newCLIErrorWithDetails(
		codeSkillNotFound,
		fmt.Sprintf("unknown embedded skill: %s", name),
		map[string]any{"requested": name, "available": names},
		[]string{"Run md2wechat skills list --json and choose a listed skill name."},
	)
}

func buildSkillSummary(name string, content []byte) skillSummary {
	summary, ok := embeddedSkillDefaults[name]
	if !ok {
		summary = skillSummary{
			Name:        name,
			Description: "Embedded agent skill",
			RiskLevel:   "unknown",
			Source:      "embedded",
		}
	}
	hash := sha256.Sum256(content)
	summary.Version = Version
	summary.ContentHash = "sha256:" + hex.EncodeToString(hash[:])
	summary.ReadCommand = fmt.Sprintf("md2wechat skills read %s --json", name)
	return summary
}
