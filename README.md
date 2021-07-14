# piggy

Piggy is a tool built for supporting AWS Secret Manager with Kubernetes. It has abilities to mutating Pods, unseal secrets and inject
into application environment.

## Installation

Current release requires AWS IRSA to provide the IAM permission to piggy for unsealing secrets. Before installing Piggy webhooks, you must
setup the IRSA on AWS. Sees https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html for complete detail of IRSA setting up. 
Alternatively, you can use Terraform the setup IRSA. Sees https://github.com/terraform-aws-modules/terraform-aws-eks/tree/master/examples/irsa

The simplest IRSA Policy for Piggy webhooks

```
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

```
helm repo add kongz https://charts.kong-z.com
helm -n piggy-webhooks install piggy-webhooks kongz/piggy-webhooks --set aws.roleArn=${piggy-role-arn}
```

2) Add these minimum annotations to your Deployment

```
apiVersion: v1
kind: Pod
metadata:
  annotations:
    piggy.kong-z.com/piggy-address: https://piggy-webhooks.piggy-webhooks.svc.cluster.local
    piggy.kong-z.com/aws-secret-name: ${your-aws-secret-name} ## e.g. myapp/sample/production
    piggy.kong-z.com/aws-region: ${your-aws-secret-region} ## e.g. ap-southeast-1
```

3) Add Env value with format `piggy:${name}`

```
  containers:
      env:
        - name: TEST_ENV
          value: piggy:TEST_ENV
```

4) That all!!. See the demo at 

## How does it work

When application is deployed on Kubernetes, the Kubernetes API will send admission request to Piggy webhooks. The Piggy webhooks will mutate the
pods and injecting secrets to containers


## Lookup mode

## Standalone mode

## License
Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.