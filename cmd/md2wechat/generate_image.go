package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"github.com/geekjourneyx/md2wechat-skill/internal/converter"
	"github.com/geekjourneyx/md2wechat-skill/internal/image"
	"github.com/geekjourneyx/md2wechat-skill/internal/promptcatalog"
)

var (
	generateImageCmdSize     string
	generateImageCmdModel    string
	generateImageCmdPreset   string
	generateImageCmdArticle  string
	generateImageCmdTitle    string
	generateImageCmdSummary  string
	generateImageCmdKeywords string
	generateImageCmdStyle    string
	generateImageCmdAspect   string
	generateImageCmdSubject  string
	generateImageCmdPlan     bool
)

type generateImageInput struct {
	Command           string
	Plan              bool
	RawPrompt         string
	Preset            string
	Article           string
	Title             string
	Summary           string
	Keywords          string
	Style             string
	Aspect            string
	Size              string
	Model             string
	SubjectReference  string
	RequiredArchetype string
}

type generateImageContext struct {
	Title     string
	Summary   string
	Keywords  string
	KeyPoints string
}

type generateImagePromptResolution struct {
	Prompt string
	Spec   *promptcatalog.PromptSpec
	Ctx    *generateImageContext
	Style  string
	Aspect string
}

type generateImagePlan struct {
	Mode                    string   `json:"mode"`
	Command                 string   `json:"command"`
	ExecutionOwner          string   `json:"execution_owner"`
	SideEffects             bool     `json:"side_effects"`
	RequiresProvider        bool     `json:"requires_provider"`
	RequiresImageAPIKey     bool     `json:"requires_image_api_key"`
	Prompt                  string   `json:"prompt"`
	RawPrompt               string   `json:"raw_prompt"`
	Preset                  string   `json:"preset"`
	Archetype               string   `json:"archetype"`
	PrimaryUseCase          string   `json:"primary_use_case"`
	CompatibleUseCases      []string `json:"compatible_use_cases"`
	RecommendedAspectRatios []string `json:"recommended_aspect_ratios"`
	DefaultAspectRatio      string   `json:"default_aspect_ratio"`
	Article                 string   `json:"article"`
	Title                   string   `json:"title"`
	Summary                 string   `json:"summary"`
	Keywords                string   `json:"keywords"`
	Style                   string   `json:"style"`
	Aspect                  string   `json:"aspect"`
	Size                    string   `json:"size"`
	ModelHint               string   `json:"model_hint"`
	SubjectReference        string   `json:"subject_reference,omitempty"`
	SuggestedFilename       string   `json:"suggested_filename"`
	AltText                 string   `json:"alt_text"`
}

func runGenerateImage(args []string) error {
	input := generateImageInput{
		Command:          "generate_image",
		Plan:             generateImageCmdPlan,
		Preset:           generateImageCmdPreset,
		Article:          generateImageCmdArticle,
		Title:            generateImageCmdTitle,
		Summary:          generateImageCmdSummary,
		Keywords:         generateImageCmdKeywords,
		Style:            generateImageCmdStyle,
		Aspect:           generateImageCmdAspect,
		Size:             generateImageCmdSize,
		Model:            generateImageCmdModel,
		SubjectReference: generateImageCmdSubject,
	}
	if len(args) > 0 {
		input.RawPrompt = args[0]
	}
	return runGenerateImageWithInput(input)
}

func runGeneratePresetImage(archetype, defaultPreset string, input generateImageInput) error {
	if strings.TrimSpace(input.Command) == "" {
		input.Command = "generate_" + archetype
	}
	input.RequiredArchetype = archetype
	if strings.TrimSpace(input.Preset) == "" {
		input.Preset = defaultPreset
	}
	return runGenerateImageWithInput(input)
}

