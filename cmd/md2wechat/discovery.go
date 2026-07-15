package main

import (
	"fmt"
	"sort"
	"strings"

	articleadvise "github.com/geekjourneyx/md2wechat-skill/internal/advise"
	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"github.com/geekjourneyx/md2wechat-skill/internal/converter"
	"github.com/geekjourneyx/md2wechat-skill/internal/image"
	"github.com/geekjourneyx/md2wechat-skill/internal/layoutcatalog"
	"github.com/geekjourneyx/md2wechat-skill/internal/promptcatalog"
	titlebuilder "github.com/geekjourneyx/md2wechat-skill/internal/title"
	"github.com/spf13/cobra"
)

const (
	codeCapabilitiesShown = "CAPABILITIES_SHOWN"
	codeProvidersShown    = "PROVIDERS_SHOWN"
	codeThemesShown       = "THEMES_SHOWN"
	codePromptsShown      = "PROMPTS_SHOWN"
)

type providerView struct {
	Name            string                    `json:"name"`
	Aliases         []string                  `json:"aliases,omitempty"`
	Description     string                    `json:"description"`
	RequiredConfig  []string                  `json:"required_config,omitempty"`
	OptionalConfig  []string                  `json:"optional_config,omitempty"`
	DefaultBaseURL  string                    `json:"default_base_url,omitempty"`
	DefaultModel    string                    `json:"default_model,omitempty"`
	SupportedModels []image.ProviderModelMeta `json:"supported_models,omitempty"`
	SupportsSize    bool                      `json:"supports_size"`
	Current         bool                      `json:"current"`
	Configured      bool                      `json:"configured"`
}

type providerListItem struct {
	Name         string   `json:"name"`
	Aliases      []string `json:"aliases,omitempty"`
	Description  string   `json:"description"`
	SupportsSize bool     `json:"supports_size"`
	Current      bool     `json:"current"`
	Configured   bool     `json:"configured"`
}

type themeView struct {
	Name               string               `json:"name"`
	Type               string               `json:"type"`
	Description        string               `json:"description"`
	Version            string               `json:"version,omitempty"`
	APITheme           string               `json:"api_theme,omitempty"`
	Selectable         bool                 `json:"selectable"`
	MetadataIncomplete bool                 `json:"metadata_incomplete"`
	Style              converter.ThemeStyle `json:"style,omitempty"`
}

type themeListItem struct {
	Name               string `json:"name"`
	Type               string `json:"type"`
	Description        string `json:"description"`
	Version            string `json:"version,omitempty"`
	APITheme           string `json:"api_theme,omitempty"`
	Selectable         bool   `json:"selectable"`
	MetadataIncomplete bool   `json:"metadata_incomplete"`
}

type promptListItem struct {
	Name                    string   `json:"name"`
	Kind                    string   `json:"kind"`
	Description             string   `json:"description"`
	Version                 string   `json:"version"`
	Archetype               string   `json:"archetype,omitempty"`
	PrimaryUseCase          string   `json:"primary_use_case,omitempty"`
	CompatibleUseCases      []string `json:"compatible_use_cases,omitempty"`
	RecommendedAspectRatios []string `json:"recommended_aspect_ratios,omitempty"`
	DefaultAspectRatio      string   `json:"default_aspect_ratio,omitempty"`
	Tags                    []string `json:"tags,omitempty"`
	Variables               []string `json:"variables,omitempty"`
}

var (
	promptKind      string
	promptArchetype string
	promptTag       string
	promptVars      []string
)

var capabilitiesCmd = &cobra.Command{
	Use:   "capabilities",
	Short: "Show machine-readable CLI capabilities",
}

func runCapabilities(cmd *cobra.Command, args []string) error {
	data, err := buildCapabilitiesData()
	if err != nil {
		return wrapCLIError(codeError, err, err.Error())
	}
	responseSuccessWith(codeCapabilitiesShown, "Capabilities shown", data)
	return nil
}

var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "Inspect supported image providers",
}

var providersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List supported image providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		providers, err := buildProviderViews()
		if err != nil {
			return wrapCLIError(codeError, err, err.Error())
		}
		items := make([]providerListItem, 0, len(providers))
		for _, provider := range providers {
			items = append(items, providerToListItem(provider))
		}
		responseSuccessWith(codeProvidersShown, "Providers shown", map[string]any{"providers": items})
		return nil
	},
}

var providersShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show provider details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		providers, err := buildProviderViews()
		if err != nil {
			return wrapCLIError(codeError, err, err.Error())
		}
		for _, provider := range providers {
			if provider.Name == args[0] || contains(provider.Aliases, args[0]) {
				responseSuccessWith(codeProvidersShown, "Provider shown", map[string]any{"provider": provider})
				return nil
			}
		}
		return newCLIError(codeConfigInvalid, fmt.Sprintf("unknown provider: %s", args[0]))
	},
}

var themesCmd = &cobra.Command{
	Use:   "themes",
	Short: "Inspect available convert themes",
}

var themesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available themes",
	RunE: func(cmd *cobra.Command, args []string) error {
		themes, err := listThemeViews()
		if err != nil {
			return wrapCLIError(codeError, err, err.Error())
		}
		items := make([]themeListItem, 0, len(themes))
		for _, theme := range themes {
			items = append(items, themeToListItem(theme))
		}
		responseSuccessWith(codeThemesShown, "Themes shown", map[string]any{"themes": items})
		return nil
	},
}

var themesShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show theme details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tm := converter.NewThemeManager()
		if err := tm.LoadThemes(); err != nil {
			return wrapCLIError(codeError, err, err.Error())
		}
		theme, err := tm.GetTheme(args[0])
		if err != nil {
			return newCLIError(codeConfigInvalid, err.Error())
		}
		responseSuccessWith(codeThemesShown, "Theme shown", map[string]any{"theme": themeToView(*theme)})
		return nil
	},
}

var promptsCmd = &cobra.Command{
	Use:   "prompts",
	Short: "Inspect bundled prompt assets",
}

var promptsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List prompts in the catalog",
	RunE: func(cmd *cobra.Command, args []string) error {
		cat, err := promptcatalog.DefaultCatalog()
		if err != nil {
			return wrapCLIError(codeError, err, err.Error())
		}
		prompts := cat.ListFiltered(promptcatalog.ListFilter{
			Kind:      promptKind,
			Archetype: promptArchetype,
			Tag:       promptTag,
		})
		items := make([]promptListItem, 0, len(prompts))
		for _, prompt := range prompts {
			items = append(items, promptToListItem(prompt))
		}
		responseSuccessWith(codePromptsShown, "Prompts shown", map[string]any{
			"kind":      promptKind,
			"archetype": promptArchetype,
			"tag":       promptTag,
			"prompts":   items,
		})
		return nil
	},
}

var promptsShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show prompt details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cat, err := promptcatalog.DefaultCatalog()
		if err != nil {
			return wrapCLIError(codeError, err, err.Error())
		}
		spec, err := cat.Get(promptKind, args[0])
		if err != nil {
			return newCLIError(codeConfigInvalid, err.Error())
		}
		if promptArchetype != "" && !strings.EqualFold(spec.Archetype, promptArchetype) {
			return newCLIError(codeConfigInvalid, fmt.Sprintf("prompt %s archetype mismatch: expected %s, got %s", spec.Name, promptArchetype, spec.Archetype))
		}
		if promptTag != "" && !contains(spec.Tags, promptTag) {
			return newCLIError(codeConfigInvalid, fmt.Sprintf("prompt %s does not have tag %s", spec.Name, promptTag))
		}
		responseSuccessWith(codePromptsShown, "Prompt shown", map[string]any{"prompt": spec})
		return nil
	},
}

var promptsRenderCmd = &cobra.Command{
	Use:   "render <name>",
	Short: "Render a prompt template with variables",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cat, err := promptcatalog.DefaultCatalog()
		if err != nil {
			return wrapCLIError(codeError, err, err.Error())
		}
		vars, err := parsePromptVars(promptVars)
		if err != nil {
			return newCLIError(codeConfigInvalid, err.Error())
		}
		rendered, spec, err := cat.Render(promptKind, args[0], vars)
		if err != nil {
			return newCLIError(codeConfigInvalid, err.Error())
		}
		responseSuccessWith(codePromptsShown, "Prompt rendered", map[string]any{
			"prompt":   promptToListItem(*spec),
			"vars":     vars,
			"rendered": rendered,
		})
		return nil
	},
}

