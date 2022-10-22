package secret

import (
	"github.com/spf13/cobra"
	decryptCmd "hover/cmd/secret/decrypt"
	encryptCmd "hover/cmd/secret/encrypt"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secret <command>",
		Short: "Work with secrets",
	}

	cmd.AddCommand(encryptCmd.Cmd())
	cmd.AddCommand(decryptCmd.Cmd())

	return cmd
}
