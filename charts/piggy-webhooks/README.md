# Piggy Webhooks Helm Chart

This chart installs the Piggy Webhooks service, which provides admission control for pod mutation and secret injection.

## Prerequisites

- Kubernetes 1.16+
- Helm 3.0+
- AWS IRSA configured (for AWS Secrets Manager access)

## Installation

```bash
helm repo add piggysec https://piggysec.com
helm install piggy-webhooks piggysec/piggy-webhooks \
  --namespace piggy-webhooks \
  --create-namespace \
  --set aws.roleArn=arn:aws:iam::123456789012:role/piggy-webhooks-role
```

## Configuration

The following table lists the most commonly used configurable parameters of the Piggy Webhooks chart and their default values.

| Parameter                  | Description                         | Default                        |
| -------------------------- | ----------------------------------- | ------------------------------ |
| `replicaCount`             | Number of replicas                  | `1`                            |
| `image.repository`         | Image repository                    | `ghcr.io/kongz/piggy-webhooks` |
| `image.tag`                | Image tag                           | `0.7.0`                        |
| `aws.roleArn`              | AWS IAM Role ARN for IRSA           | `""`                           |
| `mutate.excludeNamespaces` | Namespaces to exclude from mutation | `[]`                           |
| `debug`                    | Enable debug logging                | `false`                        |

## Webhook Optimization

By default, the webhook matches all pods in non-excluded namespaces. To improve performance, you can use `objectSelector` or `matchConditions`.

### Using Match Conditions (Recommended for K8s 1.27+)

This pre-filters pods at the API server level, reducing load on the webhook.

```yaml
mutate:
  matchConditions:
    - name: "include-piggy-pods"
      expression: |
        has(object.metadata.annotations) && (
          'piggysec.com/piggy-address' in object.metadata.annotations ||
          'piggysec.com/aws-secret-name' in object.metadata.annotations ||
          'piggysec.com/aws-ssm-parameter-path' in object.metadata.annotations
        )
```
