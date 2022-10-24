package aws

import (
	"context"
	"encoding/base64"
	"errors"
	awsLib "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cloudformationTypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrTypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmsTypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/smithy-go/ptr"
	"os"
	"strings"
)

type Aws struct {
	config *awsLib.Config

	ssmClient            *ssm.Client
	s3Client             *s3.Client
	ecrClient            *ecr.Client
	lambdaClient         *lambda.Client
	cloudformationClient *cloudformation.Client
	apiGatewayClient     *apigatewayv2.Client
	kmsClient            *kms.Client
}

func New(profile string, region string) (*Aws, error) {
	awsConfig, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region), config.WithSharedConfigProfile(profile))
	if err != nil {
		return nil, err
	}

	return &Aws{
		config: &awsConfig,
	}, nil
}

func (aws *Aws) BucketExists(name *string) bool {
	_, err := aws.s3().HeadBucket(context.Background(), &s3.HeadBucketInput{
		Bucket: name,
	})

	return err == nil
}

func (aws *Aws) CreateAssetsBucket(name *string, region *string) error {
	_, err := aws.s3().CreateBucket(context.Background(), &s3.CreateBucketInput{
		Bucket: name,
		CreateBucketConfiguration: &s3Types.CreateBucketConfiguration{
			LocationConstraint: s3Types.BucketLocationConstraint(*region),
		},
	})
	if err != nil {
		return err
	}

	_, err = aws.s3().PutBucketCors(context.Background(), &s3.PutBucketCorsInput{
		Bucket: name,
		CORSConfiguration: &s3Types.CORSConfiguration{
			CORSRules: []s3Types.CORSRule{
				{
					AllowedMethods: []string{"HEAD", "GET", "PUT", "POST"},
					AllowedOrigins: []string{"*"},
					AllowedHeaders: []string{"*"},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (aws *Aws) UploadFileToAssetsBucket(bucketName *string, fileName *string, file *os.File) error {
	_, err := aws.s3().PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: bucketName,
		Key:    fileName,
		ACL:    s3Types.ObjectCannedACLPublicRead,
		Body:   file,
	})
	if err != nil {
		return err
	}

	return nil
}

func (aws *Aws) WalkBucketObjects(bucketName *string, walker func(output *s3.ListObjectsV2Output) error) error {
	paginator := s3.NewListObjectsV2Paginator(aws.s3(), &s3.ListObjectsV2Input{
		Bucket: bucketName,
	})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.Background())
		if err != nil {
			return err
		}

		err = walker(output)
		if err != nil {
			return err
		}
	}
	return nil
}

func (aws *Aws) DeleteBucketObjects(bucketName *string, objects *[]s3Types.ObjectIdentifier) error {
	_, err := aws.s3().DeleteObjects(context.Background(), &s3.DeleteObjectsInput{
		Bucket: bucketName,
		Delete: &s3Types.Delete{
			Objects: *objects,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (aws *Aws) DeleteBucket(name *string) error {
	_, err := aws.s3().DeleteBucket(context.Background(), &s3.DeleteBucketInput{
		Bucket: name,
	})
	if err != nil {
		return err
	}

	return nil
}

func (aws *Aws) GetEcrRepository(name *string) (*ecr.DescribeRepositoriesOutput, error) {
	result, err := aws.ecr().DescribeRepositories(context.Background(), &ecr.DescribeRepositoriesInput{
		RepositoryNames: []string{*name},
	})

	return result, err
}

func (aws *Aws) EcrRepositoryDoesntExist(err error) bool {
	var repositoryNotfoundError *ecrTypes.RepositoryNotFoundException

	return errors.As(err, &repositoryNotfoundError)
}

func (aws *Aws) CreateEcrRepository(name *string) (*ecr.CreateRepositoryOutput, error) {
	result, err := aws.ecr().CreateRepository(context.Background(), &ecr.CreateRepositoryInput{
		RepositoryName:     name,
		ImageTagMutability: ecrTypes.ImageTagMutabilityImmutable,
	})

	return result, err
}

func (aws *Aws) DeleteRepository(name *string) error {
	_, err := aws.ecr().DeleteRepository(context.Background(), &ecr.DeleteRepositoryInput{
		RepositoryName: name,
		Force:          true,
	})

	return err
}

func (aws *Aws) GetEcrToken() (string, error) {
	result, err := aws.ecr().GetAuthorizationToken(context.Background(), &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return "", err
	}

	authorizationToken, err := base64.StdEncoding.DecodeString(*result.AuthorizationData[0].AuthorizationToken)
	if err != nil {
		return "", err
	}

	return strings.Replace(string(authorizationToken), "AWS:", "", 1), err
}

func (aws *Aws) GetEcrImages(name *string) (*ecr.DescribeImagesOutput, error) {
	result, err := aws.ecr().DescribeImages(context.Background(), &ecr.DescribeImagesInput{
		RepositoryName: name,
	})

	return result, err
}

func (aws *Aws) PurgeEcrImages(name *string, images *[]ecrTypes.ImageIdentifier) (*ecr.BatchDeleteImageOutput, error) {
	result, err := aws.ecr().BatchDeleteImage(context.Background(), &ecr.BatchDeleteImageInput{
		RepositoryName: name,
		ImageIds:       *images,
	})

	return result, err
}

func (aws *Aws) GetLambda(name *string, qualifier *string) (*lambda.GetFunctionOutput, error) {
	result, err := aws.lambda().GetFunction(context.Background(), &lambda.GetFunctionInput{
		FunctionName: name,
		Qualifier:    qualifier,
	})

	return result, err
}

func (aws *Aws) PublishLambdaVersion(name *string) (*lambda.PublishVersionOutput, error) {
	result, err := aws.lambda().PublishVersion(context.Background(), &lambda.PublishVersionInput{
		FunctionName: name,
	})

	return result, err
}

func (aws *Aws) InvokeLambda(name *string, version *string, payload []byte) (*lambda.InvokeOutput, error) {
	result, err := aws.lambda().Invoke(context.Background(), &lambda.InvokeInput{
		FunctionName: name,
		LogType:      "None",
		Payload:      payload,
		Qualifier:    version,
	})

	return result, err
}

func (aws *Aws) UpdateLambdaAlias(name *string, version *string, alias *string) (*lambda.UpdateAliasOutput, error) {
	result, err := aws.lambda().UpdateAlias(context.Background(), &lambda.UpdateAliasInput{
		FunctionName:    name,
		Name:            alias,
		FunctionVersion: version,
	})

	return result, err
}

func (aws *Aws) GetStack(name *string) (cloudformationTypes.Stack, error) {
	result, err := aws.cloudformation().DescribeStacks(context.Background(), &cloudformation.DescribeStacksInput{
		StackName: name,
	})
	if err != nil {
		return cloudformationTypes.Stack{}, err
	}

	return result.Stacks[0], nil
}

func (aws *Aws) StackDoesntExist(err error) bool {
	return strings.Contains(err.Error(), "does not exist")
}

func (aws *Aws) DeleteStack(name *string) (*cloudformation.DeleteStackOutput, error) {
	result, err := aws.cloudformation().DeleteStack(context.Background(), &cloudformation.DeleteStackInput{
		StackName: name,
	})

	return result, err
}

func (aws *Aws) UpdateStack(name *string, template *string) (*cloudformation.UpdateStackOutput, error) {
	result, err := aws.cloudformation().UpdateStack(context.Background(), &cloudformation.UpdateStackInput{
		StackName: name,
		Capabilities: []cloudformationTypes.Capability{
			cloudformationTypes.CapabilityCapabilityNamedIam,
		},
		TemplateBody: template,
	})

	return result, err
}

func (aws *Aws) CreateStack(name *string, template *string, roleArn *string) (*cloudformation.CreateStackOutput, error) {
	result, err := aws.cloudformation().CreateStack(context.Background(), &cloudformation.CreateStackInput{
		StackName: name,
		Capabilities: []cloudformationTypes.Capability{
			cloudformationTypes.CapabilityCapabilityNamedIam,
		},
		TemplateBody: template,
		RoleARN:      roleArn,
	})

	return result, err
}

func (aws *Aws) GetStackResources(name *string) (*cloudformation.DescribeStackResourcesOutput, error) {
	result, err := aws.cloudformation().DescribeStackResources(context.Background(), &cloudformation.DescribeStackResourcesInput{
		StackName: name,
	})

	return result, err
}

func (aws *Aws) GetStackEvents(name *string) (*cloudformation.DescribeStackEventsOutput, error) {
	result, err := aws.cloudformation().DescribeStackEvents(context.Background(), &cloudformation.DescribeStackEventsInput{
		StackName: name,
	})

	return result, err
}

func (aws *Aws) CreateKmsKey(name *string) error {
	result, err := aws.kms().CreateKey(context.Background(), &kms.CreateKeyInput{
		Description: name,
	})
	if err != nil {
		return err
	}

	_, err = aws.kms().CreateAlias(context.Background(), &kms.CreateAliasInput{
		AliasName:   ptr.String("alias/" + *name),
		TargetKeyId: result.KeyMetadata.KeyId,
	})
	if err != nil {
		return err
	}

	return nil
}

func (aws *Aws) GetKmsKey(name *string) (*kms.DescribeKeyOutput, error) {
	result, err := aws.kms().DescribeKey(context.Background(), &kms.DescribeKeyInput{
		KeyId: ptr.String("alias/" + *name),
	})

	return result, err
}

func (aws *Aws) KmsKeyDoesntExist(err error) bool {
	var notFoundError *kmsTypes.NotFoundException

	return errors.As(err, &notFoundError)
}

func (aws *Aws) EncryptWithKms(key string, value string) (*kms.EncryptOutput, error) {
	result, err := aws.kms().Encrypt(context.Background(), &kms.EncryptInput{
		KeyId:     ptr.String("alias/" + key),
		Plaintext: []byte(value),
	})

	return result, err
}

func (aws *Aws) DecryptWithKms(key string, encryptedValue []byte) (*kms.DecryptOutput, error) {
	result, err := aws.kms().Decrypt(context.Background(), &kms.DecryptInput{
		KeyId:          ptr.String("alias/" + key),
		CiphertextBlob: encryptedValue,
	})

	return result, err
}

func (aws *Aws) ssm() *ssm.Client {
	if aws.ssmClient == nil {
		aws.ssmClient = ssm.NewFromConfig(*aws.config)
	}

	return aws.ssmClient
}

func (aws *Aws) s3() *s3.Client {
	if aws.s3Client == nil {
		aws.s3Client = s3.NewFromConfig(*aws.config)
	}

	return aws.s3Client
}

func (aws *Aws) ecr() *ecr.Client {
	if aws.ecrClient == nil {
		aws.ecrClient = ecr.NewFromConfig(*aws.config)
	}

	return aws.ecrClient
}

func (aws *Aws) lambda() *lambda.Client {
	if aws.lambdaClient == nil {
		aws.lambdaClient = lambda.NewFromConfig(*aws.config)
	}

	return aws.lambdaClient
}

func (aws *Aws) cloudformation() *cloudformation.Client {
	if aws.cloudformationClient == nil {
		aws.cloudformationClient = cloudformation.NewFromConfig(*aws.config)
	}

	return aws.cloudformationClient
}

func (aws *Aws) apiGateway() *apigatewayv2.Client {
	if aws.apiGatewayClient == nil {
		aws.apiGatewayClient = apigatewayv2.NewFromConfig(*aws.config)
	}

	return aws.apiGatewayClient
}

func (aws *Aws) kms() *kms.Client {
	if aws.kmsClient == nil {
		aws.kmsClient = kms.NewFromConfig(*aws.config)
	}

	return aws.kmsClient
}
