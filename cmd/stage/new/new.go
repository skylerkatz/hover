package new

import (
	"fmt"
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"hover/embeds"
	"hover/utils"
	"os"
	"path/filepath"
	"strings"
)

type options struct {
	alias   string
	project string
	region  string
	profile string
}

func Cmd() *cobra.Command {
	opts := options{}

	cmd := &cobra.Command{
		Use:   "new <PROJECT> <ALIAS>",
		Args:  cobra.ExactArgs(2),
		Short: "Create a manifest file for a new stage",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.project = args[0]
			opts.alias = args[1]

			return Run(&opts)
		},
	}

	return cmd
}

func Run(o *options) error {
	fmt.Println()

	manifestPath := filepath.Join(utils.Path.Hover, o.alias+".yml")

	if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
		return fmt.Errorf("A manifest file already exists with the same alias")
	}

	fmt.Println()

	utils.PrintInfo(heredoc.Doc(`
			To create a new stage, you need to specify the AWS region, the AWS credentials profile
			and two IAM roles. One to be assumed by CloudFormation and one by Lambda.
`))

	fmt.Println()

	region, _ := pterm.DefaultInteractiveTextInput.Show("AWS region")
	awsProfile, _ := pterm.DefaultInteractiveTextInput.Show("AWS credentials profile")
	stackRole, _ := pterm.DefaultInteractiveTextInput.Show("Stack execution IAM role ARN")
	lambdaRole, _ := pterm.DefaultInteractiveTextInput.Show("Lambda execution IAM role ARN")

	replacer := strings.NewReplacer(
		"stage-name",
		o.project+"-"+o.alias,
		"aws-region-name",
		region,
		"aws-profile-name",
		awsProfile,
		"stack-role-arn",
		stackRole,
		"lambda-role-arn",
		lambdaRole,
	)

	manifest := replacer.Replace(embeds.HoverManifest)

	if _, err := os.Stat(utils.Path.Hover); os.IsNotExist(err) {
		err = os.Mkdir(utils.Path.Hover, os.ModePerm)
		if err != nil {
			return err
		}

		err = os.WriteFile(filepath.Join(utils.Path.Hover, ".gitignore"), []byte(heredoc.Doc(`
		/out
		*-secrets.plain.env
		`)), os.ModePerm)
		if err != nil {
			return err
		}
	}

	err := os.WriteFile(manifestPath, []byte(manifest), os.ModePerm)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(utils.Path.Hover, o.alias+"-secrets.plain.env"), []byte(heredoc.Doc(`
			# Use this environment file to declare your secret environment variables.
			# Once ready, run "hover secret encrypt --stage=`+o.alias+`" to encrypt
			# the file so it can safely be packaged into the Docker image.
			
			APP_KEY=
			DB_PASSWORD=
	`)), os.ModePerm)
	if err != nil {
		return err
	}

	if _, err = os.Stat(filepath.Join(utils.Path.Hover, ".Dockerfile")); os.IsNotExist(err) {
		err = os.WriteFile(filepath.Join(utils.Path.Hover, ".Dockerfile"), []byte(embeds.HoverDockerfile), os.ModePerm)
		if err != nil {
			return err
		}
	}

	if _, err = os.Stat(filepath.Join(utils.Path.Current, ".dockerignore")); os.IsNotExist(err) {
		err = os.WriteFile(filepath.Join(utils.Path.Current, ".dockerignore"), []byte(heredoc.Doc(`
			.git
			.idea
			node_modules
			vendor
	`)), os.ModePerm)
		if err != nil {
			return err
		}
	}

	utils.PrintSuccess("A manifest file was created in " + manifestPath)

	fmt.Println()

	utils.PrintWarning(heredoc.Doc(`
			Hover requires the following composer dependencies:
			- hollodotme/fast-cgi-client:^3.1
			- guzzlehttp/promises:^1.5
			- aws/aws-sdk-php:^3.2
			
			You may run the following composer command to require these in your composer.json file:
			▶ composer require hollodotme/fast-cgi-client:^3.1 guzzlehttp/promises:^1.5 aws/aws-sdk-php:^3.2
`))

	return nil
}
