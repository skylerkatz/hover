package command

import (
	"github.com/spf13/cobra"
	runCmd "hover/cmd/command/run"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "command <command>",
		Short: "Interact with command invocations",
	}

	cmd.AddCommand(runCmd.Cmd())

	return cmd
}