func runGenerateImageWithInput(input generateImageInput) error {
	if input.Plan {
		return runGenerateImagePlan(input)
	}

	if err := prepareWeChatSideEffect(); err != nil {
		return err
	}
	if cfg.ImageAPIKey == "" {
		err := &config.ConfigError{Field: "ImageAPIKey", Message: "IMAGE_API_KEY is required for image generation"}
		return wrapCLIError(codeConfigInvalid, err, err.Error())
	}

	prompt, err := resolveGenerateImagePrompt(input)
	if err != nil {
		return newCLIError(codeConfigInvalid, err.Error())
	}

	subjectReference := strings.TrimSpace(input.SubjectReference)
	if subjectReference != "" {
		if err := ensureSubjectReferenceSupported(input.Model, subjectReference); err != nil {
			return err
		}
	}

	processor := resolveImageProcessor(input.Model)
	if subjectReference != "" {
		if strings.TrimSpace(input.Size) != "" {
			cfgCopy := *cfg
			cfgCopy.ImageSize = strings.TrimSpace(input.Size)
			if strings.TrimSpace(input.Model) != "" {
				cfgCopy.ImageModel = strings.TrimSpace(input.Model)
			}
			processor = newImageProcessorWithConfig(&cfgCopy)
		}
		subjectProcessor, ok := processor.(subjectReferenceImageProcessor)
		if !ok {
			return newCLIError(codeConfigInvalid, "configured image processor does not support --subject-reference")
		}
		result, err := subjectProcessor.GenerateAndUploadWithSubject(prompt, subjectReference)
		if err != nil {
			return wrapCLIError(codeImageGenerateFailed, err, err.Error())
		}
		responseSuccess(result)
		return nil
	}
	if input.Size != "" {
		result, err := processor.GenerateAndUploadWithSize(prompt, input.Size)
		if err != nil {
			return wrapCLIError(codeImageGenerateFailed, err, err.Error())
		}
		responseSuccess(result)
		return nil
	}

	result, err := processor.GenerateAndUpload(prompt)
	if err != nil {
		return wrapCLIError(codeImageGenerateFailed, err, err.Error())
	}
	responseSuccess(result)
	return nil
}

func resolveImageProcessor(model string) imageProcessor {
	model = strings.TrimSpace(model)
	if model == "" {
		return newImageProcessor()
	}

	cfgCopy := *cfg
	cfgCopy.ImageModel = model
	return newImageProcessorWithConfig(&cfgCopy)
}

// resolveImageProviderName returns the configured image provider, falling back
// to the same default the provider factory uses.
func resolveImageProviderName() string {
	provider := strings.TrimSpace(cfg.ImageProvider)
	if provider == "" {
		provider = "openai"
	}
	return provider
}

// resolveImageModelName returns the model this command will actually use,
// honouring --model first, then the config, then the provider default.
func resolveImageModelName(provider, override string) string {
	if model := strings.TrimSpace(override); model != "" {
		return model
	}
	if model := strings.TrimSpace(cfg.ImageModel); model != "" {
		return model
	}
	return image.DefaultProviderModel(provider)
}

// ensureSubjectReferenceSupported rejects --subject-reference from provider
// metadata. A runtime interface assertion cannot do this because the concrete
// processor always implements the subject-reference method regardless of the
// configured provider, which would defer the failure to the generate call.
func ensureSubjectReferenceSupported(modelOverride, subjectReference string) error {
	if err := image.ValidateSubjectReferenceURL(subjectReference); err != nil {
		return newCLIError(codeConfigInvalid, err.Error())
	}

	provider := resolveImageProviderName()
	if !image.ProviderSupportsSubjectReference(provider) {
		message := fmt.Sprintf("image provider %s does not support --subject-reference", provider)
		if supported := image.SubjectReferenceProviderNames(); len(supported) > 0 {
			message += "; supported providers: " + strings.Join(supported, ", ")
		}
		return newCLIError(codeConfigInvalid, message)
	}

	model := resolveImageModelName(provider, modelOverride)
	if !image.ModelSupportsSubjectReference(provider, model) {
		message := fmt.Sprintf("image model %s does not support --subject-reference", model)
		if hint := image.SubjectReferenceModelsHint(provider); hint != "" {
			message += "; " + hint
		}
		return newCLIError(codeConfigInvalid, message)
	}
	return nil
}

