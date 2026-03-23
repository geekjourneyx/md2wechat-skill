package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
  - ai:  Use Claude AI to generate HTML (flexible, requires AI)

Supported themes (38 total):
  Basic (6): default, bytedance, apple, sports, chinese, cyber
  Minimal (8): minimal-gold, minimal-green, minimal-blue, minimal-orange, minimal-red, minimal-navy, minimal-gray, minimal-sky
  Focus (8): focus-gold, focus-green, focus-blue, focus-orange, focus-red, focus-navy, focus-gray, focus-sky
  Elegant (8): elegant-gold, elegant-green, elegant-blue, elegant-orange, elegant-red, elegant-navy, elegant-gray, elegant-sky
  Bold (8): bold-gold, bold-green, bold-blue, bold-orange, bold-red, bold-navy, bold-gray, bold-sky

  AI modes: autumn-warm, spring-fresh, ocean-calm, custom`,
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
	convertMode         string
	convertTheme        string
	convertCustomPrompt string
	convertOutput       string
	convertPreview      bool
	convertUpload       bool
	convertDraft        bool
	convertSaveDraft    string
	convertCoverImage   string // 封面图片路径
)

func init() {
	// 添加 flags
	convertCmd.Flags().StringVar(&convertMode, "mode", "ai", "Conversion mode: ai (only)")
	convertCmd.Flags().StringVar(&convertTheme, "theme", "default", "Theme name")
	convertCmd.Flags().StringVar(&convertCustomPrompt, "custom-prompt", "", "Custom AI prompt")
	convertCmd.Flags().StringVarP(&convertOutput, "output", "o", "", "Output HTML file path")
	convertCmd.Flags().BoolVar(&convertPreview, "preview", false, "Preview only, do not upload images")
	convertCmd.Flags().BoolVar(&convertUpload, "upload", false, "Upload images to WeChat and replace URLs")
	convertCmd.Flags().BoolVar(&convertDraft, "draft", false, "Create WeChat draft after conversion")
	convertCmd.Flags().StringVar(&convertSaveDraft, "save-draft", "", "Save draft JSON to file")
	convertCmd.Flags().StringVar(&convertCoverImage, "cover", "", "Cover image path for draft (required when using --draft)")
}

// runConvert 执行转换
func runConvert(cmd *cobra.Command, args []string) error {
	markdownFile := args[0]

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

	metadata := converter.ParseArticleMetadata(string(markdown))
	service := newPublishService()
	input := &publish.ConvertInput{
		Source: publish.ArticleSource{
			Path:     markdownFile,
			Markdown: string(markdown),
			Metadata: publish.Metadata{
				Title:  metadata.Title,
				Author: metadata.Author,
				Digest: metadata.Digest,
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
			Markdown:     string(markdown),
			Mode:         converter.ConvertMode(convertMode),
			Theme:        convertTheme,
			CustomPrompt: convertCustomPrompt,
		},
		MarkdownDir:    filepath.Dir(markdownFile),
		OutputFile:     convertOutput,
		SaveDraftPath:  convertSaveDraft,
		CoverImagePath: convertCoverImage,
	}

	output, err := service.Convert(input)
	if err != nil {
		switch e := err.(type) {
		case *publish.DraftError:
			return wrapCLIError(codeConvertDraftFailed, e, e.Error())
		default:
			switch {
			case publish.IsAssetError(err):
				return wrapCLIError(codeConvertImageFailed, err, err.Error())
			case publish.IsDraftSaveError(err), publish.IsDraftCreateError(err):
				return wrapCLIError(codeConvertDraftFailed, err, err.Error())
			default:
				return wrapCLIError(codeConvertFailed, err, err.Error())
			}
		}
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

	if convertOutput != "" {
		outputHTML(output.Artifact.HTML, convertOutput, false)
	}

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
	outputHTML(output.Artifact.HTML, "", convertPreview)

	return nil
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

	responseActionRequiredWith(codeConvertAIRequestReady, "Convert AI request prepared", response)

	if promptOutputPath != "" {
		if err := os.WriteFile(promptOutputPath, []byte(prompt), 0644); err != nil {
			log.Warn("failed to save prompt", zap.Error(err))
		} else {
			log.Info("ai prompt saved", zap.String("file", promptOutputPath))
		}
	}

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
	case "", "ai":
	default:
		return newCLIError(codeConvertInvalid, fmt.Sprintf("invalid convert mode: %s", convertMode))
	}

	if convertUpload || convertDraft {
		if err := cfg.ValidateForWeChat(); err != nil {
			return wrapCLIError(codeConfigInvalid, err, err.Error())
		}
	}

	return nil
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
func outputHTML(html, outputPath string, preview bool) {
	if preview || outputPath == "" {
		// 预览模式或未指定输出，输出到标准输出
		fmt.Println("\n=== HTML Output ===")
		fmt.Println(html)
		fmt.Println("\n=== End ===")
	}

	if outputPath != "" {
		if err := os.WriteFile(outputPath, []byte(html), 0644); err != nil {
			log.Error("failed to write output file", zap.Error(err))
		} else {
			log.Info("html saved", zap.String("file", outputPath))
		}
	}
}
