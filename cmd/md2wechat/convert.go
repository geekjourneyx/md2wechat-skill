package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/geekjourneyx/md2wechat-skill/internal/atomicfile"
	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"github.com/geekjourneyx/md2wechat-skill/internal/converter"
	"github.com/geekjourneyx/md2wechat-skill/internal/draft"
	"github.com/geekjourneyx/md2wechat-skill/internal/image"
	"github.com/geekjourneyx/md2wechat-skill/internal/publish"
	"github.com/geekjourneyx/md2wechat-skill/internal/wechat"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type imageProcessor interface {
	UploadLocalImage(filePath string) (*image.UploadResult, error)
	DownloadAndUpload(url string) (*image.UploadResult, error)
	GenerateAndUpload(prompt string) (*image.GenerateAndUploadResult, error)
	GenerateAndUploadWithSize(prompt string, size string) (*image.GenerateAndUploadResult, error)
}

type subjectReferenceImageProcessor interface {
	GenerateAndUploadWithSubject(prompt, subjectReference string) (*image.GenerateAndUploadResult, error)
}

var (
	newMarkdownConverter = func() converter.Converter {
		return converter.NewConverter(cfg, log)
	}
	newImageProcessor = func() imageProcessor {
		return newRuntimeImageProcessor()
	}
	newImageProcessorWithConfig = func(runtimeCfg *config.Config) imageProcessor {
		return newRuntimeImageProcessorWithConfig(runtimeCfg)
	}
	newDraftCreator = func() publish.DraftCreator {
		return draft.NewArtifactDraftCreator(cfg, log)
	}
	uploadCoverImageFn = uploadCoverImage
	newPublishService  = func() *publish.Service {
		return publish.NewService(log, newMarkdownConverter(), newImageProcessor(), newDraftCreator(), uploadCoverImageFn)
	}
)

// convertCmd convert 命令
var convertCmd = &cobra.Command{
	Use:   "convert <markdown_file>",
	Short: "Convert Markdown to WeChat HTML",
	Long: `Convert Markdown article to WeChat Official Account formatted HTML.

Supports two conversion modes:
  - api: Use md2wechat API (stable, requires API key)
  - ai:  Prepare an action_required prompt for a host Agent or external model.
         md2wechat does not call a model, generate HTML, upload assets, or create a draft in this mode.

Theme availability is resolved at runtime. Discover the current catalog with:
  md2wechat themes list --json

Inspect one full theme definition with:
  md2wechat themes show <name> --json`,
	Args: cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConvert(cmd, args)
	},
}

// convert 命令参数
var (
	convertMode           string
	convertTheme          string
	convertAPIKey         string
	convertFontSize       string
	convertBackgroundType string
	convertCustomPrompt   string
	convertOutput         string
	convertPreview        bool
	convertUpload         bool
	convertDraft          bool
	convertSaveDraft      string
	convertCoverImage     string // 封面图片路径
	convertCoverMediaID   string // 已存在的微信封面素材 media_id
	convertTitle          string
	convertAuthor         string
	convertDigest         string
)

func init() {
	// 添加 flags
	convertCmd.Flags().StringVar(&convertMode, "mode", "api", "Conversion mode: api or ai")
	convertCmd.Flags().StringVar(&convertTheme, "theme", "default", "Theme name")
	convertCmd.Flags().StringVar(&convertAPIKey, "api-key", "", "API key for md2wechat.cn")
	convertCmd.Flags().StringVar(&convertFontSize, "font-size", "medium", "Font size: small/medium/large (API mode only)")
	convertCmd.Flags().StringVar(&convertBackgroundType, "background-type", "none", "Background type: default/grid/none (API mode only)")
	convertCmd.Flags().StringVar(&convertCustomPrompt, "custom-prompt", "", "Custom AI prompt (AI mode only)")
	convertCmd.Flags().StringVarP(&convertOutput, "output", "o", "", "Output HTML file path")
	convertCmd.Flags().BoolVar(&convertPreview, "preview", false, "Preview only, do not upload images")
	convertCmd.Flags().BoolVar(&convertUpload, "upload", false, "Upload images to WeChat and replace URLs")
	convertCmd.Flags().BoolVar(&convertDraft, "draft", false, "Create WeChat draft after conversion")
	convertCmd.Flags().StringVar(&convertSaveDraft, "save-draft", "", "Save draft JSON to file")
	convertCmd.Flags().StringVar(&convertCoverImage, "cover", "", "Cover image path for draft (required when using --draft)")
	convertCmd.Flags().StringVar(&convertCoverMediaID, "cover-media-id", "", "Existing WeChat cover media_id for draft (mutually exclusive with --cover)")
	convertCmd.Flags().StringVar(&convertTitle, "title", "", "Override article title (max 32 characters)")
	convertCmd.Flags().StringVar(&convertAuthor, "author", "", "Override article author (max 16 characters)")
	convertCmd.Flags().StringVar(&convertDigest, "digest", "", "Override article digest (max 128 characters)")
}

