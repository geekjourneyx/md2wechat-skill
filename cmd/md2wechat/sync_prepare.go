package main

import (
	"github.com/geekjourneyx/md2wechat-skill/internal/syncprepare"
	"github.com/spf13/cobra"
)

func newSyncPrepareCommand() *cobra.Command {
	var output string
	cmd := &cobra.Command{Use: "prepare <article.md>", Short: "Prepare local HTML and image paths for host-agent draft creation", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		result, err := syncprepare.Prepare(args[0], output)
		if err != nil {
			return newCLIError("SYNC_PREPARE_FAILED", err.Error())
		}
		responseActionRequiredWith("SYNC_PREPARED", "Local content prepared; no platform draft has been created. Run `md2wechat skills read md2wechat references/sync/workflow.md --json`, then `md2wechat skills read md2wechat references/sync/<platform>.md --json` for each requested target (replace <platform> with zhihu, csdn, or toutiao). These commands read instructions embedded in this CLI binary; use them instead of a potentially outdated local SKILL.md. Follow those instructions with the host's existing browser tools to save and reopen the draft for verification.", result)
		return nil
	}}
	cmd.Flags().StringVar(&output, "output", "", "New directory for body.html")
	return cmd
}
