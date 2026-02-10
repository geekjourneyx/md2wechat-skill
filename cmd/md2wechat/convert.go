package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/geekjourneyx/md2wechat-skill/internal/converter"
	"github.com/geekjourneyx/md2wechat-skill/internal/draft"
	"github.com/geekjourneyx/md2wechat-skill/internal/image"
	"github.com/geekjourneyx/md2wechat-skill/internal/wechat"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
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
	Run: func(cmd *cobra.Command, args []string) {
		if err := runConvert(cmd, args); err != nil {
			responseError(err)
		}
	},
}

// convert 命令参数
var (
	convertMode         string
	convertTheme        string
	convertAPIKey       string
	convertFontSize     string
	convertBackgroundType string
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
	convertCmd.Flags().StringVar(&convertMode, "mode", "api", "Conversion mode: api or ai")
	convertCmd.Flags().StringVar(&convertTheme, "theme", "default", "Theme name")
	convertCmd.Flags().StringVar(&convertAPIKey, "api-key", "", "API key for md2wechat.cn")
	convertCmd.Flags().StringVar(&convertFontSize, "font-size", "medium", "Font size: small/medium/large (API mode only)")
	convertCmd.Flags().StringVar(&convertBackgroundType, "background-type", "default", "Background type: default/grid/none (API mode only)")
	convertCmd.Flags().StringVar(&convertCustomPrompt, "custom-prompt", "", "Custom AI prompt (AI mode only)")
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

	log.Info("starting conversion",
		zap.String("file", markdownFile),
		zap.String("mode", convertMode),
		zap.String("theme", convertTheme))

	// 读取 Markdown 文件
	markdown, err := os.ReadFile(markdownFile)
	if err != nil {
		return fmt.Errorf("read markdown file: %w", err)
	}

	// 创建转换器
	conv := converter.NewConverter(cfg, log)

	// 构建转换请求
	req := &converter.ConvertRequest{
		Markdown:       string(markdown),
		Mode:           converter.ConvertMode(convertMode),
		Theme:          convertTheme,
		APIKey:         convertAPIKey,
		FontSize:       convertFontSize,
		BackgroundType: convertBackgroundType,
		CustomPrompt:   convertCustomPrompt,
	}

	// 执行转换
	result := conv.Convert(req)

	if !result.Success {
		return fmt.Errorf("conversion failed: %s", result.Error)
	}

	log.Info("conversion completed",
		zap.String("mode", string(result.Mode)),
		zap.String("theme", result.Theme),
		zap.Int("image_count", len(result.Images)))

	// 根据模式处理结果
	if convertMode == "ai" && converter.IsAIRequest(result) {
		// AI 模式需要外部处理
		return handleAIResult(result, markdownFile)
	}

	// 处理图片
	if convertUpload || convertDraft {
		if err := processImages(result); err != nil {
			log.Warn("image processing failed", zap.Error(err))
		}
	}

	// 输出结果
	if convertSaveDraft != "" {
		if err := saveDraft(result); err != nil {
			return fmt.Errorf("save draft: %w", err)
		}
	}

	if convertDraft {
		if err := createWeChatDraft(result, convertCoverImage); err != nil {
			return fmt.Errorf("create draft: %w", err)
		}
	}

	// 输出 HTML
	outputHTML(result.HTML, convertOutput, convertPreview)

	return nil
}

// handleAIResult 处理 AI 模式结果
func handleAIResult(result *converter.ConvertResult, markdownFile string) error {
	prompt, images, ok := converter.GetAIRequestInfo(result)
	if !ok {
		return fmt.Errorf("invalid AI request result")
	}

	log.Info("AI mode request prepared",
		zap.Int("image_count", len(images)),
		zap.Int("prompt_length", len(prompt)))

	// 输出 AI 请求信息
	response := map[string]any{
		"success":       true,
		"mode":          "ai",
		"action":        "ai_request",
		"markdown_file": markdownFile,
		"prompt":        prompt,
		"images":        images,
	}

	printJSON(response)

	if convertOutput != "" {
		// 同时保存原始 markdown 到输出文件，方便用户使用
		if err := os.WriteFile(convertOutput, []byte(prompt), 0644); err != nil {
			log.Warn("failed to save prompt", zap.Error(err))
		}
	}

	return nil
}

