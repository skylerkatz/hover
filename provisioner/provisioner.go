package provisioner

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/pterm/pterm"
	"golang.org/x/exp/maps"
	"hover/aws"
	"hover/utils"
	"hover/utils/manifest"
	"strconv"
	"strings"
	"time"
)

func Provision(manifest *manifest.Manifest, imageUri string, aws *aws.Aws) (*types.Stack, *cloudformation.DescribeStackResourcesOutput, error) {
	utils.PrintStep("Provisioning the stack")

	template := getTemplate(manifest, imageUri, manifest.BuildDetails.Hash)
	currentStack, err := getCloudFormationStack(manifest.Name, aws)
	if err != nil {
		return nil, nil, err
	}

	if currentStack.StackId != nil {
		if currentStack.StackStatus == types.StackStatusRollbackComplete {
			_, err = aws.DeleteStack(&manifest.Name)
			if err != nil {
				return nil, nil, err
			}

			return nil, nil, fmt.Errorf("a failed stack is being deleted. Try again in a bit")
		}

		stackResources, _ := aws.GetStackResources(&manifest.Name)

		_, err = aws.UpdateStack(&manifest.Name, template)
		if err != nil {
			if strings.Contains(err.Error(), "No updates are to be performed") {
				fmt.Println("No stack changes to perform")

				return &currentStack, stackResources, nil
			} else {
				return nil, nil, err
			}
		}
	} else {
		_, err = aws.CreateStack(&manifest.Name, template, &manifest.Auth.StackRole)
		if err != nil {
			return nil, nil, err
		}
	}

	spinner, _ := pterm.DefaultSpinner.Start("Updating the CloudFormation stack...")

	time.Sleep(5 * time.Second)

	for {
		results, err := aws.GetStack(&manifest.Name)
		if err != nil {
			return nil, nil, err
		}

		switch results.StackStatus {
		case
			types.StackStatusCreateComplete,
			types.StackStatusUpdateComplete,
			types.StackStatusUpdateRollbackComplete:
			resources, _ := aws.GetStackResources(&manifest.Name)
			spinner.Stop()
			return &results, resources, nil
		case
			types.StackStatusUpdateInProgress,
			types.StackStatusCreateInProgress,
			types.StackStatusUpdateCompleteCleanupInProgress:
		default:
			return nil, nil, printError(manifest, aws)
		}

		time.Sleep(5 * time.Second)
	}
}

func printError(manifest *manifest.Manifest, aws *aws.Aws) error {
	events, _ := aws.GetStackEvents(&manifest.Name)

	for _, event := range events.StackEvents {
		if event.ResourceStatus == types.ResourceStatusUpdateInProgress && *event.ResourceType == "AWS::CloudFormation::Stack" {
			break
		}

		switch event.ResourceStatus {
		case
			types.ResourceStatusCreateFailed,
			types.ResourceStatusDeleteFailed,
			types.ResourceStatusUpdateFailed:
			if *event.ResourceStatusReason != "Resource update cancelled" {
				pterm.Error.Println(*event.PhysicalResourceId + ": " + *event.ResourceStatusReason)
			}
		default:
			continue
		}
	}

	return fmt.Errorf("stack '" + manifest.Name + "' provisioning failed")
}

func getCloudFormationStack(name string, aws *aws.Aws) (types.Stack, error) {
	response, err := aws.GetStack(&name)
	if err == nil {
		return response, nil
	}

	if aws.StackDoesntExist(err) {
		return types.Stack{}, nil
	}

	return types.Stack{}, err
}

func GetLambdaFunctionName(stageName string, functionName string) string {
	return stageName + "-" + functionName
}