// runConvert 执行转换
func runConvert(cmd *cobra.Command, args []string) error {
	markdownFile := args[0]
	applyEffectiveCommandTheme(cmd, &convertTheme)

	if err := validateConvertConfig(); err != nil {
		return err
	}

	log.Info("starting conversion",
		zap.String("file", markdownFile),
		zap.String("mode", convertMode),
		zap.String("theme", convertTheme))

	// 读取 Markdown 文件
	markdown, err := os.ReadFile(markdownFile)
	if err != nil {
		return wrapCLIError(codeConvertReadFailed, err, fmt.Sprintf("read markdown file: %v", err))
	}

	document := converter.ParseArticleDocument(string(markdown))
	metadata := document.Metadata
	bodyMarkdown := document.Body
	resolvedTitle := firstNonEmptyTrimmed(convertTitle, metadata.Title)
	resolvedAuthor := firstNonEmptyTrimmed(convertAuthor, metadata.Author)
	resolvedDigest := firstNonEmptyTrimmed(convertDigest, metadata.Digest)
	if err := validateConvertMetadata(resolvedTitle, resolvedAuthor, resolvedDigest); err != nil {
		return err
	}

	service := newPublishService()
	input := &publish.ConvertInput{
		Source: publish.ArticleSource{
			Path:     markdownFile,
			Markdown: bodyMarkdown,
			Metadata: publish.Metadata{
				Title:  resolvedTitle,
				Author: resolvedAuthor,
				Digest: resolvedDigest,
			},
		},
		Intent: publish.PublishIntent{
			Mode:        convertMode,
			Preview:     convertPreview,
			Upload:      convertUpload,
			CreateDraft: convertDraft,
			SaveDraft:   convertSaveDraft != "",
		},
		ConvertRequest: &converter.ConvertRequest{
			Markdown: bodyMarkdown,
			Metadata: converter.ArticleMetadata{
				Title:  resolvedTitle,
				Author: resolvedAuthor,
				Digest: resolvedDigest,
			},
			Mode:           converter.ConvertMode(convertMode),
			Theme:          convertTheme,
			APIKey:         convertAPIKey,
			FontSize:       convertFontSize,
			BackgroundType: convertBackgroundType,
			CustomPrompt:   convertCustomPrompt,
		},
		MarkdownDir:    filepath.Dir(markdownFile),
		OutputFile:     convertOutput,
		SaveDraftPath:  convertSaveDraft,
		CoverImagePath: convertCoverImage,
		CoverMediaID:   strings.TrimSpace(convertCoverMediaID),
	}

	if err := service.Preflight(input); err != nil {
		return mapConvertServiceError(err)
	}
	if convertUpload || convertDraft {
		if err := prepareWeChatSideEffectWithAPIKey(convertAPIKey); err != nil {
			return err
		}
	}

	output, err := service.Convert(input)
	if err != nil {
		return mapConvertServiceError(err)
	}
	result := output.Conversion
	if result == nil {
		return newCLIError(codeConvertFailed, "conversion returned no result")
	}

	// AI 模式返回的是待外部执行的请求，不应被当作失败路径拦截
	if convertMode == "ai" && converter.IsAIRequest(result) {
		return handleAIResult(result, markdownFile)
	}

	log.Info("conversion completed",
		zap.String("mode", string(result.Mode)),
		zap.String("theme", result.Theme),
		zap.Int("image_count", len(output.Artifact.Assets)))

	if jsonOutput {
		responseSuccessWith(codeConvertCompleted, "Conversion completed", map[string]any{
			"mode":        string(result.Mode),
			"theme":       result.Theme,
			"html":        output.Artifact.HTML,
			"image_count": len(output.Artifact.Assets),
			"assets":      output.Artifact.Assets,
			"output_file": output.Artifact.OutputFile,
			"preview":     convertPreview,
			"upload":      convertUpload,
			"draft":       convertDraft,
			"save_draft":  output.DraftSaved,
			"title":       output.Artifact.Metadata.Title,
			"author":      output.Artifact.Metadata.Author,
			"digest":      output.Artifact.Metadata.Digest,
			"draft_id":    output.Artifact.DraftMediaID,
			"draft_url":   output.Artifact.DraftURL,
			"cover_id":    output.Artifact.CoverMediaID,
		})
		return nil
	}

	// 输出 HTML
	if err := outputHTML(output.Artifact.HTML, "", convertPreview); err != nil {
		return wrapCLIError(codeConvertFailed, err, err.Error())
	}

	return nil
}

