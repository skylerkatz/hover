package deploy

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/smithy-go/ptr"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"hover/aws"
	"hover/provisioner"
	"hover/utils"
	"hover/utils/manifest"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type options struct {
}

func Cmd() *cobra.Command {
	opts := options{}

	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy the current build to AWS",
		RunE: func(cmd *cobra.Command, args []string) error {

			return Run(&opts)
		},
	}

	return cmd
}

func Run(o *options) error {
	fmt.Println()

	stage, err := getBuildManifest()
	if err != nil {
		return err
	}

	utils.PrintStep("Deploying stage " + stage.Name)

	awsClient, _ := aws.New(stage.AwsProfile, stage.Region)

	repositoryUri, err := ensureEcrRepoExists(stage.Name, awsClient)
	if err != nil {
		return err
	}

	err = uploadAssets(stage, awsClient)
	if err != nil {
		return err
	}

	imageUri, err := pushDockerImage(stage, stage.BuildDetails.Id, awsClient, repositoryUri)
	if err != nil {
		return err
	}

	stack, resources, err := provisioner.Provision(stage, imageUri, awsClient)
	if err != nil {
		return err
	}

	err = publishNewLambdaVersions(stage, resources, awsClient)
	if err != nil {
		return err
	}

	table := pterm.DefaultTable

	tableData := pterm.TableData{}

	for _, output := range stack.Outputs {
		if *output.OutputKey == "Signature" {
			continue
		}

		tableData = append(tableData, []string{*output.Description, pterm.FgYellow.Sprint(*output.OutputValue)})
	}

	utils.PrintSuccess("Deployed to AWS Lambda")

	fmt.Println()

	table.WithData(tableData).Render()

	return nil
}

func getBuildManifest() (*manifest.Manifest, error) {
	path := filepath.Join(utils.Path.ApplicationOut, "hover_runtime", "manifest.json")

	manifestFile, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read manifest file at: %s. Error: %w", path, err)
	}

	var data manifest.Manifest

	err = json.Unmarshal(manifestFile, &data)
	if err != nil {
		return nil, fmt.Errorf("cannot read manifest file content. Error: %w", err)
	}

	if len(data.VPC.SecurityGroups) == 0 {
		data.VPC.SecurityGroups = []string{}
	}

	if len(data.VPC.Subnets) == 0 {
		data.VPC.Subnets = []string{}
	}

	return &data, nil
}

