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
