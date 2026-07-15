package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/geekjourneyx/md2wechat-skill/internal/action"
	"github.com/geekjourneyx/md2wechat-skill/internal/converter"
	inspectpkg "github.com/geekjourneyx/md2wechat-skill/internal/inspect"
	"github.com/spf13/cobra"
)

var (
	previewMode           string
	previewTheme          string
	previewFontSize       string
	previewBackgroundType string
	previewTitle          string
	previewAuthor         string
	previewDigest         string
	previewCover          string
	previewUpload         bool
	previewDraft          bool
	previewOutput         string
)

type previewRunResult struct {
	Inspect    *inspectpkg.Result
	Conversion *converter.ConvertResult
	OutputFile string
}

var previewWriteTemp = func(file *os.File, data []byte) (int, error) {
	return file.Write(data)
}

var replaceOutputFileFn = replaceOutputFile

var previewCmd = &cobra.Command{
	Use:   "preview <markdown_file>",
	Short: "Generate a standalone article preview HTML file",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := runPreview(args[0])
		if err != nil {
			return err
		}
		if converter.IsAIRequest(result.Conversion) || result.Conversion.Status == action.StatusActionRequired {
			responseActionRequiredWith(codePreviewActionRequired, "Preview requires final HTML from the host Agent", map[string]any{
				"output_file": "",
				"inspect":     result.Inspect,
				"prompt":      result.Conversion.Prompt,
			})
			return nil
		}
		render := map[string]any{
			"mode":       result.Inspect.Context.Mode,
			"theme":      result.Inspect.Context.Theme,
			"fidelity":   inspectpkg.PreviewFidelityExact,
			"exact_html": true,
		}
		if jsonOutput {
			responseSuccessWith(codePreviewReady, "Preview ready", map[string]any{
				"output_file": result.OutputFile,
				"inspect":     result.Inspect,
				"render":      render,
			})
			return nil
		}
		fmt.Printf("Preview written to %s\n", result.OutputFile)
		return nil
	},
}

func init() {
	previewCmd.Flags().StringVar(&previewMode, "mode", "api", "Preview context mode: api or ai")
	previewCmd.Flags().StringVar(&previewTheme, "theme", "default", "Theme name")
	previewCmd.Flags().StringVar(&previewFontSize, "font-size", "medium", "Font size: small/medium/large")
	previewCmd.Flags().StringVar(&previewBackgroundType, "background-type", "none", "Background type: default/grid/none")
	previewCmd.Flags().StringVar(&previewTitle, "title", "", "Override article title")
	previewCmd.Flags().StringVar(&previewAuthor, "author", "", "Override article author")
	previewCmd.Flags().StringVar(&previewDigest, "digest", "", "Override article digest")
	previewCmd.Flags().StringVar(&previewCover, "cover", "", "Cover image path to evaluate draft target state")
	previewCmd.Flags().BoolVar(&previewUpload, "upload", false, "Evaluate upload target state in preview")
	previewCmd.Flags().BoolVar(&previewDraft, "draft", false, "Evaluate draft target state in preview")
	previewCmd.Flags().StringVarP(&previewOutput, "output", "o", "", "Write preview HTML to this file (default: temp file)")
}

func runPreview(markdownFile string) (*previewRunResult, error) {
	requestedOutput := previewOutput
	previewOutput = ""

	markdown, err := os.ReadFile(markdownFile)
	if err != nil {
		return nil, wrapCLIError(codeConvertReadFailed, err, fmt.Sprintf("read markdown file: %v", err))
	}

	result, err := runInspectWithInput(markdownFile, string(markdown), inspectpkg.Input{
		Mode:            previewMode,
		Theme:           previewTheme,
		FontSize:        previewFontSize,
		BackgroundType:  previewBackgroundType,
		TitleOverride:   previewTitle,
		AuthorOverride:  previewAuthor,
		DigestOverride:  previewDigest,
		CoverImagePath:  previewCover,
		UploadRequested: previewUpload,
		DraftRequested:  previewDraft,
	})
	if err != nil {
		return nil, err
	}

	req := &converter.ConvertRequest{
		Markdown:       result.Body,
		Mode:           converter.ConvertMode(result.Context.Mode),
		Theme:          result.Context.Theme,
		FontSize:       result.Context.FontSize,
		BackgroundType: result.Context.BackgroundType,
		Metadata: converter.ArticleMetadata{
			Title:  result.Metadata.Title.Value,
			Author: result.Metadata.Author.Value,
			Digest: result.Metadata.Digest.Value,
		},
	}
	conv := newMarkdownConverter()
	if conv == nil {
		err := fmt.Errorf("markdown converter is not available")
		return nil, wrapCLIError(codePreviewFailed, err, err.Error())
	}
	converted := conv.Convert(req)
	runResult := &previewRunResult{Inspect: result, Conversion: converted}
	if converted == nil {
		err := fmt.Errorf("converter returned nil result")
		return nil, wrapCLIError(codePreviewFailed, err, err.Error())
	}
	if converter.IsAIRequest(converted) || converted.Status == action.StatusActionRequired {
		return runResult, nil
	}
	if !converted.Success {
		message := converted.Error
		if message == "" {
			message = "preview conversion failed"
		}
		err := fmt.Errorf("%s", message)
		return nil, wrapCLIError(codePreviewFailed, err, err.Error())
	}
	if strings.TrimSpace(converted.HTML) == "" {
		err := fmt.Errorf("preview conversion returned empty HTML")
		return nil, wrapCLIError(codePreviewFailed, err, err.Error())
	}

	outputFile, err := writeOutputFileAtomically(requestedOutput, []byte(converted.HTML))
	if err != nil {
		return nil, wrapCLIError(codePreviewFailed, err, err.Error())
	}
	runResult.OutputFile = outputFile
	return runResult, nil
}

func writeOutputFileAtomically(outputFile string, data []byte) (string, error) {
	explicitOutput := outputFile != ""
	tempDir := ""
	tempPattern := "md2wechat-preview-*.html"
	if explicitOutput {
		tempDir = filepath.Dir(outputFile)
		tempPattern = "." + filepath.Base(outputFile) + ".tmp-*"
	}

	temp, err := os.CreateTemp(tempDir, tempPattern)
	if err != nil {
		return "", err
	}
	tempPath := temp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = temp.Close()
			_ = os.Remove(tempPath)
		}
	}()

	written, err := previewWriteTemp(temp, data)
	if err != nil {
		return "", err
	}
	if written != len(data) {
		return "", io.ErrShortWrite
	}
	if explicitOutput {
		if err := temp.Chmod(0644); err != nil {
			return "", err
		}
	}
	if err := temp.Sync(); err != nil {
		return "", err
	}
	if err := temp.Close(); err != nil {
		return "", err
	}

	if explicitOutput {
		if err := replaceOutputFileFn(tempPath, outputFile); err != nil {
			return "", err
		}
		removeTemp = false
		return outputFile, nil
	}

	removeTemp = false
	return tempPath, nil
}
