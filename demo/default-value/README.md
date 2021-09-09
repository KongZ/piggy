This demo require default settings on Piggy Webhook. You can install Piggy Webhooks with default settings by

```bash
helm -n piggy-webhooks install piggy-webhooks piggysec/piggy-webhooks \
  --set aws.roleArn=${piggy-role-arn} \
  --set env.AWS_REGION=ap-southeast-1 \
  --set env.PIGGY_DEFAULT_SECRET_NAME_SUFFIX=/test
```

Or use a value file

```yaml
aws:
  roleArn: arn:aws:iam::123456789:role/piggy-webhooks

env:
  AWS_REGION: "ap-southeast-1"
  PIGGY_DEFAULT_SECRET_NAME_PREFIX: ""
  PIGGY_DEFAULT_SECRET_NAME_SUFFIX: "/test"
```

```bash
helm -n piggy-webhooks install piggy-webhooks piggysec/piggy-webhooks -f values.yaml
```

![default-secret-name](https://raw.githubusercontent.com/KongZ/piggy/main/docs/images/secret-name-default-value.png "default-secret-name")

