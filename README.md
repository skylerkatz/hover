<h1 align="center"><img src="/docs/images/splash.png" alt="Hover" width="600"></h1>

Hover is a CLI for deploying web applications to AWS Lambda. It containerizes and deploys your app using Docker, CloudFormation, and the AWS SDK.

**Laravel PHP** is currently supported by Hover. Contributions to help support more web frameworks (PHP or otherwise) are greatly appreciated.

## Features

- Multiple stages (dev, sandbox, production, ...)
- Manages each stage separately. You can deploy each stage in a different AWS account.
- Docker-based. Full control over the runtime environment.
- Tests run on the same docker image that gets deployed.
- Can configure multiple queue lambdas for better prioritization management.
- Environment variables are packaged with the code. Forget the 4 KB Lambda environment variables limit.
- Environment secrets are securely encrypted and packaged into the image.
- Deploys from your local/CI machines. From the machine to AWS APIs directly.
- The application and its asset files are served from the same domain.

## Motivation

[Taylor Otwell](https://twitter.com/taylorotwell) introduced serverless to Laravel in 2019 with the release of [Laravel Vapor](https://vapor.laravel.com/), and I was part of the team that worked on Vapor from the beginning. I also contributed to the platform's upkeep for over two years.

During this time, I've seen how serverless has assisted monolithic application developers in scaling their apps without the need for complex infrastructure management. As a result, they focused less on infrastructure and more on creating awesome apps.

However, due to strict compliance requirements, using Vapor or any other deployment platform was not an option in some cases. Teams developing apps under such constraints were not permitted to share their AWS credentials with a third party. Hover was designed specifically for these teams.

You can deploy serverless web applications directly from your CI or local machines using Hover. The used AWS credentials can be restricted to only handle the resources required to run the app.

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
