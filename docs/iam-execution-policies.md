# IAM Management & Execution Policies

There are three policies needed for Hover to operate:

1. A policy that allows CloudFormation to perform the deployment.
2. A policy that allows Hover to manage the different resources and start the deployment.
3. A policy that allows your lambda functions to perform different actions.

The first policy will be attached to a role that will be assumed by CloudFormation, the second will be attached to the IAM user Hover uses and the third will be assumed by Lambda.

If you feel confident, you may attach the `AdministratorAccess` policy to the different users and roles. But if you want granular control, continue reading.

## CloudFormation Execution Role & Policy

This role will be assumed by CloudFormation. It will govern what the CloudFormation stack can do to your AWS account.

For a quick start, create a policy named `default-stack-execution` and use the following permissions:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "events",
            "Effect": "Allow",
            "Action": [
                "events:*"
            ],
            "Resource": "*"
        },
        {
            "Sid": "logs",
            "Effect": "Allow",
            "Action": [
                "logs:*"
            ],
            "Resource": "*"
        },
        {
            "Sid": "iam",
            "Effect": "Allow",
            "Action": [
                "iam:PassRole"
            ],
            "Resource": "*"
        },
        {
            "Sid": "lambda",
            "Effect": "Allow",
            "Action": [
                "lambda:*"
            ],
            "Resource": [
                "*"
            ]
        },
        {
            "Sid": "ecr",
            "Effect": "Allow",
            "Action": [
                "ecr:*"
            ],
            "Resource": "*"
        },
        {
            "Sid": "apigateway",
            "Effect": "Allow",
            "Action": [
                "apigateway:*"
            ],
            "Resource": [
                "*"
            ]
        },
        {
            "Sid": "cloudfront",
            "Effect": "Allow",
            "Action": [
                "cloudfront:*"
            ],
            "Resource": [
                "*"
            ]
        },
        {
            "Sid": "sqs",
            "Effect": "Allow",
            "Action": [
                "sqs:*"
            ],
            "Resource": [
                "*"
            ]
        },
        {
            "Sid": "vpc",
            "Effect": "Allow",
            "Action": [
                "ec2:DescribeSubnets",
                "ec2:DescribeSecurityGroups",
                "ec2:DescribeVpcs"
            ],
            "Resource": [
                "*"
            ]
        }
    ]
}
```

Once the policy is created, create an IAM role and attach the policy to it. Keep the ARN of that role in mind as you'll use it while creating a new stage.


## Management Policy

This policy is going to be attached to the IAM user that Hover uses to manage a specific stage. You may use different AWS users, or event different accounts, for every stage.

If you want, you can attach the `AdministratorAccess` policy to the user. This will give Hover access to do anything in your AWS account. However, if you want granular control, this policy ensures Hover gets just enough permissions to do its job:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "delegation",
            "Effect": "Allow",
            "Action": [
                "iam:PassRole"
            ],
            "Resource": [
                "<CloudFormation_EXECUTION_ROLE_ARN>"
            ]
        },
        {
            "Sid": "cloudformation",
            "Effect": "Allow",
            "Action": [
                "cloudformation:*"
            ],
            "Resource": [
                "*"
            ]
        },
        {
            "Sid": "ecr",
            "Effect": "Allow",
            "Action": [
                "ecr:CreateRepository",
                "ecr:GetAuthorizationToken",
                "ecr:DescribeRepositories"
            ],
            "Resource": "*"
        },
        {
            "Sid": "ecrPush",
            "Effect": "Allow",
            "Action": [
                "ecr:*"
            ],
            "Resource": "*"
        },
        {
            "Sid": "secrets",
            "Effect": "Allow",
            "Action": [
                "ssm:DescribeParameters",
                "ssm:PutParameter",
                "ssm:DeleteParameter"
            ],
            "Resource": [
                "*"
            ]
        },
        {
            "Sid": "lambda",
            "Effect": "Allow",
            "Action": [
                "lambda:*"
            ],
            "Resource": [
                "*"
            ]
        },
        {
            "Sid": "s3",
            "Effect": "Allow",
            "Action": [
                "s3:*"
            ],
            "Resource": [
                "*"
            ]
        },
        {
            "Sid": "ApiGateway",
            "Effect": "Allow",
            "Action": [
                "apigateway:GET",
                "apigateway:POST"
            ],
            "Resource": [
                "*"
            ]
        },
        {
            "Sid": "kms",
            "Effect": "Allow",
            "Action": [
                "kms:CreateKey",
                "kms:CreateAlias",
                "kms:Encrypt",
                "kms:Decrypt",
                "kms:DescribeKey"
            ],
            "Resource": [
                "*"
            ]
        }
    ]
}
```

Make sure to replace `<CloudFormation_EXECUTION_ROLE_ARN>` with the ARN of the CloudFormation Execution Role you created earlier.

## Lambda Execution Role & Policy

This role will be assumed by the different Lambda functions Hover creates. It will govern what the code running on this Lambda is allowed to do to your AWS account.

As a quick start, you may use the following permissions:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": [
                "ec2:CreateNetworkInterface",
                "ec2:DeleteNetworkInterface",
                "ec2:DescribeNetworkInterfaces",
                "logs:CreateLogGroup",
                "logs:CreateLogStream",
                "logs:PutLogEvents",
                "lambda:invokeFunction",
                "s3:*",
                "sqs:*",
                "dynamodb:*",
                "kms:DescribeKey",
                "kms:Decrypt"
            ],
            "Effect": "Allow",
            "Resource": "*"
        }
    ]
}
```

Once the policy is created, create an IAM role and attach the policy to it. Keep the ARN of that role in mind as you'll use it while creating a new stage.