func getTemplate(manifest *manifest.Manifest, imageUri string, manifestHash string) *string {
	template := map[string]any{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Resources":                map[string]any{},
		"Outputs":                  map[string]any{},
	}

	var resources map[string]any
	var outputs map[string]any

	resources = map[string]any{}

	outputs = map[string]any{
		"Signature": map[string]string{
			"Value": manifestHash,
		},
	}

	maps.Copy(resources, lambdaFunction("HTTPLambda", "http", imageUri, manifest, manifest.HTTP.Timeout, manifest.HTTP.Memory, manifest.HTTP.Concurrency))
	maps.Copy(resources, lambdaAlias("HTTPLambda", "HTTPLambdaLiveAlias"))
	maps.Copy(resources, warmer("HTTPLambda", "HTTPLambdaLiveAlias", manifest))
	maps.Copy(resources, apiGateway("HTTPLambda", "HTTPLambdaLiveAlias", manifest))
	maps.Copy(resources, lambdaFunction("CliLambda", "cli", imageUri, manifest, manifest.Cli.Timeout, manifest.Cli.Memory, manifest.Cli.Concurrency))
	maps.Copy(resources, lambdaAlias("CliLambda", "CliLambdaLiveAlias"))
	maps.Copy(resources, scheduler("CliLambda", manifest))
	maps.Copy(resources, cloudFrontDistribution("ApiGateway", manifest))

	for queueFunctionName, queueConfiguration := range manifest.Queue {
		maps.Copy(resources, lambdaFunction(queueFunctionName+"QueueLambda", queueFunctionName+"-queue", imageUri, manifest, queueConfiguration.Timeout, queueConfiguration.Memory, queueConfiguration.Concurrency))
		maps.Copy(resources, lambdaAlias(queueFunctionName+"QueueLambda", queueFunctionName+"LambdaLiveAlias"))

		for _, queueName := range queueConfiguration.Queues {
			maps.Copy(resources, queue(queueFunctionName+"QueueLambda", queueName+"Queue", queueName, manifest, queueConfiguration.Timeout+10))
		}
	}

	maps.Copy(outputs, map[string]any{
		"ApiGatewayUri": map[string]any{
			"Description": "Internal Domain",
			"Value": map[string]any{
				"Fn::GetAtt": []string{"ApiGateway", "ApiEndpoint"},
			},
		},
		"AssetsDomain": map[string]any{
			"Description": "Assets Domain",
			"Value": map[string]any{
				"Fn::GetAtt": []string{"CFDistribution", "DomainName"},
			},
		},
	})

	template["Resources"] = resources
	template["Outputs"] = outputs

	jsonPrint, _ := json.MarshalIndent(template, "", " ")

	jsonPrintString := string(jsonPrint)

	return &jsonPrintString
}

func lambdaFunction(resourceName string, functionName string, imageUri string, manifest *manifest.Manifest, timeout int, memory int, concurrency int) map[string]any {
	result := map[string]any{
		resourceName: map[string]any{
			"Type": "AWS::Lambda::Function",
			"Properties": map[string]any{
				"FunctionName": GetLambdaFunctionName(manifest.Name, functionName),
				"Role":         manifest.Auth.LambdaRole,
				"Environment": map[string]any{
					"Variables": map[string]any{
						"SQS_PREFIX": map[string]any{
							"Fn::Join": []any{
								"",
								[]any{
									"https://",
									"sqs.",
									map[string]any{
										"Ref": "AWS::Region",
									},
									".",
									map[string]any{
										"Ref": "AWS::URLSuffix",
									},
									"/",
									map[string]any{
										"Ref": "AWS::AccountId",
									},
								},
							},
						},
						"ASSET_URL": map[string]any{
							"Fn::Join": []any{"/",
								[]any{
									"assets",
									manifest.BuildDetails.Id,
								},
							},
						},
						"SQS_SUFFIX":   "-" + manifest.Name,
						"CACHE_PREFIX": manifest.Name,
						"CF_DOMAIN": map[string]any{
							"Fn::GetAtt": []string{"CFDistribution", "DomainName"},
						},
						"APP_CONFIG_CACHE": "/tmp/storage/bootstrap/cache/config.php",
						"APP_EVENTS_CACHE": "/tmp/storage/bootstrap/cache/events.php",
						"APP_ROUTES_CACHE": "/tmp/storage/bootstrap/cache/routes-v7.php",
					},
				},
				"PackageType": "Image",
				"Code": map[string]any{
					"ImageUri": imageUri,
				},
				"VpcConfig": map[string]any{
					"SecurityGroupIds": manifest.VPC.SecurityGroups,
					"SubnetIds":        manifest.VPC.Subnets,
				},
			},
		},
		resourceName + "LogGroup": map[string]any{
			"Type": "AWS::Logs::LogGroup",
			"Properties": map[string]any{
				"LogGroupName": map[string]any{
					"Fn::Join": []any{
						"/",
						[]any{
							"/aws/lambda",
							map[string]any{
								"Ref": resourceName,
							},
						},
					},
				},
				"RetentionInDays": 14,
			},
		},
	}

	if concurrency != 0 {
		result[resourceName].(map[string]any)["Properties"].(map[string]any)["ReservedConcurrentExecutions"] = concurrency
	}

	if timeout != 0 {
		result[resourceName].(map[string]any)["Properties"].(map[string]any)["Timeout"] = timeout
	}

	if memory != 0 {
		result[resourceName].(map[string]any)["Properties"].(map[string]any)["MemorySize"] = memory
	}

	return result
}

