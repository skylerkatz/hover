# Manifest File Reference

```yaml
name: project-stage
```

This is the stage name that was assigned when you first run `hover stage new`. This name is used to prefix, and sometimes suffix, the names of the different AWS resources created by the stage. This should never change.

```yaml
aws-profile: staging-account
```

This is the name of the AWS credentials profile Hover will use to interact with AWS. Typically, it is only used when using Hover on local machines. In a CI environment, AWS environment variables are used instead and Hover ignores the `aws-profile` provided.

```yaml
region: eu-west-1
```

This is the name of the AWS region where the stage is deployed.

```yaml
dockerfile: .dockerfile
```

This is the name of the dockerfile inside the `.hover` directory that the stage uses. You can use the same docker file to build multiple stages, or have a special docker file for each stage.

```yaml
environment:
    APP_ENV: staging
    APP_DEBUG: true
    APP_LOG_LEVEL: debug
```

These are the [configuration variables](/stage-variables-secrets.md#stage-variables-vs-secrets) of the stage.

```yaml
auth:
    stack-role: arn:aws:iam::<account>:role/DefaultStackExecution
    lambda-role: arn:aws:iam::<account>:role/DefaultLambdaExecutionRole
```

These are the ARNs of the stack execution role and the Lambda execution roles assumed by CloudFormation and Lambda respectively. These are set when you first run `hover stage new`.

```yaml
vpc:
    security-groups:
        - sg-*****
    subnets:
        - subnet-*****
        - subnet-*****
```

These are the names of the security groups and subnets of the VPC should you choose to run your functions inside a VPC.

```yaml
deploy-commands:
  - 'php artisan migrate --force'
```

These are the commands that Hover runs after publishing new versions of your stage functions. If any of these commands fail, the new version will not be used and the deployment will fail.

```yaml
http:
    memory: 512
    timeout: 30
    warm: 10
    concurrency: 100
```

These are the configurations of the HTTP function.

- `memory` and `timeout` controls the maximum memory and maximum timeout the Lambda allocates.
- `concurrency` controls the maximum concurrency slots reserved by the function.
- `warm` controls the minimum number of containers to keep warm.

```yaml
cli:
    memory: 512
    timeout: 30
    concurrency: 100
```

These are the configurations of the CLI function.

```yaml
queue:
  default:
    memory: 512
    timeout: 120
    concurrency: 5
    tries: 3
    backoff: "5, 10"
    queues:
      - default
      - notifications
```

These are the different queue functions that you want to create for the stage. Each function may process jobs from one or more queues.

The `tries` and `backoff` attributes configure the default number of tries and default backoff settings for jobs that don't have this defined internally.