func applyEffectiveCommandTheme(cmd *cobra.Command, theme *string) {
	if cmd == nil || theme == nil {
		return
	}
	themeFlag := cmd.Flags().Lookup("theme")
	if themeFlag != nil && themeFlag.Changed {
		return
	}
	*theme = effectiveDefaultTheme(cfg)
}

// handleAIResult 处理 AI 模式结果
func handleAIResult(result *converter.ConvertResult, markdownFile string) error {
	prompt, images, ok := converter.GetAIRequestInfo(result)
	if !ok {
		return newCLIError(codeConvertFailed, "invalid AI request result")
	}

	log.Info("AI mode request prepared",
		zap.Int("image_count", len(images)),
		zap.Int("prompt_length", len(prompt)))

	promptOutputPath := resolveAIPromptOutputPath(convertOutput)

	// 输出 AI 请求信息
	response := map[string]any{
		"markdown_file": markdownFile,
		"mode":          "ai",
		"action":        "ai_request",
		"prompt":        prompt,
		"images":        images,
		"prompt_file":   promptOutputPath,
	}
	if convertOutput != "" {
		response["requested_output_file"] = convertOutput
	}

	if promptOutputPath != "" {
		if _, err := atomicfile.Write(promptOutputPath, []byte(prompt)); err != nil {
			wrapped := fmt.Errorf("write AI prompt file: %w", err)
			return wrapCLIError(codeConvertFailed, wrapped, wrapped.Error())
		}
		log.Info("ai prompt saved", zap.String("file", promptOutputPath))
	}

	responseActionRequiredWith(codeConvertAIRequestReady, "Convert AI request prepared", response)

	return nil
}

func resolveAIPromptOutputPath(outputPath string) string {
	if outputPath == "" {
		return ""
	}

	ext := strings.ToLower(filepath.Ext(outputPath))
	switch ext {
	case ".html", ".htm":
		return strings.TrimSuffix(outputPath, ext) + ".prompt.txt"
	default:
		return outputPath
	}
}

func validateConvertConfig() error {
	switch convertMode {
	case "", "api", "ai":
	default:
		return newCLIError(codeConvertInvalid, fmt.Sprintf("invalid convert mode: %s", convertMode))
	}

	if convertPreview && (convertUpload || convertDraft) {
		return newCLIError(codeConfigInvalid, "--preview cannot be combined with --upload or --draft")
	}
	if !convertDraft && (strings.TrimSpace(convertCoverImage) != "" || strings.TrimSpace(convertCoverMediaID) != "") {
		return newCLIError(codeConfigInvalid, "--cover and --cover-media-id require --draft")
	}

	if convertMode == "api" {
		if convertAPIKey == "" && cfg.MD2WechatAPIKey == "" {
			return newCLIError(codeConvertInvalid, "MD2WECHAT_API_KEY is required for API mode")
		}
	}

	if err := resolveExplicitWeChatAccountIfProvided(); err != nil {
		return err
	}

	if strings.TrimSpace(convertCoverImage) != "" && strings.TrimSpace(convertCoverMediaID) != "" {
		return newCLIError(codeConvertInvalid, "--cover and --cover-media-id are mutually exclusive and cannot be used together")
	}
	if strings.TrimSpace(convertCoverMediaID) != "" && looksLikeURL(convertCoverMediaID) {
		return newCLIError(codeConvertInvalid, "--cover-media-id expects a WeChat media_id, not a URL")
	}

	if err := validateConvertThemeCompatibility(); err != nil {
		return err
	}

	return nil
}

