package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const (
	codeFactsExported        = "FACTS_EXPORTED"
	runtimeFactsSchemaVersion = "v1"
)

// SourceCommit is injected at build time for release binaries.
var SourceCommit = "unknown"

type runtimeFacts struct {
	SchemaVersion string                  `json:"schema_version"`
	Version       string                  `json:"version"`
	SourceCommit  string                  `json:"source_commit"`
	Counts        runtimeFactsCounts      `json:"counts"`
	SideEffects   runtimeFactsSideEffects `json:"side_effects"`
}

type runtimeFactsCounts struct {
	APIThemes            int `json:"api_themes"`
	RecommendedScenarios int `json:"recommended_scenarios"`
	RecommendedSyntax    int `json:"recommended_syntax"`
	RenderSyntax         int `json:"render_syntax"`
}

type runtimeFactsSideEffects struct {
	Convert     bool `json:"convert"`
	CreateDraft bool `json:"create_draft"`
}

func newFactsCommand() *cobra.Command {
	factsCmd := &cobra.Command{
		Use:   "facts",
		Short: "Inspect release runtime facts",
	}
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export the release runtime facts contract",
		Args:  cobra.NoArgs,
		RunE:  runFactsExport,
	}
	factsCmd.AddCommand(exportCmd)
	return factsCmd
}

func runFactsExport(cmd *cobra.Command, args []string) error {
	data, err := buildRuntimeFacts()
	if err != nil {
		return wrapCLIError(codeError, err, "build runtime facts")
	}
	responseSuccessWith(codeFactsExported, "Runtime facts exported", data)
	return nil
}

func buildRuntimeFacts() (runtimeFacts, error) {
	themes, err := listThemeViews()
	if err != nil {
		return runtimeFacts{}, fmt.Errorf("list themes: %w", err)
	}
	apiThemeCount := 0
	for _, theme := range themes {
		if theme.Type == "api" && theme.Selectable {
			apiThemeCount++
		}
	}

	layout := buildLayoutCapabilityData()
	available, ok := layout["available"].(bool)
	if !ok || !available {
		if message, ok := layout["error"].(string); ok && message != "" {
			return runtimeFacts{}, fmt.Errorf("load layout catalog: %s", message)
		}
		return runtimeFacts{}, fmt.Errorf("load layout catalog")
	}

	recommendedScenarios, err := runtimeFactsCount(layout, "recommended_scenario_count")
	if err != nil {
		return runtimeFacts{}, err
	}
	recommendedSyntax, err := runtimeFactsCount(layout, "recommended_syntax_count")
	if err != nil {
		return runtimeFacts{}, err
	}
	renderSyntax, err := runtimeFactsCount(layout, "render_syntax_count")
	if err != nil {
		return runtimeFacts{}, err
	}

	sourceCommit := strings.TrimSpace(SourceCommit)
	if sourceCommit == "" {
		sourceCommit = "unknown"
	}

	return runtimeFacts{
		SchemaVersion: runtimeFactsSchemaVersion,
		Version:       Version,
		SourceCommit:  sourceCommit,
		Counts: runtimeFactsCounts{
			APIThemes:            apiThemeCount,
			RecommendedScenarios: recommendedScenarios,
			RecommendedSyntax:    recommendedSyntax,
			RenderSyntax:         renderSyntax,
		},
		SideEffects: runtimeFactsSideEffects{
			Convert:     false,
			CreateDraft: true,
		},
	}, nil
}

func runtimeFactsCount(layout map[string]any, key string) (int, error) {
	value, ok := layout[key].(int)
	if !ok {
		return 0, fmt.Errorf("layout discovery %q is missing or invalid", key)
	}
	return value, nil
}