func resolveGenerateImagePrompt(input generateImageInput) (string, error) {
	resolved, err := resolveGenerateImagePromptDetails(input)
	if err != nil {
		return "", err
	}
	return resolved.Prompt, nil
}

func resolveGenerateImagePromptDetails(input generateImageInput) (*generateImagePromptResolution, error) {
	if strings.TrimSpace(input.Preset) == "" {
		if strings.TrimSpace(input.RawPrompt) == "" {
			return nil, fmt.Errorf("generate_image requires a prompt or --preset")
		}
		return &generateImagePromptResolution{Prompt: input.RawPrompt}, nil
	}

	if strings.TrimSpace(input.RawPrompt) != "" {
		return nil, fmt.Errorf("do not pass a raw prompt when --preset is used")
	}

	cat, err := promptcatalog.DefaultCatalog()
	if err != nil {
		return nil, err
	}
	spec, err := cat.Get("image", input.Preset)
	if err != nil {
		return nil, err
	}
	if input.RequiredArchetype != "" && !promptcatalog.SupportsUseCase(spec, input.RequiredArchetype) {
		return nil, fmt.Errorf("preset %s is %s/%s, expected %s", spec.Name, spec.Archetype, spec.PrimaryUseCase, input.RequiredArchetype)
	}

	ctx, err := buildGenerateImageContext(input)
	if err != nil {
		return nil, err
	}

	style := defaultString(input.Style, defaultVisualStyle(spec.Archetype))
	aspect := defaultString(input.Aspect, spec.DefaultAspectRatio, defaultAspectRatio(spec.Archetype))
	rendered, _, err := cat.Render("image", input.Preset, map[string]string{
		"ARTICLE_TITLE":   ctx.Title,
		"ARTICLE_SUMMARY": ctx.Summary,
		"KEYWORDS":        ctx.Keywords,
		"KEY_POINTS":      ctx.KeyPoints,
		"VISUAL_STYLE":    style,
		"ASPECT_RATIO":    aspect,
	})
	if err != nil {
		return nil, err
	}
	return &generateImagePromptResolution{
		Prompt: rendered,
		Spec:   spec,
		Ctx:    ctx,
		Style:  style,
		Aspect: aspect,
	}, nil
}

func runGenerateImagePlan(input generateImageInput) error {
	if !jsonOutput {
		return newCLIError(codeConfigInvalid, "--plan requires --json")
	}

	resolved, err := resolveGenerateImagePromptDetails(input)
	if err != nil {
		return newCLIError(codeConfigInvalid, err.Error())
	}

	responseActionRequiredWith(codeImagePlanReady, "Image plan ready; generate the image with a host Agent or configured provider.", buildGenerateImagePlan(input, resolved))
	return nil
}

func buildGenerateImagePlan(input generateImageInput, resolved *generateImagePromptResolution) generateImagePlan {
	command := strings.TrimSpace(input.Command)
	if command == "" {
		command = "generate_image"
	}

	plan := generateImagePlan{
		Mode:                    "plan",
		Command:                 command,
		ExecutionOwner:          "host_agent",
		SideEffects:             false,
		RequiresProvider:        false,
		RequiresImageAPIKey:     false,
		Prompt:                  resolved.Prompt,
		RawPrompt:               strings.TrimSpace(input.RawPrompt),
		Article:                 strings.TrimSpace(input.Article),
		Style:                   strings.TrimSpace(resolved.Style),
		Aspect:                  strings.TrimSpace(resolved.Aspect),
		Size:                    strings.TrimSpace(input.Size),
		ModelHint:               strings.TrimSpace(input.Model),
		SubjectReference:        strings.TrimSpace(input.SubjectReference),
		CompatibleUseCases:      []string{},
		RecommendedAspectRatios: []string{},
	}
	if resolved.Ctx != nil {
		plan.Title = strings.TrimSpace(resolved.Ctx.Title)
		plan.Summary = strings.TrimSpace(resolved.Ctx.Summary)
		plan.Keywords = strings.TrimSpace(resolved.Ctx.Keywords)
	}
	if resolved.Spec != nil {
		plan.Preset = resolved.Spec.Name
		plan.Archetype = resolved.Spec.Archetype
		plan.PrimaryUseCase = resolved.Spec.PrimaryUseCase
		plan.CompatibleUseCases = nonNilStringSlice(resolved.Spec.CompatibleUseCases)
		plan.RecommendedAspectRatios = nonNilStringSlice(resolved.Spec.RecommendedAspectRatios)
		plan.DefaultAspectRatio = defaultString(resolved.Spec.DefaultAspectRatio, defaultAspectRatio(resolved.Spec.Archetype))
	}
	plan.SuggestedFilename = suggestedImagePlanFilename(command, plan.Title, plan.RawPrompt)
	plan.AltText = suggestedImagePlanAltText(plan.Title, plan.Summary, plan.RawPrompt)
	return plan
}

