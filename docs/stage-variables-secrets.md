# Working with Stage Variables

Hover has three sets of environment variables.

1. Hover managed variables: These are needed by the runtime and are automatically injected.
2. Stage variables: These are configuration environment variables set inside the `environment` attribute of the manifest file.
3. Stage secrets: These are environment variables that contain secret keys that shouldn't be stored inside the manifest file and pushed to git.

## Stage Variables vs. Secrets

To make it easy for your team to configure the stage, put all your non-secret environment variables inside the manifest file and push it with your application files to git. Now any team member can change these values and submit a pull request.

For variables that hold sensitive information, like secret keys or tokens, it's more secure to manage these inside the `<stage_name>-secrets.plain.env` file.

This file is created automatically when you add a new stage using the `hover stage new` command. It uses the Dotenv syntax to store simple key-value pairs:

```env
APP_KEY=base64:ZR+casdas*****zxczxczxcsdcsdcsd=

PRIVATE_KEY="-----BEGIN PRIVATE KEY-----
MIIEvAIBAD****1jnkeRvBNAdc2X0xB7yAZAsyQYoo6VBPHd/eT8kganZW4
VZ0XnKhPq5svtNNR0OChgq6PtchGeoQPn7iDJjDgB4fSCr1S3gQjM2AD4eoThq6m
wBcymjnyDJvpes5c9VP+8A==
-----END PRIVATE KEY-----"

DB_PASSWORD=ASD&G*A^SGssD*A
```

Once you are done editing secrets, run `hover secret encrypt --stage=<stage_name>`. Hover will generate an encryption key unique to the stage and use it to encrypt the secrets into a `<stage_name>-secrets.env` file. This file can safely be committed to git with the rest of your application.

The encryption key is then encrypted by a KMS key and appended to the `<stage_name>-secrets.env` file.

![Encrypting Secrets](images/secret-encrypt.png)

After a successful run of the `hover secret encrypt` command, the plain secrets file will be completely deleted.

```shell
hover secret encrypt --stage=<stage_name>
```

Now when a Lambda container boots up, the runtime will decrypt the secrets file and populate the `$_ENV` and `$_SERVER` global variables with the values read from that file along with any environment variables defined in the manifest file.

To edit secrets, you may run the `secret decrypt` command.

```shell
hover secret decrypt --stage=<stage_name>
```

This command will extract the encrypted key, decrypt it using KMS and use it to decrypt the secrets into a `<stage_name>-secrets.plain.env` file. It will also add the plain secrets file to `.gitignore` so you don't accidentally commit it with your application files.

![Decrypting Secrets](images/secret-decrypt.png)

## Permissions to Encrypt & Decrypt Secrets

The KMS key created by Hover is given an alias with the following naming convention:

```
[app_name]-[stage_name]-secrets-key
```

To control who can encrypt and decrypt the secrets of each environment, you may utilize AWS IAM resource permissions.

```json
{
   "Sid": "KMS",
   "Effect": "Allow",
   "Action": [
       "kms:Decrypt",
       "kms:Encrypt",
       "kms:CreateAlias",
       "kms:DescribeKey"
   ],
   "Resource": [
       "arn:aws:kms:<region>:<account>:alias/clouder-staging-secrets-key"
   ]
}
```

Including this in a policy, allows the user assuming the policy to encrypt & decrypt secrets that belong to the `staging` stage of the `clouder` application.

If a user without KMS permissions tries to run `hover stage decrypt`, an error will be displayed and the secrets will not get exposed.

The `kms:CreateAlias` permission is needed only when you first run `hover stage encrypt` on a freshly created stage. You can exclude this one for users who only need to encrypt and decrypt secrets of an existing stage.