func lambdaAlias(httpLambdaResourceName string, resourceName string) map[string]any {
	return map[string]any{
		resourceName: map[string]any{
			"Type": "AWS::Lambda::Alias",
			"Properties": map[string]any{
				"FunctionName": map[string]any{
					"Ref": httpLambdaResourceName,
				},
				"FunctionVersion": "$LATEST",
				"Name":            "live",
			},
		},
	}
}

func warmer(httpLambdaResourceName string, httpLambdaAliasResourceName string, manifest *manifest.Manifest) map[string]any {
	warm := "1"

	if manifest.HTTP.Warm != 0 {
		warm = strconv.Itoa(manifest.HTTP.Warm)
	}

	return map[string]any{
		"WarmerEventRule": map[string]any{
			"Type": "AWS::Events::Rule",
			"DependsOn": []any{
				httpLambdaAliasResourceName,
			},
			"Properties": map[string]any{
				"Name":               manifest.Name + "-warmer",
				"ScheduleExpression": "rate(5 minutes)",
				"State":              "ENABLED",
				"Targets": []any{
					map[string]any{
						"Arn": map[string]any{
							"Fn::Join": []any{
								":",
								[]any{
									map[string]any{
										"Fn::GetAtt": []any{
											httpLambdaResourceName,
											"Arn",
										},
									},
									"live",
								},
							},
						},
						"Id":    "hover-warmer",
						"Input": "{\"warmer\": true, \"containers\": " + warm + "}",
					},
				},
			},
		},
		"WarmerEventInvokePermission": map[string]any{
			"Type": "AWS::Lambda::Permission",
			"DependsOn": []any{
				httpLambdaAliasResourceName,
			},
			"Properties": map[string]any{
				"FunctionName": map[string]any{
					"Fn::Join": []any{
						":",
						[]any{
							map[string]any{
								"Ref": httpLambdaResourceName,
							},
							"live",
						},
					},
				},
				"Action":    "lambda:InvokeFunction",
				"Principal": "events.amazonaws.com",
				"SourceArn": map[string]any{
					"Fn::GetAtt": []any{
						"WarmerEventRule",
						"Arn",
					},
				},
			},
		},
	}
}

func scheduler(cliLambdaResourceName string, manifest *manifest.Manifest) map[string]any {
	return map[string]any{
		"SchedulerEventRule": map[string]any{
			"Type": "AWS::Events::Rule",
			"Properties": map[string]any{
				"Name":               manifest.Name + "-scheduler",
				"ScheduleExpression": "rate(1 minute)",
				"State":              "ENABLED",
				"Targets": []any{
					map[string]any{
						"Arn": map[string]any{
							"Fn::Join": []any{
								":",
								[]any{
									map[string]any{
										"Fn::GetAtt": []any{
											cliLambdaResourceName,
											"Arn",
										},
									},
									"live",
								},
							},
						},
						"Id":    "hover-scheduler",
						"Input": "{\"command\": \"schedule:run\"}",
					},
				},
			},
		},
		"SchedulerEventRuleInvokePermission": map[string]any{
			"Type": "AWS::Lambda::Permission",
			"Properties": map[string]any{
				"FunctionName": map[string]any{
					"Fn::Join": []any{
						":",
						[]any{
							map[string]any{
								"Ref": cliLambdaResourceName,
							},
							"live",
						},
					},
				},
				"Action":    "lambda:InvokeFunction",
				"Principal": "events.amazonaws.com",
				"SourceArn": map[string]any{
					"Fn::GetAtt": []any{
						"SchedulerEventRule",
						"Arn",
					},
				},
			},
		},
	}
}

