package delete

import (
	"fmt"
	"github.com/spf13/cobra"
	"hover/aws"
	"hover/utils"
	"hover/utils/manifest"
)

type options struct {
	stage  string
	domain string
}

func Cmd() *cobra.Command {
	opts := options{}

	cmd := &cobra.Command{
		Use:   "delete <DOMAIN> --stage",
		Args:  cobra.ExactArgs(1),
		Short: "Delete a custom domain from ApiGateway",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.domain = args[0]

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

	mappings, err := awsClient.GetDomainMappings(&o.domain)
	if err != nil {
		return err
	}

	if len(mappings.Items) > 0 {
		return fmt.Errorf("cannot delete a domain that is already mapped. Detach it from API `%s` first", *mappings.Items[0].ApiId)
	}

	_, err = awsClient.DeleteDomain(&o.domain)
	if err != nil {
		return fmt.Errorf("couldn't delete the domain. Error: %w", err)
	}

	utils.PrintSuccess("Stage deleted")

	return nil
}
