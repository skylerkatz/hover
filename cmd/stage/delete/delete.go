package delete

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/ptr"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"hover/aws"
	"hover/utils"
	"hover/utils/manifest"
	"os"
	"path/filepath"
	"strings"
)

type options struct {
	alias string
}

func Cmd() *cobra.Command {
	opts := options{}

	cmd := &cobra.Command{
		Use:   "delete <ALIAS>",
		Args:  cobra.ExactArgs(1),
		Short: "Delete a stage with all its AWS resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.alias = args[0]

			return Run(&opts)
		},
	}

	return cmd
}

func Run(o *options) error {
	fmt.Println()

	result, _ := pterm.DefaultInteractiveConfirm.Show(fmt.Sprintf("Are you sure you want to delete stage `%s`?", o.alias))
	if !result {
		fmt.Println("abort")
		os.Exit(0)
	}

	stage, err := manifest.Get(o.alias)
	if err != nil {
		return err
	}

	awsClient, _ := aws.New(stage.AwsProfile, stage.Region)

	_, err = awsClient.GetStack(&stage.Name)
	if err == nil {
		err := deleteStack(stage, awsClient)
		if err != nil {
			return err
		}
	} else {
		if strings.Contains(err.Error(), "does not exist") {
			fmt.Println("A CloudFormation stack doesn't exist. Skipping")
		} else {
			return fmt.Errorf("unable to read the Cloudformation stack in AWS. Probably a credentials issue. Error: %w", err)
		}
	}

	if err = deleteEcrRepo(stage, awsClient); err != nil {
		return err
	}

	if err = deleteAssetsBucket(stage, awsClient); err != nil {
		return err
	}

	if err = os.RemoveAll(filepath.Join(utils.Path.Hover, o.alias+".yml")); err != nil {
		return err
	}

	utils.PrintInfo("It may take a few minutes for the CloudFormation stack to be completely deleted in AWS.")

	return nil
}

func deleteEcrRepo(stage *manifest.Manifest, aws *aws.Aws) error {
	utils.PrintStep("Deleting ECR repository")

	err := aws.DeleteRepository(&stage.Name)
	if err != nil {
		return err
	}

	return nil
}

func deleteAssetsBucket(stage *manifest.Manifest, aws *aws.Aws) error {
	utils.PrintStep("Deleting assets bucket")

	err := aws.WalkBucketObjects(ptr.String(stage.Name+"-assets"), func(output *s3.ListObjectsV2Output) error {
		var objectsToDelete []types.ObjectIdentifier

		for _, object := range output.Contents {
			objectsToDelete = append(objectsToDelete, types.ObjectIdentifier{
				Key: object.Key,
			})
		}

		if len(objectsToDelete) > 0 {
			err := aws.DeleteBucketObjects(ptr.String(stage.Name+"-assets"), &objectsToDelete)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	err = aws.DeleteBucket(ptr.String(stage.Name + "-assets"))
	if err != nil {
		return err
	}

	return nil
}

func deleteStack(stage *manifest.Manifest, aws *aws.Aws) error {
	_, err := aws.DeleteStack(&stage.Name)
	if err != nil {
		return err
	}

	return nil
}
