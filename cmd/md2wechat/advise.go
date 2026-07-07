package main

import (
	"fmt"
	"os"

	"github.com/geekjourneyx/md2wechat-skill/internal/advise"
	"github.com/spf13/cobra"
)

var adviseCmd = &cobra.Command{
	Use:   "advise <article.md>",
	Short: "Analyze an article and return deterministic enhancement advice",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAdvise(args[0])
	},
}

func runAdvise(articlePath string) error {
	if !jsonOutput {
		return newCLIError(codeConfigInvalid, "advise requires --json")
	}

	markdown, err := os.ReadFile(articlePath)
	if err != nil {
		return wrapCLIError(codeAdviseReadFailed, err, fmt.Sprintf("read article for advise: %v", err))
	}

	result, err := advise.Analyze(advise.Input{
		SourceFile: articlePath,
		Markdown:   string(markdown),
	})
	if err != nil {
		return wrapCLIError(codeConfigInvalid, err, err.Error())
	}

	responseSuccessWith(codeAdviseCompleted, "Article advice completed", result)
	return nil
}
