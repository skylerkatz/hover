# Working with Custom Domains

For each stage created by Hover, a CloudFront distribution is created and assigned a unique domain, which can be used to test the application. It looks like this:

```
https://d1ascr3e2rsbz3.cloudfront.net
```

To access the app using your own domain, Hover utilizes CloudFront aliases. To get started, you need to issue a certificate for the domain from [AWS certificate manager](https://console.aws.amazon.com/acm). This certificate *must* be issued in `us-east-1`.

Once the certificate is issued, you need to update the stage manifest file to instruct Hover to map the domain to the stage by providing two attributes under the `http` key: `domains` and `certificate`.

```yaml

http:
  memory: 256
  // ...
  domains: domain.com
  certificate: arn:aws:acm:us-east-1:<account>:certificate/<id>
```

For the changes to take effect, you need to build and deploy the stage:

```shell
hover build <stage_name>
hover deploy
```

To use multiple domains, separate between them using a comma:

```yaml
domains: domain.com, *.domain.com
```
