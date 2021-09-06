[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Action](https://github.com/KongZ/piggy/workflows/Build%20to%20GHCR/badge.svg?branch=main)](https://github.com/KongZ/piggy/actions)
[![CodeQL](https://github.com/KongZ/piggy/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/KongZ/piggy/actions/workflows/codeql-analysis.yml)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/piggy)](https://artifacthub.io/packages/search?repo=piggy)

![piggy logo](https://raw.githubusercontent.com/KongZ/piggy/main/docs/images/piggy.png "Piggy Logo")

# Piggy

Piggy is a tool built for supporting AWS Secret Manager with Kubernetes. It has abilities to mutating Pods, unseal secrets and inject
into application environment.

## Installation

Current release requires AWS IRSA to provide the IAM permission to piggy for unsealing secrets. Before installing Piggy webhooks, you must
setup the IRSA on AWS. Sees [https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html] for complete detail of IRSA setting up.
Alternatively, you can use Terraform the setup IRSA. Sees [https://github.com/terraform-aws-modules/terraform-aws-eks/tree/master/examples/irsa]

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

1) Run helm chart install

```bash
helm repo add piggysec https://piggysec.com
helm -n piggy-webhooks install piggy-webhooks piggysec/piggy-webhooks --set aws.roleArn=${piggy-role-arn}
```

2) Add these minimum annotations to your Deployment

```yaml
apiVersion: v1
kind: Pod
metadata:
  annotations:
    piggysec.com/piggy-address: https://piggy-webhooks.piggy-webhooks.svc.cluster.local
    piggysec.com/aws-secret-name: ${your-aws-secret-name} ## e.g. myapp/sample/production
    piggysec.com/aws-region: ${your-aws-secret-region} ## e.g. ap-southeast-1
```

You can define a default AWS region by setting `AWS_REGION` environment value in Piggy Webhooks

3) Add Env value with format `piggy:${name}`

```yaml
  containers:
      env:
        - name: TEST_ENV
          value: piggy:TEST_ENV
```

4) That all!!. See the demo at [https://github.com/KongZ/piggy/tree/main/demo]

### Default settings

Some setting such as AWS region can set a default value by setting `env` in piggy webhooks helm chart value.
Simply remove prefix `piggysec.com` from annotation, put it all in upper case, and change `-` to `_`.

For example:

```yaml
env:
  AWS_REGION: "ap-southeast-1"
  PIGGY_ENFORCE_SERVICE_ACCOUNT: "true"
```

## Proxy mode

This is a default mode. The Piggy Webhooks requires a permission to read secret from AWS Secret Manager.
The application containers will send request to Piggy Webhooks and Piggy Webhooks will inject the secret into containers environments
which prefix with `piggy:`

```bash
                (1)  ┌───────────┐ (10)
                ───▶ │           │ ───▶
              ───────│ Container │───────
                     │           │
                     └───────────┘
                           │
                         «tls»
                           │
                          ││▲
                       (2)│││(9)
                          ▼││
┌───────────┐  (3)   ┌───────────┐  (5)   ┌───────────┐
│Kubernetes │  ◀───  │   Piggy   │  ───▶  │           │
│    API    │────────│ Webhooks  │────────│  AWS STS  │
│           │  ───▶  │           │  ◀───  │           │
└───────────┘   (4)  └───────────┘   (6)  └───────────┘
                           │▲
                          │││(8)
                       (7)│││
                          ▼│
                     ┌───────────┐
                     │AWS Secret │
                     │  Manager  │
                     │           │
                     └───────────┘
```

The example manifest file for Pod. To receive the Piggy Webhooks injection, you will need only 3 annotations

  - `piggysec.com/piggy-address` - set a value to Piggy Webhooks service
  - `piggysec.com/aws-secret-name` - set a value to your AWS secret name
  - `piggysec.com/aws-region` - set a value to your AWS secret manager region

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp
  annotations:
    piggysec.com/piggy-address: https://piggy-webhooks.piggy-webhooks.svc.cluster.local
    piggysec.com/aws-secret-name: myapp/sample
    piggysec.com/aws-region: ap-southeast-1
spec:
  containers:
    - image: myapp:v1
      name: myapp
      env:
        - name: TEST_ENV
          value: piggy:TEST_ENV
```

Then you can read the `TEST_ENV` value from environment variable.

```go
func main() {
  val := os.Getenv("TEST_ENV")
  fmt.Printf("%s", val)
}
```

### Restrict process to run

Set [piggy-enforce-integrity](https://github.com/KongZ/piggy/blob/enforce-integrity/docs/annotations.md#piggy-enforce-integrity) annotation to `true` (default is true) will restrict piggy-env to resolve the variable only process defined on container arguments.

### Limit secrets injection only allowed service accounts

You may improve security by restrict only Pod service account to read the secrets.
You can limit access by adding variable name `PIGGY_ALLOWED_SA` to AWS secret where value is `namespace:service_account` name.

The Piggy Webhooks will not inject secrets into containers if the Pod service account name is not matched with value of `PIGGY_ALLOWED_SA`.

You can add multiple service account name by separate each name with comma.

For example:

```bash
myapp-namespace:myapp,myanotherapp-namespace:default
```

### Preventing unauthorized pods to read secrets

The Piggy provides 3 cencepts to protect secrets.

  - By enabling [piggy-enforce-integrity](https://github.com/KongZ/piggy/blob/enforce-integrity/docs/annotations.md#piggy-enforce-integrity). The Piggy will generate a check sum using SHA256 algorithm from a container command.
  Then piggy-env will generate another check sum on running command every time when communicate with piggy-webhooks. If the check sum is not matched with original value, it will reject the request.
  For example, if your container starts with command `rails server`, you won't be able to `exec` into pod and run `rails console` to get secrets. This option is enabled by default.
  - Piggy will generate a UID for each containers during mutation process. If the requests from container is not matched the UID which was generated, it will reject.
  - Uses [PIGGY_ALLOWED_SA](https://github.com/KongZ/piggy#limit-secrets-injection-only-allowed-service-accounts) to limit access to secrests by service account name

## Standalone mode

The standalone mode will not use Piggy Webhooks to inject secrets into containers. It will requires Pod service account with IRSA to
read the secrets from AWS Secret Manager. You can enable standalone mode by adding annotation `piggysec.com/standalone: "true"` to Pod

```bash
     (1)  ┌───────────┐ (6)
     ───▶ │           │ ───▶
   ───────│ Container │───────
          │           │
          └───────────┘
             │     │
             │     │
      ┌──────┘     └─────┐▲
     ││▲                │││(3)
  (4)│││(5)          (2)│││
     ▼││                ▼│
┌───────────┐      ┌───────────┐
│AWS Secret │      │           │
│  Manager  │      │  AWS STS  │
│           │      │           │
└───────────┘      └───────────┘
```

Since the standalone mode does not use Piggy Webhooks thus the Pod must have a permission to read secrets from AWS Secret Manager.
You need to setup AWS IRSA with at least this permission

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
      "Resource": "${your-secret-name-arn}"
    }
  ]
}
```

Then add then follow annotations to Pod. You may notice, you don't have to provide the Piggy Webhooks address in this mode.

  - `piggysec.com/aws-secret-name` - set a value to your AWS secret name
  - `piggysec.com/aws-region` - set a value to your AWS secret manager region
  - `piggysec.com/standalone` - set a value to true

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp
  annotations:
    piggysec.com/aws-secret-name: omise-staging/sample/test
    piggysec.com/aws-region: ap-southeast-1
    piggysec.com/standalone: "true"
spec:
  serviceAccount: myapp
  containers:
    - image: myapp:v1
      name: myapp
      volumeMounts:
        - mountPath: /var/run/secrets/eks.amazonaws.com/serviceaccount
          name: aws-iam-token
          readOnly: true
      env:
        - name: TEST_ENV
          value: piggy:TEST_ENV
        - name: AWS_ROLE_ARN
          value: ${your-role-arn}
        - name: AWS_WEB_IDENTITY_TOKEN_FILE
          value: /var/run/secrets/eks.amazonaws.com/serviceaccount/token
  volumes:
    - name: aws-iam-token
      projected:
        defaultMode: 420
        sources:
        - serviceAccountToken:
            audience: sts.amazonaws.com
            expirationSeconds: 86400
            path: token
```

And the service account

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: myapp
  annotations:
    eks.amazonaws.com/role-arn: ${your-role-arn}

```

Note: in standalone mode, piggy-env talks directly to AWS secret manager without talking to Kubernetes API or piggy-webhooks.
The secrets is fully protected by AWS role permissions.

## How does it work

When application is deployed on Kubernetes, the Kubernetes API will send admission request to Piggy webhooks. The Piggy webhooks will mutate the
pods and injecting secrets into containers

```bash
 (1)   ┌───────────┐  (2)   ┌───────────┐ (5)   ┌───────────┐ (6)   ┌───────────┐
 ───▶  │           │  ───▶  │ Mutating  │ ───▶  │  Object   │ ───▶  │           │
───────│Create Pod │────────│ Admission │───────│Validation │───────│ Persisted │
       │           │        │           │       │           │       │           │
       └───────────┘        └───────────┘       └───────────┘       └───────────┘
                                  │
                                «tls»
                                  │▲
                                 │││(4)
                              (3)│││
                                 ▼│
                            ┌───────────┐
                            │   Piggy   │
                            │ Webhooks  │
                            │           │
                            └───────────┘
```

See [how it works](https://github.com/KongZ/piggy/tree/main/docs/how-it-works.md)

## Annotations

See [annotations](https://github.com/KongZ/piggy/tree/main/docs/annotations.md)

## License

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at

<http://www.apache.org/licenses/LICENSE-2.0/>

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
