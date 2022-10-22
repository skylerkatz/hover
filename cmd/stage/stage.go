package stage

import (
	"github.com/spf13/cobra"
	deleteCmd "hover/cmd/stage/delete"
	newCmd "hover/cmd/stage/new"
	purgeCmd "hover/cmd/stage/purge"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stage <command>",
		Short: "Work with stages",
	}

	cmd.AddCommand(newCmd.Cmd())
	cmd.AddCommand(deleteCmd.Cmd())
	cmd.AddCommand(purgeCmd.Cmd())

	return cmd
}
