# How it works

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

1) A Pod is creating on the Kubernetes cluster.
2) The control plane manages and triggers admission webhooks.
3) The control plane sends a mutating admission request to Piggy Webhooks.
4) Piggy Webhooks processes the request and returns the mutated Pod to the control plane.

    - Inserts an init-container to install piggy-env into the containers.
    - Adds necessary environment variables to the containers.
    - Adds necessary annotations to the Pod.
    - Modifies the container's command arguments to start with piggy-env. (Reads the image manifest from the registry if required.)

5) The control plane validates the object.
6) The control plane persists the mutated object.

## Installation prerequisites

  - Ensure that the Kubernetes cluster is at least as new as v1.16.
  - Ensure that MutatingAdmissionWebhook and ValidatingAdmissionWebhook admission controllers are enabled
  - Ensure that the admissionregistration.k8s.io/v1 API is enabled.

## Proxy mode

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

1) A container starts running after the object has been finalized.
2) The piggy-env starts sending a request to Piggy Webhooks with the following content:

    - Pod's service account token
    - Pod's name
    - Piggy UID
    - Container's command signature (SHA256)

3) Piggy Webhooks receives the request, validates the resource object, and then sends requests to the Kubernetes API.
4) Piggy Webhooks performs the following validations:

    - Uses the Kubernetes Token Review API to validate the service account token.
    - Uses the namespace from the token review and the pod name to read the Pod manifest.
    - Validates the Piggy UID.
    - Validates the command signature (optional; can be turned off via configuration).

5) Piggy Webhooks sends a request to AWS STS to exchange it for a temporary access token.
6) AWS validates the request and returns a temporary access token.
7) Piggy Webhooks uses the temporary access token to request secret key-values from the `piggysec.com/aws-secret-name` and `piggysec.com/aws-region` annotations.
8) AWS Secrets Manager returns the secret key-values.
9) Piggy Webhooks parses the secret key-values, filters out restricted keys, and returns them to the container.

  - If `PIGGY_ALLOWED_SA` is found in the keys, Piggy Webhooks checks the requested service account. It returns empty if the name does not match.

10) The piggy-env receives the secret key-values and replaces environment variable values if the variable name is prefixed with `piggy:`.

### proxy mode prerequisites

  - Ensure Piggy Webhooks has permission to read secrets from AWS Secrets Manager. AWS [IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) is required.

## Standalone mode

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

1) A container starts running after the object has been finalized.
2) piggy-env sends a request to AWS STS to exchange it for a temporary access token.
3) AWS validates the request and returns a temporary access token.
4) piggy-env uses the temporary access token to request secret key-values based on the `PIGGY_AWS_SECRET_NAME` and `PIGGY_AWS_REGION` environment variables. These variables are injected by Piggy Webhooks during object mutation.
5) AWS Secrets Manager returns the secret key-values.
6) The piggy-env receives the secret key-values and replaces environment variable values if the variable name is prefixed with `piggy:`.

### Standalone mode prerequisites

  - Ensure that the application Pods have permission to read secrets from AWS Secrets Manager. AWS [IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) is required.

## Troubleshooting

You can check the logs of the Piggy Webhooks by running the following command:

```bash
kubectl -n piggy-webhooks logs deployment/piggy-webhooks
```

The log will look like this:
```
{"level":"info","namespace":"my-ns","owner":"my-pod","time":"2026-01-27T11:27:47Z","message":"Pod of ReplicaSet 'my-pod' has been mutated (took 30.701µs)"}
{"level":"info","namespace":"my-ns","pod_name":"my-pod","service_account":"my-ns:my-sa","secret_name":"my-ns/my-sa","time":"2026-01-27T12:30:02Z","message":"Request from [sa=my-ns:my-sa], [pod=my-pod] was successful"}
```

The first line shows that Piggy Webhooks has mutated a Pod. The second line shows that Piggy Webhooks has received a request from a Pod and the request was successful.

You can check the logs of the piggy-env by running the following command:

```bash
kubectl -n <namespace> logs <pod-name>
```

The log will look like this:
```
{"level":"info","time":"2026-01-27T12:37:19Z","message":"Request secrets was successful"}
```
This log shows that piggy-env has requested the secrets from Piggy Webhooks and the request was successful.

### Debug mode

You can enable debug mode by setting the `PIGGY_DEBUG` environment variable in piggy-webhooks to `true`. You may set this variable by specifying `debug: true` in `values.yaml` during Helm install.

For piggy-env, you can enable debug mode by setting `piggysec.com/debug` annotation to `true` in your pod spec.

### Automatic retry and initial delay

If Piggy Webhooks fails to retrieve secrets from AWS Secrets Manager, it will retry up to `piggysec.com/piggy-number-of-retry` times with a 500ms interval. This is useful when using a service mesh like Istio where the proxy might not be ready to allow outgoing requests yet. You can also set `piggysec.com/piggy-initial-delay` to set an initial delay before starting to retrieve secrets.
