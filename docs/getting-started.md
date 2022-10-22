# Getting Started

In this step-by-step guide, we're going to create a production stage for a brand-new project and deploy it to AWS Lambda.


## Security Considerations

First thing we're going to do is create two IAM Roles to govern what the CloudFormation stack and stage functions can do. We will also create an IAM user that will be used to initiate the deployments and interact with AWS through Hover.

A guide on how to create the two roles and the user can be [found here](/docs/iam-execution-policies.md).

After this step, we'll collect the following information:

1. The ARN of the CloudFormation execution role.
2. The ARN of the Lambda execution role.
3. The `aws_access_key_id` and `aws_secret_access_key` of the IAM user.

In a CI environment, we will add those keys to the `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables. However, on the local machine we're going to create an AWS [named profile](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html) inside the `~/.aws/credentials` file:

```
[production]
aws_access_key_id = AKIA******
aws_secret_access_key = uw6*******
```

## Creating The Stage

Let's run the following command inside the root directory of our project:

```
hover stage new clouder production
```

`clouder` here is the name of our app and `production` is the name of the stage.

Hover is going to ask us for the following:

1. Name of the AWS region.
2. Name of the AWS profile.
3. The ARN of the CloudFormation execution role.
4. The ARN of the Lambda execution role.

Once the command successfully executes, the following files will be created inside a `.hover` directory:

1. `production.yml`
2. `.gitignore`
3. `.Dockerfile`
4. `production-secrets.plain.env`

Hover will show us a warning to ensure some composer dependencies are installed in the application. These are:

- hollodotme/fast-cgi-client:^3.1
- guzzlehttp/promises:^1.5
- aws/aws-sdk-php:^3.2

We will run the following command to require these dependencies:

```shell
composer require hollodotme/fast-cgi-client:^3.1 guzzlehttp/promises:^1.5 aws/aws-sdk-php:^3.2
```

These are dependencies of the Hover runtime and must be installed prior to deployment.

## Configuring Stage Variables & Secrets

Every Laravel application requires an `APP_KEY` environment variable configured. Let's generate one:

```shell
php artisan key:generate --show
```

The command will output a new encryption key. We'll copy this key and update the `APP_KEY` variable inside the `production-secrets.plain.env` file:

```
APP_KEY=base64:pbx6nnS****
```

We also want to update the `DB_PASSWORD` variable to hold the production database password.

```
DB_PASSWORD:CA&G********
```

In this file, we may add any variables that hold sensitive information which shouldn't be exposed when committing to git. Once all changes are done, let's run the following command:

```shell
hover secret encrypt --stage=production
```

This command will encrypt the secrets into a `production-secrets.env` file and delete the `production-secrets.plain.env` one.

> **Note**: For more information on secrets, check [this guide](/docs/stage-variables-secrets.md)

For non-sensitive variables, we will store them inside the `production.yml` manifest file:

```yaml
environment:
  APP_DEBUG: false
  APP_LOG_LEVEL: debug
  FILESYSTEM_DRIVER: s3
  FILESYSTEM_CLOUD: s3
  LOG_CHANNEL: stderr
  QUEUE_CONNECTION: sqs
  SCHEDULE_CACHE_DRIVER: dynamodb
  SESSION_DRIVER: cookie
```

## Configuring The Stage

Inside the `./hover/production.yml` manifest file, we may configure how Hover should deploy our application. A complete reference of the file can be [found here](/docs/manifest-file-reference.md).

One of the most important things to consider is configuring the queue component so Hover creates all the queues the application uses. Otherwise, we'll get errors because the application is trying to dispatch jobs to a queue that doesn't exist.

Let's say our app pushes to 3 queues, here's how the setup will look:

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
  priority:
    memory: 512
    timeout: 300
    concurrency: 10
    tries: 5
    backoff: "1"
    queues:
      - priority
```

We created 2 functions that process jobs from the three queues (default, notifications & priority).

> **Note**: For more information on working with queues, check [this guide](/docs/working-with-queues.md)

## Configuring The Docker Build

Hover uses Docker to containerize our application and prepare it to run on AWS Lambda. It uses the same base docker image to run the tests as well, and uses a separate build stage to compile our application asset files.

To configure the build, we may edit the `.hover/.Dockerfile` file.

For example, we can change the command that executes our tests by updating the `tests` stage of the docker file:

```diff
  FROM base as tests

- CMD vendor/bin/phpunit
+ CMD php artisan test
```

> **Note**: For more information on the Docker file, check [this guide](/docs/the-build-process.md#docker)

## Building The Stage

Now that we have everything configured, let's build the stage:

```shell
hover build production
```

During the build process, Hover will inject the runtime files and build the docker images defined in the `.Dockerfile`. It will also run the tests on the "tests" docker stage.

> **Note**: For more information on the build process, check [this guide](/docs/the-build-process.md)

## Deploying

Hover deploys our application by uploading the compiled assets files to S3 and the Docker image to ECR. It then provisions a CloudFormation stack to create all the needed AWS resources to run the stage. This includes:

1. ApiGateway HTTP API
2. Lambda Functions
3. CloudFormation Distribution
4. EventBridge Rules
5. SQS Queues

> **Note**: For the complete list of resources managed by Hover and a general overview of the architecture concept. Check [this guide](/docs/concept.md).

To start the deployment, let's run the following command:

```shell
hover deploy
```

Once the stack is provisioned, Hover will publish a new version of each of the functions, warm several HTTP function containers, run deployment commands and activate the new release.

> **Note**: For more information on the deployment process, check [this guide](/docs/the-deployment-process.md)

## Interacting With The Application

After the `deploy` command finishes, Hover will print an internal domain that looks like this:

```
https://<api_id>.execute-api.eu-west-1.amazonaws.com
```

We can visit our website using this domain to test things out, we can also configure a custom domain by following [this guide](/docs/working-with-domains.md).

To run artisan command, we may use the `command run` command:

```
hover command run "inspire" --stage=production
```

This will run the `php artisan inspire` command inside the CLI function and print the output.

> **Note**: For more information on how the runtime works, check [this guide](/docs/runtime-environment.md)
