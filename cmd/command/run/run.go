package run

import (
	"fmt"
	"github.com/aws/smithy-go/ptr"
	"github.com/spf13/cobra"
	"hover/aws"
	"hover/provisioner"
	"hover/utils"
	"hover/utils/manifest"
	"strings"
)

type options struct {
	stage   string
	command string
}

func Cmd() *cobra.Command {
	opts := options{}

	cmd := &cobra.Command{
		Use:   "run <COMMAND> --stage",
		Args:  cobra.ExactArgs(1),
		Short: "Run a command on the CLI lambda of the specified stage",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.command = args[0]

			if opts.stage == "" {
				return fmt.Errorf("you must specify a --stage")
			}

			return Run(&opts)
		},
	}

	cmd.Flags().StringVarP(&opts.stage, "stage", "s", "", "The stage name")

	return cmd
}

func Run(o *options) error {
	fmt.Println()

	stage, err := manifest.Get(o.stage)
	if err != nil {
		return err
	}

	awsClient, _ := aws.New(stage.AwsProfile, stage.Region)

	output, exitCode, err := utils.RunCommand(
		ptr.String(provisioner.GetLambdaFunctionName(stage.Name, "cli")),
		ptr.String("live"),
		strings.TrimPrefix(o.command, "php artisan"),
		awsClient,
	)

	if output != "nil" {
		fmt.Print(output)
	}

	if fmt.Sprint(exitCode) != "0" {
		return fmt.Errorf("failed to run %s", o.command)
	}

	utils.PrintSuccess("Executed command 'php artisan " + o.command + "'")

	return nil
}
