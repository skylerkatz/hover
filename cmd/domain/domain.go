package domain

import (
	"github.com/spf13/cobra"
	createCmd "hover/cmd/domain/create"
	deleteCmd "hover/cmd/domain/delete"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain <command>",
		Short: "Work with ApiGateway custom domains",
	}

	cmd.AddCommand(createCmd.Cmd())
	cmd.AddCommand(deleteCmd.Cmd())

	return cmd
}