func mapConvertServiceError(err error) error {
	if draftErr, ok := err.(*publish.DraftError); ok {
		return wrapCLIError(codeConvertDraftFailed, draftErr, draftErr.Error())
	}
	switch {
	case publish.IsAssetError(err):
		return wrapCLIError(codeConvertImageFailed, err, err.Error())
	case publish.IsDraftSaveError(err), publish.IsDraftCreateError(err):
		return wrapCLIError(codeConvertDraftFailed, err, err.Error())
	default:
		return wrapCLIError(codeConvertFailed, err, err.Error())
	}
}

func validateConvertThemeCompatibility() error {
	mode := converter.ConvertMode(convertMode)
	if mode == "" {
		mode = converter.ModeAPI
	}
	if mode == converter.ModeAI && strings.TrimSpace(convertCustomPrompt) != "" {
		return nil
	}

	themeManager := converter.NewThemeManager()
	if _, err := themeManager.ResolveThemeForMode(mode, convertTheme); err != nil {
		details := map[string]any{
			"mode":  string(mode),
			"theme": convertTheme,
		}
		nextActions := []string{
			"Run md2wechat themes list --json and choose a selectable theme for the selected mode.",
		}

		switch {
		case converter.IsThemeNotFound(err):
			return newCLIErrorWithDetails("THEME_NOT_FOUND", err.Error(), details, nextActions)
		case converter.IsThemeNotSelectable(err):
			return newCLIErrorWithDetails("THEME_NOT_SELECTABLE", err.Error(), details, nextActions)
		case converter.IsThemeModeMismatch(err):
			return newCLIErrorWithDetails("THEME_MODE_MISMATCH", err.Error(), details, nextActions)
		default:
			return wrapCLIError(codeConvertInvalid, err, err.Error())
		}
	}

	return nil
}

func looksLikeURL(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

func validateConvertMetadata(title, author, digest string) error {
	if err := validateConvertMetadataField("--title", title, 32); err != nil {
		return err
	}
	if err := validateConvertMetadataField("--author", author, 16); err != nil {
		return err
	}
	if err := validateConvertMetadataField("--digest", digest, 128); err != nil {
		return err
	}
	return nil
}

func validateConvertMetadataField(flagName, value string, limit int) error {
	if utf8.RuneCountInString(value) <= limit {
		return nil
	}
	return newCLIError(codeConvertInvalid, fmt.Sprintf("%s exceeds %d characters", flagName, limit))
}

func firstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

// uploadCoverImage 上传封面图片到微信素材库
func uploadCoverImage(imagePath string) (string, error) {
	svc := wechat.NewService(cfg, log)
	result, err := svc.UploadMaterial(imagePath)
	if err != nil {
		return "", err
	}
	return result.MediaID, nil
}

// outputHTML 输出 HTML
func outputHTML(html, outputPath string, preview bool) error {
	if preview || outputPath == "" {
		// 预览模式或未指定输出，输出纯 HTML 到标准输出
		if _, err := fmt.Fprint(os.Stdout, html); err != nil {
			log.Error("failed to write html to stdout", zap.Error(err))
			return fmt.Errorf("write HTML to stdout: %w", err)
		}
	}

	if outputPath != "" {
		if _, err := atomicfile.Write(outputPath, []byte(html)); err != nil {
			log.Error("failed to write output file", zap.Error(err))
			return fmt.Errorf("write output file: %w", err)
		}
		log.Info("html saved", zap.String("file", outputPath))
	}
	return nil
}
