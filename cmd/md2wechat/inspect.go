package main

import (
	"fmt"
	"os"
	"strings"

	inspectpkg "github.com/geekjourneyx/md2wechat-skill/internal/inspect"
	"github.com/spf13/cobra"
)

var (
	inspectMode           string
	inspectTheme          string
	inspectFontSize       string
	inspectBackgroundType string
	inspectCustomPrompt   string
	inspectTitle          string
	inspectAuthor         string
	inspectDigest         string
	inspectCover          string
	inspectCoverMediaID   string
	inspectUpload         bool
	inspectDraft          bool
	inspectStrict         bool
)

var inspectCmd = &cobra.Command{
	Use:   "inspect <markdown_file>",
	Short: "Inspect resolved article metadata, readiness, and publish risks",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := initConfig(); err != nil {
			return err
		}
		applyEffectiveCommandTheme(cmd, &inspectTheme)
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := runInspect(args[0])
		if err != nil {
			return err
		}
		if jsonOutput {
			responseSuccessWith(codeInspectCompleted, "Inspect completed", result)
		} else {
			printInspect(result)
		}
		if inspectStrict && hasErrorCheck(result.Checks) {
			exitFunc(2)
		}
		return nil
	},
}

func init() {
	inspectCmd.Flags().StringVar(&inspectMode, "mode", "api", "Inspection context mode: api or ai")
	inspectCmd.Flags().StringVar(&inspectTheme, "theme", "default", "Theme name")
	inspectCmd.Flags().StringVar(&inspectFontSize, "font-size", "medium", "Font size: small/medium/large")
	inspectCmd.Flags().StringVar(&inspectBackgroundType, "background-type", "none", "Background type: default/grid/none")
	inspectCmd.Flags().StringVar(&inspectCustomPrompt, "custom-prompt", "", "Custom AI prompt context; mirrors convert AI custom-prompt theme handling")
	inspectCmd.Flags().StringVar(&inspectTitle, "title", "", "Override article title")
	inspectCmd.Flags().StringVar(&inspectAuthor, "author", "", "Override article author")
	inspectCmd.Flags().StringVar(&inspectDigest, "digest", "", "Override article digest")
	inspectCmd.Flags().StringVar(&inspectCover, "cover", "", "Cover image path to evaluate draft target state")
	inspectCmd.Flags().StringVar(&inspectCoverMediaID, "cover-media-id", "", "Existing WeChat cover media_id to evaluate draft target state")
	inspectCmd.Flags().BoolVar(&inspectUpload, "upload", false, "Evaluate upload target state")
	inspectCmd.Flags().BoolVar(&inspectDraft, "draft", false, "Evaluate draft target state")
	inspectCmd.Flags().BoolVar(&inspectStrict, "strict", false, "Exit with status 2 if error-level checks are found")
}

func runInspect(markdownFile string) (*inspectpkg.Result, error) {
	markdown, err := os.ReadFile(markdownFile)
	if err != nil {
		return nil, wrapCLIError(codeConvertReadFailed, err, fmt.Sprintf("read markdown file: %v", err))
	}
	if strings.TrimSpace(wechatAccountName) != "" || inspectUpload || inspectDraft {
		if err := cfg.ResolveWeChatAccount(wechatAccountName); err != nil {
			return nil, mapConfigAccountError(err)
		}
	}
	return runInspectWithInput(markdownFile, string(markdown), inspectpkg.Input{
		MarkdownFile:    markdownFile,
		Markdown:        string(markdown),
		Mode:            inspectMode,
		Theme:           inspectTheme,
		FontSize:        inspectFontSize,
		BackgroundType:  inspectBackgroundType,
		CustomPrompt:    inspectCustomPrompt,
		TitleOverride:   inspectTitle,
		AuthorOverride:  inspectAuthor,
		DigestOverride:  inspectDigest,
		CoverImagePath:  inspectCover,
		CoverMediaID:    inspectCoverMediaID,
		UploadRequested: inspectUpload,
		DraftRequested:  inspectDraft,
		Config:          cfg,
	})
}

func runInspectWithInput(markdownFile, markdown string, input inspectpkg.Input) (*inspectpkg.Result, error) {
	if err := validateConfirmMode(input.Mode); err != nil {
		return nil, err
	}
	input.MarkdownFile = markdownFile
	input.Markdown = markdown
	input.Config = cfg
	return inspectpkg.Run(&input)
}

func validateConfirmMode(mode string) error {
	mode = strings.TrimSpace(mode)
	switch mode {
	case "", "api", "ai":
		return nil
	default:
		return newCLIError(codeConvertInvalid, fmt.Sprintf("invalid convert mode: %s", mode))
	}
}

func printInspect(result *inspectpkg.Result) {
	if result == nil {
		return
	}

	fmt.Printf("Article: %s\n\n", result.SourceFile)
	fmt.Println("Metadata")
	fmt.Printf("- title: %s\n  source: %s\n  limit: %d\n", result.Metadata.Title.Value, result.Metadata.Title.Source, result.Metadata.Title.Limit)
	fmt.Printf("- author: %s\n  source: %s\n  limit: %d\n", emptyDash(result.Metadata.Author.Value), result.Metadata.Author.Source, result.Metadata.Author.Limit)
	fmt.Printf("- digest: %s\n  source: %s\n  limit: %d\n\n", emptyDash(result.Metadata.Digest.Value), result.Metadata.Digest.Source, result.Metadata.Digest.Limit)

	fmt.Println("Structure")
	fmt.Printf("- body_h1: %s\n", emptyDash(result.Structure.BodyH1.Text))
	fmt.Printf("- duplicate_title_risk: %t\n", result.Structure.DuplicateTitleRisk)
	fmt.Printf("- images: %d\n", result.Structure.Images.Total)
	fmt.Printf("- preview_fidelity: %s\n", result.Readiness.PreviewFidelity)
	fmt.Printf("- convert_ready: %t\n- upload_ready: %t\n- draft_ready: %t\n\n", result.Readiness.ConvertReady, result.Readiness.UploadReady, result.Readiness.DraftReady)

	fmt.Println("Checks")
	if len(result.Checks) == 0 {
		fmt.Println("- none")
		return
	}
	for _, check := range result.Checks {
		fmt.Printf("- %s %s: %s\n", strings.ToUpper(check.Level), check.Code, check.Message)
		if check.SuggestedFix != "" {
			fmt.Printf("  fix: %s\n", check.SuggestedFix)
		}
	}
}

func hasErrorCheck(checks []inspectpkg.Check) bool {
	for _, check := range checks {
		if check.Level == inspectpkg.LevelError {
			return true
		}
	}
	return false
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return value
}