// processImages 处理图片上传
func processImages(result *converter.ConvertResult) error {
	if len(result.Images) == 0 {
		log.Info("no images to process")
		return nil
	}

	processor := image.NewProcessor(cfg, log)

	for i, imgRef := range result.Images {
		log.Info("processing image",
			zap.Int("index", i),
			zap.String("type", string(imgRef.Type)),
			zap.String("original", imgRef.Original))

		var uploadResult *image.UploadResult
		var err error

		switch imgRef.Type {
		case converter.ImageTypeLocal:
			uploadResult, err = processor.UploadLocalImage(imgRef.Original)
		case converter.ImageTypeOnline:
			uploadResult, err = processor.DownloadAndUpload(imgRef.Original)
		case converter.ImageTypeAI:
			// AI 生成的图片需要先调用生成 API
			genResult, genErr := processor.GenerateAndUpload(imgRef.AIPrompt)
			if genErr != nil {
				err = genErr
			} else {
				uploadResult = &image.UploadResult{
					MediaID:   genResult.MediaID,
					WechatURL: genResult.WechatURL,
				}
			}
		}

		if err != nil {
			log.Warn("image upload failed",
				zap.Int("index", i),
				zap.Error(err))
			continue
		}

		// 更新图片 URL
		result.Images[i].WechatURL = uploadResult.WechatURL

		log.Info("image uploaded",
			zap.Int("index", i),
			zap.String("media_id", maskMediaID(uploadResult.MediaID)),
			zap.String("wechat_url", uploadResult.WechatURL))
	}

	// 替换 HTML 中的图片占位符
	result.HTML = converter.ReplaceImagePlaceholders(result.HTML, result.Images)

	return nil
}

// saveDraft 保存草稿 JSON 到文件
func saveDraft(result *converter.ConvertResult) error {
	articles := []draft.Article{
		{
			Title:   "Draft Article", // TODO: 从 markdown 提取标题
			Content: result.HTML,
		},
	}

	draftData := map[string]any{
		"articles": articles,
	}

	jsonData, err := json.MarshalIndent(draftData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal draft: %w", err)
	}

	if err := os.WriteFile(convertSaveDraft, jsonData, 0644); err != nil {
		return fmt.Errorf("write draft file: %w", err)
	}

	log.Info("draft saved", zap.String("file", convertSaveDraft))
	return nil
}

// createWeChatDraft 创建微信草稿
func createWeChatDraft(result *converter.ConvertResult, coverImagePath string) error {
	svc := draft.NewService(cfg, log)

	// 检查封面图片（微信要求必须有封面图）
	if coverImagePath == "" {
		return &DraftError{
			Message: "创建草稿需要封面图片",
			Hint:    "请使用 --cover 参数指定封面图片路径，例如: --cover /path/to/cover.jpg\n" +
				"或者先上传封面图片到微信素材库: md2wechat upload_image /path/to/cover.jpg",
		}
	}

	// 上传封面图片到微信素材库
	log.Info("uploading cover image", zap.String("path", coverImagePath))
	coverMediaID, err := uploadCoverImage(coverImagePath)
	if err != nil {
		return fmt.Errorf("上传封面图片失败: %w", err)
	}
	log.Info("cover image uploaded", zap.String("media_id", maskMediaID(coverMediaID)))

	// 提取标题（TODO: 从 markdown frontmatter 或第一个标题获取）
	title := "Article Title"

	draftResult, err := svc.CreateDraft([]draft.Article{
		{
			Title:          title,
			Content:        result.HTML,
			Digest:         draft.GenerateDigestFromContent(result.HTML, 120),
			ThumbMediaID:   coverMediaID,
			ShowCoverPic:   1, // 显示封面
		},
	})

	if err != nil {
		return fmt.Errorf("create draft: %w", err)
	}

	log.Info("draft created",
		zap.String("media_id", maskMediaID(draftResult.MediaID)),
		zap.String("draft_url", draftResult.DraftURL))

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

// DraftError 草稿错误
type DraftError struct {
	Message string
	Hint    string
}

func (e *DraftError) Error() string {
	msg := fmt.Sprintf("草稿错误: %s", e.Message)
	if e.Hint != "" {
		msg += fmt.Sprintf("\n💡 提示:\n   %s", e.Hint)
	}
	return msg
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
