package main

import "github.com/spf13/cobra"

var syncCmd = newSyncCommand()

func newSyncCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "sync", Short: "Prepare local content for host-agent draft creation", SilenceErrors: true, SilenceUsage: true}
	cmd.Args = cobra.NoArgs
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return cmd.Help() }
	cmd.AddCommand(newSyncPrepareCommand())
	return cmd
}
