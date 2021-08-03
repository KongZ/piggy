All source code for demo application and kubernetes manifests files can be found at [demo](https://github.com/KongZ/piggy/tree/main/demo)

## Create secret in AWS Secret Manager
The demo here will retrieve the secret name `TEST_ENV` from AWS secret manager name `demo/sample/test`

![secret-manager](https://raw.githubusercontent.com/KongZ/piggy/main/docs/images/secret-manager.png "secret-manager")

### Create IRSA
In this demo, I use terraform to create IRSA for `piggy-webhooks` namespace and service account.

The simplest IRSA Policy for Piggy webhooks

```yaml
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PiggySecretReadOnly",
      "Action": [
        "secretsmanager:DescribeSecret",
        "secretsmanager:GetResourcePolicy",
        "secretsmanager:GetSecretValue",
        "secretsmanager:ListSecretVersionIds",
        "secretsmanager:ListSecrets"
      ],
      "Effect": "Allow",
      "Resource": "*"
    },
    {
      "Sid": "PiggyECRReadOnly",
      "Action": [
        "ecr:BatchCheckLayerAvailability",
        "ecr:BatchGetImage",
        "ecr:DescribeImages",
        "ecr:GetAuthorizationToken",
        "ecr:GetDownloadUrlForLayer"
      ],
      "Effect": "Allow",
      "Resource": "*"
    }
  ]
}
```

![terraform-irsa](https://raw.githubusercontent.com/KongZ/piggy/main/docs/images/terraform-irsa.png "terraform-irsa")

### Run helm chart install

`${piggy-role-arn}` can be found from Role ARN which was created by terraform above

```bash
helm repo add piggysec https://charts.piggysec.com
helm install piggy-webhooks piggysec/piggy-webhooks --set aws.roleArn=${piggy-role-arn}
```

Check running piggy-webhooks pod

![get-po-piggy](https://raw.githubusercontent.com/KongZ/piggy/main/docs/images/get-po-piggy.png "get-po-piggy")

### Create demo pod
Now piggy-webhooks pod is ready, create a pod with annotations to piggy-webhooks service, secret name, and aws-region.
You can see a yaml file at [demo/lookup/pod.yaml](https://github.com/KongZ/piggy/tree/main/demo/lookup/pod.yaml)

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: demo
  annotations:
    piggysec.com/piggy-address: https://piggy-webhooks.piggy-webhooks.svc.cluster.local
    piggysec.com/aws-secret-name: demo/sample/test
    piggysec.com/aws-region: ap-southeast-1
spec:
  containers:
    - image: ghcr.io/kongz/piggy-demo:latest
      name: demo
      env:
        - name: TEST_ENV
          value: piggy:TEST_ENV
      resources:
        limits:
          memory: "64Mi"
          cpu: "200m"
```

![create-po-demo](https://raw.githubusercontent.com/KongZ/piggy/main/docs/images/create-po-demo.png "create-po-demo")

### Exec to pod and test
Use `kubectl exec -it demo -- /bin/bash` to execute into pod

![exec-po-demo](https://raw.githubusercontent.com/KongZ/piggy/main/docs/images/exec-po-demo.png "exec-po-demo")

#### Check environment value
Just simply `echo $TEST_ENV` and see the result. The environment variable value is not resolved. You won't see the value on container shell.

```bash
> echo $TEST_ENV
> piggy:TEST_ENV
```

![echo-demo-env](https://raw.githubusercontent.com/KongZ/piggy/main/docs/images/echo-demo-env.png "echo-demo-env")

Now, try to curl to demo app and let's app resolve the environment variable 

```bash
> curl -s localhost:8080 | grep TEST_ENV
> TEST_ENV=hello world
```

![curl-demo-env](https://raw.githubusercontent.com/KongZ/piggy/main/docs/images/curl-demo-env.png "curl-demo-env")