func queue(queueLambdaResourceName string, resourceName string, queueName string, manifest *manifest.Manifest, visibilityTimeout int) map[string]any {
	if visibilityTimeout == 0 {
		visibilityTimeout = 3
	}

	return map[string]any{
		resourceName: map[string]any{
			"Type": "AWS::SQS::Queue",
			"Properties": map[string]any{
				"QueueName":         queueName + "-" + manifest.Name,
				"VisibilityTimeout": visibilityTimeout,
			},
		},
		resourceName + "QueueSourceMapping": map[string]any{
			"Type": "AWS::Lambda::EventSourceMapping",
			"Properties": map[string]any{
				"BatchSize": 1,
				"FunctionResponseTypes": []any{
					"ReportBatchItemFailures",
				},
				"EventSourceArn": map[string]any{
					"Fn::GetAtt": []any{
						resourceName,
						"Arn",
					},
				},
				"FunctionName": map[string]any{
					"Fn::Join": []any{
						":",
						[]any{
							map[string]any{
								"Ref": queueLambdaResourceName,
							},
							"live",
						},
					},
				},
			},
		},
	}
}

func apiGateway(httpLambdaResourceName string, httpLambdaAliasResourceName string, manifest *manifest.Manifest) map[string]any {
	return map[string]any{
		"ApiGateway": map[string]any{
			"Type": "AWS::ApiGatewayV2::Api",
			"Properties": map[string]any{
				"Name":         manifest.Name + "-api",
				"ProtocolType": "HTTP",
			},
		},
		"ApiGatewayLambdaIntegration": map[string]any{
			"Type": "AWS::ApiGatewayV2::Integration",
			"DependsOn": []any{
				httpLambdaAliasResourceName,
			},
			"Properties": map[string]any{
				"ApiId": map[string]any{
					"Ref": "ApiGateway",
				},
				"IntegrationType":      "AWS_PROXY",
				"IntegrationMethod":    "POST",
				"PayloadFormatVersion": "2.0",
				"IntegrationUri": map[string]any{
					"Fn::Sub": []any{
						"arn:aws:apigateway:${AWS::Region}:lambda:path/2015-03-31/functions/${functionArn}:live/invocations",
						map[string]any{
							"functionArn": map[string]any{
								"Fn::GetAtt": []any{
									httpLambdaResourceName,
									"Arn",
								},
							},
						},
					},
				},
			},
		},
		"ApiGatewayRoute": map[string]any{
			"Type": "AWS::ApiGatewayV2::Route",
			"DependsOn": []any{
				"ApiGatewayLambdaIntegration",
			},
			"Properties": map[string]any{
				"ApiId": map[string]any{
					"Ref": "ApiGateway",
				},
				"RouteKey":          "$default",
				"AuthorizationType": "NONE",
				"Target": map[string]any{
					"Fn::Join": []any{
						"/",
						[]any{
							"integrations",
							map[string]any{
								"Ref": "ApiGatewayLambdaIntegration",
							},
						},
					},
				},
			},
		},
		"ApiGatewayStage": map[string]any{
			"Type": "AWS::ApiGatewayV2::Stage",
			"Properties": map[string]any{
				"ApiId": map[string]any{
					"Ref": "ApiGateway",
				},
				"StageName":  "$default",
				"AutoDeploy": true,
			},
		},
		"ApiGatewayDeployment": map[string]any{
			"Type": "AWS::ApiGatewayV2::Deployment",
			"DependsOn": []any{
				"ApiGatewayRoute",
			},
			"Properties": map[string]any{
				"ApiId": map[string]any{
					"Ref": "ApiGateway",
				},
			},
		},
		"ApiGatewayInvokePermission": map[string]any{
			"Type": "AWS::Lambda::Permission",
			"DependsOn": []any{
				httpLambdaAliasResourceName,
			},
			"Properties": map[string]any{
				"Action": "lambda:InvokeFunction",
				"FunctionName": map[string]any{
					"Fn::Join": []any{
						":",
						[]any{
							map[string]any{
								"Ref": httpLambdaResourceName,
							},
							"live",
						},
					},
				},
				"Principal": "apigateway.amazonaws.com",
				"SourceArn": map[string]any{
					"Fn::Join": []any{
						"",
						[]any{
							"arn:aws:execute-api:",
							map[string]any{
								"Ref": "AWS::Region",
							},
							":",
							map[string]any{
								"Ref": "AWS::AccountId",
							},
							":",
							map[string]any{
								"Ref": "ApiGateway",
							},
							"/*",
						},
					},
				},
			},
		},
	}
}

