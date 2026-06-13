package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/geekjourneyx/md2wechat-skill/internal/action"
	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"github.com/geekjourneyx/md2wechat-skill/internal/draft"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	cfg         *config.Config
	log         *zap.Logger
	jsonOutput  bool
	exitFunc              = os.Exit
	stdinReader io.Reader = os.Stdin
)

// Version is injected at build time.
var Version = "dev"

const (
	codeOK                     = "OK"
	codeError                  = "ERROR"
	codeConfigInvalid          = "CONFIG_INVALID"
	codeConfigNotFound         = "CONFIG_NOT_FOUND"
	codeConfigWriteFailed      = "CONFIG_WRITE_FAILED"
	codeConvertInvalid         = "CONVERT_INVALID"
	codeConvertReadFailed      = "CONVERT_READ_FAILED"
	codeConvertFailed          = "CONVERT_FAILED"
	codeConvertImageFailed     = "CONVERT_IMAGE_FAILED"
	codeConvertDraftFailed     = "CONVERT_DRAFT_FAILED"
	codeVersionShown           = "VERSION_SHOWN"
	codeConfigShown            = "CONFIG_SHOWN"
	codeConfigValidated        = "CONFIG_VALIDATED"
	codeConfigInitialized      = "CONFIG_INITIALIZED"
	codeWriteInputInvalid      = "WRITE_INPUT_INVALID"
	codeWriteReadFailed        = "WRITE_READ_FAILED"
	codeWriteFailed            = "WRITE_FAILED"
	codeWriteAIRequestReady    = "WRITE_AI_REQUEST_READY"
	codeHumanizeReadFailed     = "HUMANIZE_READ_FAILED"
	codeHumanizeWriteFailed    = "HUMANIZE_WRITE_FAILED"
	codeHumanizeRequestReady   = "HUMANIZE_REQUEST_READY"
	codeConvertAIRequestReady  = "CONVERT_AI_REQUEST_READY"
	codeConvertCompleted       = "CONVERT_COMPLETED"
	codeInspectCompleted       = "INSPECT_COMPLETED"
	codePreviewReady           = "PREVIEW_READY"
	codePreviewFailed          = "PREVIEW_FAILED"
	codeImageUploadFailed      = "IMAGE_UPLOAD_FAILED"
	codeImageGenerateFailed    = "IMAGE_GENERATE_FAILED"
	codeDraftCreateFailed      = "DRAFT_CREATE_FAILED"
	codeImagePostInvalid       = "IMAGE_POST_INVALID"
	codeImagePostPreviewFailed = "IMAGE_POST_PREVIEW_FAILED"
	codeImagePostCreateFailed  = "IMAGE_POST_CREATE_FAILED"
	codeImagePostPreviewReady  = "IMAGE_POST_PREVIEW_READY"
	codeImagePostCreated       = "IMAGE_POST_CREATED"
	codeDraftUpdateFailed      = "DRAFT_UPDATE_FAILED"
	codeTestDraftReadFailed    = "TEST_DRAFT_READ_FAILED"
	codeTestDraftCoverFailed   = "TEST_DRAFT_COVER_FAILED"
	codeTestDraftCreateFailed  = "TEST_DRAFT_CREATE_FAILED"
	codeTestDraftCreated       = "TEST_DRAFT_CREATED"

	codeLayoutModuleNotFound       = "LAYOUT_MODULE_NOT_FOUND"
	codeLayoutInvalidFilter        = "LAYOUT_INVALID_FILTER"
	codeLayoutMissingRequiredField = "LAYOUT_MISSING_REQUIRED_FIELD"
	codeLayoutInvalidFieldValue    = "LAYOUT_INVALID_FIELD_VALUE"
	codeLayoutValidateHasErrors    = "LAYOUT_VALIDATE_HAS_ERRORS"

	codeBrandInitialized = "BRAND_INITIALIZED"
	codeBrandInitFailed  = "BRAND_INIT_FAILED"
	codeBrandShown       = "BRAND_SHOWN"
	codeBrandNotFound    = "BRAND_NOT_FOUND"
	codeBrandReadFailed  = "BRAND_READ_FAILED"

	codeDoctorCompleted = "DOCTOR_COMPLETED"
)

type cliResponse struct {
	Success       bool           `json:"success"`
	Code          string         `json:"code,omitempty"`
	Message       string         `json:"message,omitempty"`
	SchemaVersion string         `json:"schema_version"`
	Status        action.Status  `json:"status"`
	Retryable     bool           `json:"retryable"`
	Data          any            `json:"data,omitempty"`
	Error         string         `json:"error,omitempty"`
	ErrorDetails  map[string]any `json:"error_details,omitempty"`
	NextActions   []string       `json:"next_actions,omitempty"`
}

type cliError struct {
	Code        string
	Message     string
	Retryable   bool
	Err         error
	Details     map[string]any
	NextActions []string
}

func (e *cliError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Code
}