func uploadAssets(stage *manifest.Manifest, awsClient *aws.Aws) error {
	utils.PrintStep("Uploading assets")

	bucketName := stage.Name + "-assets"

	if !awsClient.BucketExists(&bucketName) {
		fmt.Println("Assets bucket doesn't exist. Creating...")

		err := awsClient.CreateAssetsBucket(&bucketName, &stage.Region)
		if err != nil {
			return err
		}
	}

	err := filepath.WalkDir(utils.Path.AssetsOut, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, _ := os.Open(path)

		defer file.Close()

		objectKey := stage.BuildDetails.Id + "/" + strings.TrimPrefix(path,
			utils.Path.AssetsOut+string(os.PathSeparator),
		)

		err = awsClient.UploadFileToAssetsBucket(&bucketName, &objectKey, file)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func pushDockerImage(stage *manifest.Manifest, buildId string, awsClient *aws.Aws, repositoryUri *string) (string, error) {
	utils.PrintStep("Pushing the container image")

	token, err := awsClient.GetEcrToken()
	if err != nil {
		return "", err
	}

	err = utils.Exec(fmt.Sprintf("docker tag %s %s",
		stage.Name+":latest",
		*repositoryUri+":"+buildId,
	), utils.Path.ApplicationOut)
	if err != nil {
		return "", err
	}

	err = utils.Exec(fmt.Sprintf("docker login --username AWS --password %s %s",
		token,
		strings.Split(*repositoryUri, "/")[0],
	), utils.Path.ApplicationOut)
	if err != nil {
		return "", err
	}

	err = utils.Exec(fmt.Sprintf("docker push %s",
		*repositoryUri+":"+buildId,
	), utils.Path.ApplicationOut)
	if err != nil {
		return "", err
	}

	return *repositoryUri + ":" + buildId, nil
}

func ensureEcrRepoExists(name string, awsClient *aws.Aws) (*string, error) {
	repository, err := awsClient.GetEcrRepository(&name)
	if err == nil {
		return repository.Repositories[0].RepositoryUri, nil
	}

	if awsClient.EcrRepositoryDoesntExist(err) {
		fmt.Println("Container repository doesn't exist. Creating...")

		result, creationErr := awsClient.CreateEcrRepository(&name)
		if creationErr != nil {
			return nil, creationErr
		}

		return result.Repository.RepositoryUri, nil
	}

	return nil, err
}

func publishNewLambdaVersions(stage *manifest.Manifest, resources *cloudformation.DescribeStackResourcesOutput, awsClient *aws.Aws) error {
	type function struct {
		functionType string
		resourceName *string
		functionName *string
		version      *string
	}

	utils.PrintStep("Publishing the new build")

	var functions []function

	for _, resource := range resources.StackResources {
		if *resource.ResourceType == "AWS::Lambda::Function" {
			nameSlices := strings.Split(*resource.PhysicalResourceId, "-")

			functions = append(functions, function{
				resourceName: resource.LogicalResourceId,
				functionName: resource.PhysicalResourceId,
				functionType: nameSlices[len(nameSlices)-1],
			})
		}
	}

	var waitGroup sync.WaitGroup

	waitGroup.Add(len(functions))

	publishingHasFailed := false

	for idx, aFunction := range functions {
		i := idx
		currentFunction := aFunction

		go func() {
			defer waitGroup.Done()

			versionResult, err := awsClient.PublishLambdaVersion(currentFunction.functionName)
			if err != nil {
				publishingHasFailed = true
				utils.PrintWarning(err.Error())
			}

			functions[i].version = versionResult.Version

			fmt.Println(fmt.Sprintf("Published version %s of the %s lambda", pterm.FgYellow.Sprint("#"+*versionResult.Version), pterm.FgYellow.Sprint(*currentFunction.functionName)))
		}()
	}

	waitGroup.Wait()

	if publishingHasFailed {
		return fmt.Errorf("failed to publish new lambda version")
	}

	for _, aFunction := range functions {
		currentFunction := aFunction

		if currentFunction.functionType == "http" {
			utils.PrintStep("Warming HTTP lambdas...")

			waitGroup.Add(stage.HTTP.Warm)

			for i := 0; i < stage.HTTP.Warm; i++ {

				go func() {
					defer waitGroup.Done()

					_, warmingError := awsClient.InvokeLambda(currentFunction.functionName,
						currentFunction.version,
						[]byte("{\"warmer_ping\": true}"),
					)
					if warmingError != nil {
						utils.PrintWarning(warmingError.Error())
					}
				}()
			}
		}

		if currentFunction.functionType == "cli" {
			utils.PrintStep("Running deploy commands")

			for _, command := range stage.DeployCommands {
				output, exitCode, err := utils.RunCommand(currentFunction.functionName, currentFunction.version, strings.TrimPrefix(command, "php artisan"), awsClient)
				if err != nil {
					return err
				}

				fmt.Print(output)

				if fmt.Sprint(exitCode) != "0" {
					return fmt.Errorf("failed to run %s", command)
				}
			}
		}
	}

	waitGroup.Wait()

	waitGroup.Add(len(functions))

	utils.PrintStep("Activating the new version...")

	for _, aFunction := range functions {
		currentFunction := aFunction

		go func() {
			defer waitGroup.Done()

			_, err := awsClient.UpdateLambdaAlias(currentFunction.functionName, currentFunction.version, ptr.String("live"))
			if err != nil {
				utils.PrintWarning(err.Error())
			}
		}()
	}

	waitGroup.Wait()

	return nil
}
