package create

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
	"hover/aws"
	"hover/utils"
	"hover/utils/manifest"
)

type options struct {
	stage       string
	domain      string
	certificate string
}

func Cmd() *cobra.Command {
	opts := options{}

	cmd := &cobra.Command{
		Use:   "create <DOMAIN> <CERTIFICATE> --stage",
		Args:  cobra.ExactArgs(2),
		Short: "Create a new domain in ApiGateway",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.domain = args[0]
			opts.certificate = args[1]

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

	cname, err := ensureDomainExists(awsClient, o)
	if err != nil {
		return err
	}

	utils.PrintSuccess("Stage created")

	fmt.Println()

	pterm.DefaultTable.WithData(pterm.TableData{
		{"Region", pterm.FgYellow.Sprint(stage.Region)},
		{"Profile", pterm.FgYellow.Sprint(stage.AwsProfile)},
		{"Domain", pterm.FgYellow.Sprint(o.domain)},
		{"CNAME", pterm.FgYellow.Sprint(*cname)},
	}).Render()

	return nil
}

func ensureDomainExists(aws *aws.Aws, o *options) (*string, error) {
	result, err := aws.GetDomain(&o.domain)
	if err == nil {
		return getRegionalDomain(&result.DomainNameConfigurations), nil
	}

	if aws.DomainDoesntExist(err) {
		return createDomainName(aws, o)
	}

	return nil, err
}

func createDomainName(aws *aws.Aws, o *options) (*string, error) {
	result, err := aws.CreateDomain(&o.domain, &o.certificate)
	if err != nil {
		return nil, err
	}

	fmt.Println("Domain created")

	return getRegionalDomain(&result.DomainNameConfigurations), nil
}

func getRegionalDomain(configurations *[]types.DomainNameConfiguration) *string {
	index := slices.IndexFunc(*configurations, func(c types.DomainNameConfiguration) bool {
		return c.EndpointType == types.EndpointTypeRegional
	})

	configuration := (*configurations)[index]

	return configuration.ApiGatewayDomainName
}