func init() {
	capabilitiesCmd.RunE = runCapabilities
	providersCmd.AddCommand(providersListCmd, providersShowCmd)
	themesCmd.AddCommand(themesListCmd, themesShowCmd)
	promptsListCmd.Flags().StringVar(&promptKind, "kind", "", "Prompt kind filter")
	promptsListCmd.Flags().StringVar(&promptArchetype, "archetype", "", "Prompt archetype filter")
	promptsListCmd.Flags().StringVar(&promptTag, "tag", "", "Prompt tag filter")
	promptsShowCmd.Flags().StringVar(&promptKind, "kind", "", "Prompt kind")
	promptsShowCmd.Flags().StringVar(&promptArchetype, "archetype", "", "Expected prompt archetype")
	promptsShowCmd.Flags().StringVar(&promptTag, "tag", "", "Expected prompt tag")
	promptsRenderCmd.Flags().StringVar(&promptKind, "kind", "", "Prompt kind")
	promptsRenderCmd.Flags().StringArrayVar(&promptVars, "var", nil, "Prompt variable in KEY=VALUE form")
	promptsCmd.AddCommand(promptsListCmd, promptsShowCmd, promptsRenderCmd)
}

func loadDiscoveryConfig() *config.Config {
	if cfg != nil {
		return cfg
	}
	config.SetQuiet(jsonOutput)
	discoveryCfg, err := config.LoadWithDefaults("")
	if err != nil {
		return &config.Config{
			DefaultConvertMode: "api",
			DefaultTheme:       "default",
			ImageProvider:      "openai",
		}
	}
	return discoveryCfg
}

func buildProviderViews() ([]providerView, error) {
	currentCfg := loadDiscoveryConfig()
	currentProvider := strings.TrimSpace(currentCfg.ImageProvider)
	if currentProvider == "" {
		currentProvider = "openai"
	}
	hasAPIKey := strings.TrimSpace(currentCfg.ImageAPIKey) != ""
	result := make([]providerView, 0, len(image.SupportedProviders()))
	for _, meta := range image.SupportedProviders() {
		current := meta.Name == currentProvider || contains(meta.Aliases, currentProvider)
		result = append(result, providerView{
			Name:            meta.Name,
			Aliases:         meta.Aliases,
			Description:     meta.Description,
			RequiredConfig:  meta.RequiredConfig,
			OptionalConfig:  meta.OptionalConfig,
			DefaultBaseURL:  meta.DefaultBaseURL,
			DefaultModel:    meta.DefaultModel,
			SupportedModels: meta.SupportedModels,
			SupportsSize:    meta.SupportsSize,
			Current:         current,
			Configured:      current && hasAPIKey,
		})
	}
	return result, nil
}

func providerToListItem(provider providerView) providerListItem {
	return providerListItem{
		Name:         provider.Name,
		Aliases:      provider.Aliases,
		Description:  provider.Description,
		SupportsSize: provider.SupportsSize,
		Current:      provider.Current,
		Configured:   provider.Configured,
	}
}

func listThemes() ([]converter.Theme, error) {
	tm := converter.NewThemeManager()
	if err := tm.LoadThemes(); err != nil {
		return nil, err
	}
	return tm.ListThemeDefinitions(), nil
}

func themeToView(theme converter.Theme) themeView {
	return themeView{
		Name:               theme.Name,
		Type:               theme.Type,
		Description:        theme.Description,
		Version:            theme.Version,
		APITheme:           theme.APITheme,
		Selectable:         theme.Selectable(),
		MetadataIncomplete: theme.MetadataIncomplete(),
		Style:              theme.Style,
	}
}

func themeToListItem(theme themeView) themeListItem {
	return themeListItem{
		Name:               theme.Name,
		Type:               theme.Type,
		Description:        theme.Description,
		Version:            theme.Version,
		APITheme:           theme.APITheme,
		Selectable:         theme.Selectable,
		MetadataIncomplete: theme.MetadataIncomplete,
	}
}