func (e *cliError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func newCLIError(code, message string) error {
	return &cliError{Code: code, Message: message}
}

func newCLIErrorWithDetails(code, message string, details map[string]any, nextActions []string) error {
	return &cliError{Code: code, Message: message, Details: details, NextActions: nextActions}
}

func wrapCLIError(code string, err error, message string) error {
	return &cliError{Code: code, Message: message, Err: err}
}

func extractCLIError(err error) (*cliError, bool) {
	var cliErr *cliError
	if errors.As(err, &cliErr) {
		return cliErr, true
	}
	return nil, false
}

// initConfig 初始化配置（延迟加载，允许 help 命令无需配置）
func initConfig() error {
	if cfg != nil && log != nil {
		return nil
	}

	var err error
	config.SetQuiet(jsonOutput)
	cfg, err = config.Load()
	if err != nil {
		return err
	}

	if jsonOutput {
		log = zap.NewNop()
		return nil
	}

	log, err = zap.NewProduction()
	if err != nil {
		return err
	}

	return nil
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "md2wechat",
		Short: "Markdown to WeChat Official Account converter",
		Long: `md2wechat converts Markdown articles to WeChat Official Account format
and supports uploading materials and creating drafts.

Environment Variables:
  WECHAT_APPID                   WeChat Official Account AppID (required)
  WECHAT_SECRET                  WeChat API Secret (required)
  IMAGE_API_KEY                  Image generation API key (for AI images)
  IMAGE_API_BASE                 Image API base URL (default: https://api.openai.com/v1)
  COMPRESS_IMAGES                Compress images > 1920px (default: true)
  MAX_IMAGE_WIDTH                Max image width in pixels (default: 1920)

Examples:
  md2wechat upload_image ./photo.jpg
  md2wechat download_and_upload https://example.com/image.jpg
  md2wechat generate_image "A cute cat"
  md2wechat create_draft draft.json
  md2wechat update_draft <media_id> draft.json --index 0`,
		SilenceErrors: true,
		SilenceUsage:  true,
		Version:       Version,
	}
	rootCmd.SetVersionTemplate("{{printf \"%s\\n\" .Version}}")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Emit machine-readable JSON output")

	// upload_image command
	var uploadImageCmd = &cobra.Command{
		Use:   "upload_image <file_path>",
		Short: "Upload local image to WeChat material library",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			if err := cfg.ValidateForWeChat(); err != nil {
				return wrapCLIError(codeConfigInvalid, err, err.Error())
			}
			processor := newRuntimeImageProcessor()
			result, err := processor.UploadLocalImage(filePath)
			if err != nil {
				return wrapCLIError(codeImageUploadFailed, err, err.Error())
			}
			responseSuccess(result)
			return nil
		},
	}
	rootCmd.AddCommand(uploadImageCmd)

	// download_and_upload command
	var downloadAndUploadCmd = &cobra.Command{
		Use:   "download_and_upload <url>",
		Short: "Download online image and upload to WeChat",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			if err := cfg.ValidateForWeChat(); err != nil {
				return wrapCLIError(codeConfigInvalid, err, err.Error())
			}
			processor := newRuntimeImageProcessor()
			result, err := processor.DownloadAndUpload(url)
			if err != nil {
				return wrapCLIError(codeImageUploadFailed, err, err.Error())
			}
			responseSuccess(result)
			return nil
		},
	}
	rootCmd.AddCommand(downloadAndUploadCmd)

	var generateImageCmd = &cobra.Command{
		Use:   "generate_image [prompt]",
		Short: "Generate image via AI and upload to WeChat",
		Args:  cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerateImage(args)
		},
	}
	generateImageCmd.Flags().StringVarP(&generateImageCmdSize, "size", "s", "", "Image size (e.g., 2560x1440 for 16:9)")
	generateImageCmd.Flags().StringVar(&generateImageCmdPreset, "preset", "", "Prompt preset from the image prompt catalog")
	generateImageCmd.Flags().StringVarP(&generateImageCmdArticle, "article", "a", "", "Article markdown file used to render a preset prompt")
	generateImageCmd.Flags().StringVar(&generateImageCmdTitle, "title", "", "Article title used to render a preset prompt")
	generateImageCmd.Flags().StringVar(&generateImageCmdSummary, "summary", "", "Article summary used to render a preset prompt")
	generateImageCmd.Flags().StringVar(&generateImageCmdKeywords, "keywords", "", "Keywords used to render a preset prompt")
	generateImageCmd.Flags().StringVar(&generateImageCmdStyle, "style", "", "Visual style used to render a preset prompt")
	generateImageCmd.Flags().StringVar(&generateImageCmdAspect, "aspect", "", "Aspect ratio hint used to render a preset prompt, e.g. 16:9 or 3:4")
	generateImageCmd.Flags().StringVar(&generateImageCmdModel, "model", "", "Image model to use for this command (overrides IMAGE_MODEL and api.image_model)")
	rootCmd.AddCommand(generateImageCmd)
	rootCmd.AddCommand(generateCoverCmd)
	rootCmd.AddCommand(generateInfographicCmd)

	// create_draft command
	var createDraftCmd = &cobra.Command{
		Use:   "create_draft <json_file>",
		Short: "Create WeChat draft article from JSON file",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFile := args[0]
			if err := cfg.ValidateForWeChat(); err != nil {
				return wrapCLIError(codeConfigInvalid, err, err.Error())
			}
			svc := draft.NewService(cfg, log)
			result, err := svc.CreateDraftFromFile(jsonFile)
			if err != nil {
				return wrapCLIError(codeDraftCreateFailed, err, err.Error())
			}
			responseSuccess(result)
			return nil
		},
	}
	rootCmd.AddCommand(createDraftCmd)

	var updateDraftIndex int
	var updateDraftCmd = &cobra.Command{
		Use:   "update_draft <media_id> <json_file>",
		Short: "Update an existing WeChat draft article from JSON file",
		Args:  cobra.ExactArgs(2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			mediaID := args[0]
			jsonFile := args[1]
			if err := cfg.ValidateForWeChat(); err != nil {
				return wrapCLIError(codeConfigInvalid, err, err.Error())
			}
			svc := draft.NewService(cfg, log)
			result, err := svc.UpdateDraftFromFile(mediaID, updateDraftIndex, jsonFile)
			if err != nil {
				return wrapCLIError(codeDraftUpdateFailed, err, err.Error())
			}
			responseSuccess(result)
			return nil
		},
	}
	updateDraftCmd.Flags().IntVar(&updateDraftIndex, "index", 0, "Article index inside the existing draft to update")
	rootCmd.AddCommand(updateDraftCmd)

	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print CLI version",
		RunE: func(cmd *cobra.Command, args []string) error {
			runVersion()
			return nil
		},
	}
	rootCmd.AddCommand(versionCmd)

	// convert command
	rootCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(inspectCmd)
	rootCmd.AddCommand(previewCmd)

	// config command
	rootCmd.AddCommand(configCmd)

	// discovery commands
	rootCmd.AddCommand(capabilitiesCmd)
	rootCmd.AddCommand(providersCmd)
	rootCmd.AddCommand(themesCmd)
	rootCmd.AddCommand(promptsCmd)

	// write command
	rootCmd.AddCommand(writeCmd)

	// humanize command
	rootCmd.AddCommand(humanizeCmd)

	// test-draft command
	rootCmd.AddCommand(testHTMLCmd)

	// create-image-post command (小绿书)
	rootCmd.AddCommand(createImagePostCmd)

	// layout command
	rootCmd.AddCommand(layoutCmd)

	// brand command
	rootCmd.AddCommand(brandCmd)

	// doctor command
	rootCmd.AddCommand(doctorCmd)

	// embedded agent skills
	rootCmd.AddCommand(skillsCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		responseError(err)
	}
}

