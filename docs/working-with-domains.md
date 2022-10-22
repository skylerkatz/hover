# Working with Custom Domains

For each HTTP API created by Hover, AWS generates a unique domain, which can be used to test the application. It looks like this:

```
https://d3876dg38.execute-api.eu-west-1.amazonaws.com
```

To access the app using your own domain, Hover utilizes APIGateway custom domain mappings. To get started, you need to create a hosted zone in your [Route53 AWS console](https://us-east-1.console.aws.amazon.com/route53) and issue a certificate for the domain from [AWS certificate manager](https://console.aws.amazon.com/acm).

Once the zone and certificate are created, you may run the following command to
create a custom domain name in APIGateway:

```shell
hover domain create <domain> <certificate_arn> --stage=<stage_name>
```

The output of the command will include the CNAME record you need to add to your DNS provider to point the domain to APIGateway.

Now that the domain is created, you need to update the stage manifest file to instruct Hover to map the domain to the stage:

```yaml

domains:
    - mydomain.com
```

For the changes to take effect, you need to build and deploy the stage:

```shell
hover build <stage_name>
hover deploy
```