func promptToListItem(prompt promptcatalog.PromptSpec) promptListItem {
	return promptListItem{
		Name:                    prompt.Name,
		Kind:                    prompt.Kind,
		Description:             prompt.Description,
		Version:                 prompt.Version,
		Archetype:               prompt.Archetype,
		PrimaryUseCase:          prompt.PrimaryUseCase,
		CompatibleUseCases:      prompt.CompatibleUseCases,
		RecommendedAspectRatios: prompt.RecommendedAspectRatios,
		DefaultAspectRatio:      prompt.DefaultAspectRatio,
		Tags:                    prompt.Tags,
		Variables:               prompt.Variables,
	}
}

func listThemeViews() ([]themeView, error) {
	themes, err := listThemes()
	if err != nil {
		return nil, err
	}
	views := make([]themeView, 0, len(themes))
	for _, theme := range themes {
		views = append(views, themeToView(theme))
	}
	return views, nil
}

func buildCapabilitiesData() (map[string]any, error) {
	providers, err := buildProviderViews()
	if err != nil {
		return nil, err
	}
	themes, err := listThemeViews()
	if err != nil {
		return nil, err
	}
	cat, err := promptcatalog.DefaultCatalog()
	if err != nil {
		return nil, err
	}
	allPrompts := cat.List("")
	currentCfg := loadDiscoveryConfig()
	currentProvider := strings.TrimSpace(currentCfg.ImageProvider)
	if currentProvider == "" {
		currentProvider = "openai"
	}
	currentProviderConfigured := false
	for _, provider := range providers {
		if provider.Current && provider.Configured {
			currentProviderConfigured = true
			break
		}
	}
	defaultTheme := effectiveDefaultTheme(currentCfg)
	return map[string]any{
		"commands": topLevelCommandNames(),
		"convert": map[string]any{
			"default_mode":     "api",
			"supported_modes":  []string{"api", "ai"},
			"font_sizes":       []string{"small", "medium", "large"},
			"background_types": []string{"default", "grid", "none"},
			"default_theme":    defaultTheme,
		},
		"providers": map[string]any{
			"count":              len(providers),
			"current":            currentProvider,
			"current_configured": currentProviderConfigured,
		},
		"image_generation": buildImageGenerationCapabilityData(),
		"title_generation": buildTitleGenerationCapabilityData(),
		"article_advice":   buildArticleAdviceCapabilityData(),
		"themes": map[string]any{
			"count":   len(themes),
			"default": defaultTheme,
		},
		"layout": buildLayoutCapabilityData(),
		"prompts": map[string]any{
			"count":      len(allPrompts),
			"kinds":      sortedPromptKinds(allPrompts),
			"archetypes": sortedPromptArchetypes(allPrompts),
		},
	}, nil
}

func effectiveDefaultTheme(currentCfg *config.Config) string {
	if currentCfg != nil {
		if theme := strings.TrimSpace(currentCfg.DefaultTheme); theme != "" {
			return theme
		}
	}
	return "default"
}

func buildArticleAdviceCapabilityData() map[string]any {
	return map[string]any{
		"available":          true,
		"command":            "advise",
		"requires_json":      true,
		"side_effects":       false,
		"deterministic":      true,
		"max_actions":        3,
		"max_layout_modules": 3,
		"actions_scope":      "command-level recommendations only; micro_edit details are returned in data.micro_edits",
		"tools": []string{
			articleadvise.ToolTitle,
			articleadvise.ToolCover,
			articleadvise.ToolLayout,
			articleadvise.ToolMicroEdit,
		},
		"response_code": codeAdviseCompleted,
	}
}

func buildImageGenerationCapabilityData() map[string]any {
	commands := []string{"generate_image", "generate_cover", "generate_infographic"}
	return map[string]any{
		"direct_provider": map[string]any{
			"available":              true,
			"requires_provider":      true,
			"requires_image_api_key": true,
			"side_effects":           true,
			"commands":               commands,
		},
		"plan_mode": map[string]any{
			"available":              true,
			"requires_provider":      false,
			"requires_image_api_key": false,
			"side_effects":           false,
			"execution_owner":        "host_agent",
			"requires_json":          true,
			"response_code":          codeImagePlanReady,
			"commands":               commands,
		},
	}
}