func responseSuccess(data any) {
	responseSuccessWith(codeOK, "Success", data)
}

func responseSuccessWith(code, message string, data any) {
	responseWith(cliResponse{
		Success:       true,
		Code:          code,
		Message:       message,
		SchemaVersion: action.SchemaVersion,
		Status:        action.StatusCompleted,
		Retryable:     false,
		Data:          data,
	})
}

func responseActionRequiredWith(code, message string, data any) {
	responseWith(cliResponse{
		Success:       true,
		Code:          code,
		Message:       message,
		SchemaVersion: action.SchemaVersion,
		Status:        action.StatusActionRequired,
		Retryable:     false,
		Data:          data,
	})
}

func responseError(err error) {
	if cliErr, ok := extractCLIError(err); ok {
		responseErrorWith(cliErr.Code, cliErr)
		return
	}
	responseErrorWith(codeError, err)
}

func responseErrorWith(code string, err error) {
	retryable := false
	var errorDetails map[string]any
	var nextActions []string
	if cliErr, ok := extractCLIError(err); ok {
		retryable = cliErr.Retryable
		errorDetails = cliErr.Details
		nextActions = cliErr.NextActions
	}
	responseWith(cliResponse{
		Success:       false,
		Code:          code,
		Message:       err.Error(),
		SchemaVersion: action.SchemaVersion,
		Status:        action.StatusFailed,
		Retryable:     retryable,
		Error:         err.Error(),
		ErrorDetails:  errorDetails,
		NextActions:   nextActions,
	})
	exitFunc(1)
}

func responseWith(resp cliResponse) {
	printJSON(resp)
}

func printJSON(v any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "JSON encode error: %v\n", err)
		exitFunc(1)
	}
}

func runVersion() {
	if jsonOutput {
		responseSuccessWith(codeVersionShown, "Version information", map[string]any{
			"version": Version,
		})
		return
	}
	_, _ = fmt.Fprintln(os.Stdout, Version)
}

// maskMediaID 遮蔽 media_id 用于日志
func maskMediaID(id string) string {
	if len(id) < 8 {
		return "***"
	}
	return id[:4] + "***" + id[len(id)-4:]
}
