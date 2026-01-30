# Optimizing Webhook Invocations

Piggy Webhooks uses a Mutating Admission Webhook to inject secrets into Pods. By default, Kubernetes sends an AdmissionReview request to the webhook for *every* Pod creation in namespaces that aren't excluded. In large clusters, this can lead to:

1.  **Increased Latency**: Every Pod creation must wait for the webhook response.
2.  **Resource Pressure**: The webhook service must handle a high volume of no-op requests.

Piggy provides two mechanisms to optimize this: `objectSelector` and `matchConditions`.

## 1. Object Selector (Recommended for all versions)

The `objectSelector` allows you to filter pods based on their labels. If you label your pods that require secret injection, you can prevent the API server from calling the webhook for unlabeled pods.

### Configuration

In `values.yaml`:

```yaml
mutate:
  objectSelector:
    matchLabels:
      piggysec.com/mutate: "true"
```

### Usage

Add the label to your Pod spec:

```yaml
metadata:
  labels:
    piggysec.com/mutate: "true"
```

---

## 2. Match Conditions (Recommended for K8s 1.27+)

`matchConditions` use CEL (Common Expression Language) to filter requests based on the object's properties (like annotations) *before* the webhook is invoked. This is the most efficient way to filter for Piggy, as injection is usually triggered by annotations.

### Configuration

In `values.yaml`:

```yaml
mutate:
  matchConditions:
    - name: 'include-piggy-pods'
      expression: |
        has(object.metadata.annotations) && (
          'piggysec.com/piggy-address' in object.metadata.annotations ||
          'piggysec.com/aws-secret-name' in object.metadata.annotations ||
          'piggysec.com/aws-ssm-parameter-path' in object.metadata.annotations
        )
```

## Comparison

| Feature | Filter Level | Requirement | Best For |
| ------- | ------------ | ----------- | -------- |
| `excludeNamespaces` | Namespace | K8s 1.9+ | Coarse filtering by team/env |
| `objectSelector` | Labels | K8s 1.15+ | Explicit opt-in via labels |
| `matchConditions` | Any field (CEL) | K8s 1.27+ | Implicit filtering by annotations |