func buildTitleGenerationCapabilityData() map[string]any {
	return map[string]any{
		"available":                   true,
		"command":                     "title suggest",
		"prompt_kind":                 titlebuilder.PromptKind,
		"default_prompt":              titlebuilder.DefaultPromptName,
		"action":                      "ai_title_suggestion_request",
		"mode":                        "ai_request_host_agent_handoff",
		"execution_owner":             "host_agent",
		"side_effects":                false,
		"requires_external_model":     true,
		"requires_json":               true,
		"requires_provider":           false,
		"requires_image_api_key":      false,
		"requires_wechat_credentials": false,
		"response_code":               codeTitleSuggestRequestReady,
		"candidate_count": map[string]any{
			"min":     titlebuilder.MinCount,
			"max":     titlebuilder.MaxCount,
			"default": titlebuilder.DefaultCount,
		},
		"hook_levels":                     buildTitleHookLevelsCapabilityData(),
		"default_hook_level":              titlebuilder.DefaultHookLevel,
		"max_recommended_hook_level":      2,
		"level_3_requires_evidence_basis": true,
		"default_max_title_chars":         titlebuilder.DefaultMaxTitleChars,
		"metadata_title_max_chars":        titlebuilder.MetadataTitleMaxChars,
		"recommendation_only":             true,
	}
}

func buildTitleHookLevelsCapabilityData() []map[string]any {
	return []map[string]any{
		{
			"level":       1,
			"label":       titlebuilder.HookLevelLabel(1),
			"description": "Restrained factual titles that preserve the current low-risk behavior.",
		},
		{
			"level":       2,
			"label":       titlebuilder.HookLevelLabel(2),
			"description": "Punchier factual hooks with stronger contrast, entities, numbers, and consequence framing when supported.",
		},
		{
			"level":       3,
			"label":       titlebuilder.HookLevelLabel(3),
			"description": "High-tension factual hooks that require evidence_basis and risk_flags on every candidate.",
		},
	}
}

func buildLayoutCapabilityData() map[string]any {
	catalog, err := layoutcatalog.DefaultCatalog()
	if err != nil {
		return map[string]any{
			"available": false,
			"error":     err.Error(),
		}
	}
	modules := catalog.ListFiltered(layoutcatalog.ListFilter{})
	compatibilityModules := catalog.ListFiltered(layoutcatalog.ListFilter{Lifecycle: layoutcatalog.LifecycleCompatibility})
	categorySet := map[string]struct{}{}
	for _, module := range modules {
		categorySet[module.Category] = struct{}{}
	}
	categories := make([]string, 0, len(categorySet))
	for category := range categorySet {
		categories = append(categories, category)
	}
	sort.Strings(categories)

	return map[string]any{
		"available":                  true,
		"module_count":               len(modules),
		"recommended_syntax_count":   len(modules),
		"recommended_scenario_count": recommendedLayoutScenarioCount,
		"compatibility_module_count": len(compatibilityModules),
		"base_enhancement_count":     baseLayoutEnhancementCount,
		"render_syntax_count":        renderLayoutSyntaxCount,
		"supports_validate":          true,
		"api_mode_only":              true,
		"schema_version":             layoutcatalog.SchemaVersion,
		"serves":                     []string{"attention", "readability", "memorability", "conversion"},
		"categories":                 categories,
	}
}

func parsePromptVars(items []string) (map[string]string, error) {
	vars := make(map[string]string, len(items))
	for _, item := range items {
		key, value, ok := strings.Cut(item, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("invalid --var %q, expected KEY=VALUE", item)
		}
		vars[strings.TrimSpace(key)] = value
	}
	return vars, nil
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(item, target) {
			return true
		}
	}
	return false
}

func sortedPromptKinds(prompts []promptcatalog.PromptSpec) []string {
	set := map[string]struct{}{}
	for _, prompt := range prompts {
		set[prompt.Kind] = struct{}{}
	}
	kinds := make([]string, 0, len(set))
	for kind := range set {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return kinds
}

func sortedPromptArchetypes(prompts []promptcatalog.PromptSpec) []string {
	set := map[string]struct{}{}
	for _, prompt := range prompts {
		if prompt.Archetype == "" {
			continue
		}
		set[prompt.Archetype] = struct{}{}
	}
	archetypes := make([]string, 0, len(set))
	for archetype := range set {
		archetypes = append(archetypes, archetype)
	}
	sort.Strings(archetypes)
	return archetypes
}
