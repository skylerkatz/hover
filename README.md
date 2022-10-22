<h1 align="center"><img src="/docs/images/splash.png" alt="Hover" width="600"></h1>

Hover is a CLI for deploying web applications to AWS Lambda. It containerizes and deploys your app using Docker, CloudFormation, and the AWS SDK.

**Laravel PHP** is currently supported by Hover. Contributions to help support more web frameworks (PHP or otherwise) are greatly appreciated.

## Features

- Multiple stages (dev, sandbox, production, ...)
- Docker-based. Full control over the runtime environment.
- Tests run on the same docker image that gets deployed.
- Multiple queue lambdas for better prioritization of jobs.
- Environment variables are packaged with the code. Forget the 4 KB Lambda environment variables limit.
- Environment secrets are securely encrypted and packaged into the image.
- Runs on your local/CI machines. From the machine to AWS APIs directly.


## Installation

### Homebrew

```shell
brew tap themsaid/tools
brew install hover
```

### Manual download

On the [releases page](https://github.com/themsaid/hover/releases), open the latest release and download the binary that matches your OS and architecture.

## Usage

Let's create a "dev" stage for an application called "Clouder":

```shell
hover stage new clouder dev
```

This command will create two files in the root directory of our project: '/.hover/dev.yml' and '/.hover/.Dockerfile'. Using these files, we can configure how Hover builds and deploys the stage.

Next, we will build our stage:

```shell
hover build dev
```

This command will add the runtime files required for the app to run on Lambda. It will also generate the Docker images specified in the `.Dockerfile` file. More information on the build process is available [in this guide](docs/the-build-process.md).

Now that the build is complete, let's deploy:

```shell
hover deploy
```

This command will upload our asset files to S3 and our Docker image to ECR. It will also deploy a CloudFormation stack that will configure the various AWS resources that will be used to serve our application. More information on the deployment process is available [in this guide](docs/the-deployment-process.md).

## Documentation

- [Getting Started](docs/getting-started.md)
- [Architecture Concept](docs/concept.md)
- [IAM Execution Policies](docs/iam-execution-policies.md)
- [The Build Process](docs/the-build-process.md)
- [The Deployment Process](docs/the-deployment-process.md)
- [The Runtime Environment](docs/runtime-environment.md)
- [Stage Variables & Secrets](docs/stage-variables-secrets.md)
- [Working With Queues](docs/working-with-queues.md)
- [Working With Domains](docs/working-with-domains.md)
- [Manifest Reference](docs/manifest-file-reference.md)

## Fully Managed Serverless Laravel

Looking for a hosted serverless deployment platform for Laravel? Check [Laravel Vapor](https://vapor.laravel.com/). It provides both a GUI & CLI for managing all AWS resources needed to run a Laravel app on AWS Lambda. It handles databases, Redis cache, SSL certificates, S3 storage, DynamoDB tables and more.

## Contributing

Hover ships as a binary that uses Docker to build the application, allowing it to support any web framework written in any programming language. For more information on how Hover works, see the [architecture concept](/docs/concept.md) and [runtime environment](/docs/runtime-environment.md).
