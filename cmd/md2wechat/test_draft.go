package main

import (
	"github.com/spf13/cobra"
)

var testHTMLCmd = &cobra.Command{
	Use:   "test-draft <html_file> <cover_image>",
	Short: "Test creating WeChat draft from HTML file",
	Args:  cobra.ExactArgs(2),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		response, err := runTestDraft(args[0], args[1])
		if err != nil {
			return err
		}
		responseSuccessWith(codeTestDraftCreated, "Draft created successfully", response)
		return nil
	},
}

func runTestDraft(htmlFile, coverImage string) (map[string]any, error) {
	result, err := executeDraftCreation(draftExecutionInput{
		HTMLFile:   htmlFile,
		CoverImage: coverImage,
		Title:      "AI生成测试文章",
		Digest:     "这是AI生成的微信公众号文章测试",
	})
	if err != nil {
		if _, ok := extractCLIError(err); ok {
			return nil, err
		}
		return nil, wrapDraftExecutionError(err, codeTestDraftReadFailed, codeTestDraftCoverFailed, codeTestDraftCreateFailed)
	}
	return buildDraftResponse(result), nil
}
