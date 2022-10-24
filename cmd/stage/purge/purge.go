package purge

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/ptr"
	"github.com/spf13/cobra"
	"hover/aws"
	"hover/provisioner"
	"hover/utils"
	"hover/utils/manifest"
	"log"
	"sort"
	"strings"
	"sync"
)

type options struct {
	alias string
}

func Cmd() *cobra.Command {
	opts := options{}

	cmd := &cobra.Command{
		Use:   "purge <ALIAS>",
		Args:  cobra.ExactArgs(1),
		Short: "Delete expired asset files and container images",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.alias = args[0]

			return Run(&opts)
		},
	}

	return cmd
}

func Run(o *options) error {
	fmt.Println()

	stage, err := manifest.Get(o.alias)
	if err != nil {
		return err
	}

	awsClient, _ := aws.New(stage.AwsProfile, stage.Region)

	result, err := awsClient.GetLambda(ptr.String(provisioner.GetLambdaFunctionName(stage.Name, "cli")),
		ptr.String("live"),
	)
	if err != nil {
		return err
	}

	liveBuildId := strings.Split(*result.Code.ImageUri, ":")[1]

	images, err := awsClient.GetEcrImages(&stage.Name)
	if err != nil {
		return err
	}

	sort.Slice(images.ImageDetails, func(i, j int) bool {
		return images.ImageDetails[i].ImagePushedAt.Unix() > images.ImageDetails[j].ImagePushedAt.Unix()
	})

	var tagsToDelete []types.ImageIdentifier
	tagsToRetain := []*string{&liveBuildId}

	var waitGroup sync.WaitGroup

	for i, image := range images.ImageDetails {
		// Skip Deleting the latest X tags
		if i < 14 {
			for _, tag := range image.ImageTags {
				tagsToRetain = append(tagsToRetain, &tag)
			}
			continue
		}
		for _, tag := range image.ImageTags {
			if tag == liveBuildId {
				continue
			}

			tagsToDelete = append(tagsToDelete, types.ImageIdentifier{
				ImageTag: &tag,
			})

			fmt.Println("Purging ECR image: " + tag)
		}
	}

	if len(tagsToDelete) > 0 {
		waitGroup.Add(1)

		go deleteEcrTags(stage, &tagsToDelete, &waitGroup, awsClient)
	}

	err = awsClient.WalkBucketObjects(ptr.String(stage.Name+"-assets"), func(output *s3.ListObjectsV2Output) error {
		var objectsToDelete []s3Types.ObjectIdentifier

		for _, object := range output.Contents {
			shouldDelete := true
			for _, buildId := range tagsToRetain {
				if strings.HasPrefix(*object.Key, "assets/"+*buildId) {
					shouldDelete = false
					break
				}
			}

			if shouldDelete {
				objectsToDelete = append(objectsToDelete, s3Types.ObjectIdentifier{
					Key: object.Key,
				})

				fmt.Println("Purging S3 object: " + *object.Key)
			}
		}

		if len(objectsToDelete) > 0 {
			waitGroup.Add(1)

			go deleteS3Objects(stage, &objectsToDelete, &waitGroup, awsClient)
		}

		return nil
	})
	if err != nil {
		return err
	}

	waitGroup.Wait()

	utils.PrintSuccess("Stage purged")

	return nil
}

func deleteEcrTags(stage *manifest.Manifest, tagsToDelete *[]types.ImageIdentifier, waitGroup *sync.WaitGroup, awsClient *aws.Aws) {
	defer waitGroup.Done()

	_, err := awsClient.PurgeEcrImages(&stage.Name, tagsToDelete)
	if err != nil {
		log.Println(err)
	}
}

func deleteS3Objects(stage *manifest.Manifest, objectsToDelete *[]s3Types.ObjectIdentifier, waitGroup *sync.WaitGroup, awsClient *aws.Aws) {
	defer waitGroup.Done()

	err := awsClient.DeleteBucketObjects(ptr.String(stage.Name+"-assets"), objectsToDelete)
	if err != nil {
		log.Println(err)
	}
}