func suggestedImagePlanFilename(command, title, rawPrompt string) string {
	base := firstNonEmptyString(title, rawPrompt, command, "image-plan")
	slug := slugifyASCII(base)
	if slug == "" {
		slug = "image-plan"
	}
	return slug + ".png"
}

func suggestedImagePlanAltText(title, summary, rawPrompt string) string {
	return firstNonEmptyString(title, summary, rawPrompt, "Generated image")
}

func nonNilStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

func slugifyASCII(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && builder.Len() > 0 {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func buildGenerateImageContext(input generateImageInput) (*generateImageContext, error) {
	ctx := &generateImageContext{
		Title:    strings.TrimSpace(input.Title),
		Summary:  strings.TrimSpace(input.Summary),
		Keywords: strings.TrimSpace(input.Keywords),
	}

	if input.Article != "" {
		markdown, err := os.ReadFile(input.Article)
		if err != nil {
			return nil, fmt.Errorf("read article: %w", err)
		}
		meta := converter.ParseArticleMetadata(string(markdown))
		if ctx.Title == "" {
			ctx.Title = strings.TrimSpace(meta.Title)
		}
		if ctx.Summary == "" {
			ctx.Summary = firstNonEmptyString(strings.TrimSpace(meta.Digest), deriveMarkdownSummary(string(markdown)))
		}
	}

	if ctx.Title == "" && ctx.Summary == "" {
		return nil, fmt.Errorf("--preset requires --article, --title, or --summary")
	}

	if ctx.Keywords == "" {
		ctx.Keywords = deriveKeywords(ctx.Title, ctx.Summary)
	}
	ctx.KeyPoints = firstNonEmptyString(ctx.Summary, ctx.Keywords, ctx.Title)
	return ctx, nil
}

func deriveMarkdownSummary(markdown string) string {
	normalized := strings.ReplaceAll(markdown, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	var body []string
	inFrontMatter := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if i == 0 && trimmed == "---" {
			inFrontMatter = true
			continue
		}
		if inFrontMatter {
			if trimmed == "---" {
				inFrontMatter = false
			}
			continue
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "![") {
			continue
		}
		body = append(body, cleanedMarkdownLine(trimmed))
		if len(strings.Join(body, " ")) >= 140 {
			break
		}
	}

	return strings.TrimSpace(strings.Join(body, " "))
}

func cleanedMarkdownLine(line string) string {
	replacer := strings.NewReplacer("**", "", "__", "", "*", "", "`", "", ">", "", "-", "", "|", " ")
	return strings.Join(strings.Fields(replacer.Replace(line)), " ")
}

func deriveKeywords(values ...string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		parts = append(parts, value)
	}
	return strings.Join(parts, "；")
}

func defaultVisualStyle(archetype string) string {
	switch strings.ToLower(strings.TrimSpace(archetype)) {
	case "infographic":
		return "clear information design"
	case "cover":
		return "editorial clean"
	default:
		return "clean visual style"
	}
}

func defaultAspectRatio(archetype string) string {
	switch strings.ToLower(strings.TrimSpace(archetype)) {
	case "infographic":
		return "3:4"
	case "cover":
		return "16:9"
	default:
		return "1:1"
	}
}

func defaultString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