func cloudFrontDistribution(apiGatewayResourceName string, manifest *manifest.Manifest) map[string]any {
	output := map[string]any{
		"CFDistribution": map[string]any{
			"Type": "AWS::CloudFront::Distribution",
			"DependsOn": []any{
				apiGatewayResourceName,
			},
			"Properties": map[string]any{
				"DistributionConfig": map[string]any{
					"HttpVersion": "http2",
					"Origins": []any{
						map[string]any{
							"Id":         "assets-bucket",
							"DomainName": fmt.Sprintf("%s-assets.s3.%s.amazonaws.com", manifest.Name, manifest.Region),
							"S3OriginConfig": map[string]any{
								"OriginAccessIdentity": "",
							},
						},
						map[string]any{
							"Id": "gateway",
							"DomainName": map[string]any{
								"Fn::Select": []any{"1", map[string]any{
									"Fn::Split": []any{"//", map[string]any{
										"Fn::GetAtt": []any{
											apiGatewayResourceName,
											"ApiEndpoint",
										},
									}},
								}},
							},
							"CustomOriginConfig": map[string]any{
								"OriginProtocolPolicy": "https-only",
								"OriginSSLProtocols":   []string{"TLSv1.2"},
							},
						},
					},
					"Enabled": "true",
					"Comment": manifest.Name + "-assets",
					"DefaultCacheBehavior": map[string]any{
						"AllowedMethods":       []any{"GET", "HEAD", "OPTIONS", "PUT", "PATCH", "POST", "DELETE"},
						"TargetOriginId":       "gateway",
						"CachePolicyId":        "b2884449-e4de-46a7-ac36-70bc7f1ddd6d",
						"ViewerProtocolPolicy": "redirect-to-https",
					},
					"CacheBehaviors": []any{
						map[string]any{
							"AllowedMethods":        []any{"GET", "HEAD", "OPTIONS"},
							"TargetOriginId":        "assets-bucket",
							"PathPattern":           "/assets/*",
							"CachePolicyId":         "658327ea-f89d-4fab-a63d-7e88639e58f6",
							"OriginRequestPolicyId": "88a5eaf4-2fd4-4709-b370-b4c650ea3fcf",
							"ViewerProtocolPolicy":  "redirect-to-https",
						},
					},
				},
			},
		},
	}

	if len(manifest.HTTP.Domains) > 0 {
		output["CFDistribution"].(map[string]any)["Properties"].(map[string]any)["DistributionConfig"].(map[string]any)["Aliases"] = strings.Split(strings.ReplaceAll(manifest.HTTP.Domains, " ", ""), ",")
		output["CFDistribution"].(map[string]any)["Properties"].(map[string]any)["DistributionConfig"].(map[string]any)["ViewerCertificate"] = map[string]any{"" +
			"AcmCertificateArn": "arn:aws:acm:us-east-1:324027754711:certificate/e3109ab1-3ca4-4f33-b580-620bfdaf7617",
			"SslSupportMethod": "sni-only",
		}
	}

	return output
}
