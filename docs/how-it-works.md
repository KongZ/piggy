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

1) Pod is creating on Kubernetes cluster.
2) The control pane manages and triggers admission webhooks
3) The control pane sends a mutating admission webhook to Piggy Webhooks
4) Piggy Webhooks processes request and return mutated Pod to the control pane

    - Inserts init-controller to install piggy-env into the containers
    - Adds necessary environment variable into containers
    - Adds necessary annotations into Pods
    - Modify containers command arguments to start with piggy-env. (Read image manifest from registry if requires)

5) The control pane validates object
6) The control pane persists mutated object

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

1) A container is running after object has been finalized
2) A piggy-env start sending a request to the Piggy Webhooks with the following content

    - Pod's service account token
    - Pod's name
    - Piggy UID
    - Container's command signature (SHA256)

3) The Piggy Webhooks receives a request, validates resource object then send requests to Kubernetes API
4) The Piggy Webhooks does the following validation

    - Uses Kubernetes Token Review API to validate service account token
    - Uses a namespace from token review and pod name to read a pod manifest
    - Validates Piggy UID
    - Validates command signature as an option. This can be turned from by configuration

5) The Piggy Webhooks sends request to AWS STS for exchanging a temporary access token
6) AWS validates request and return a temporary access token
7) The Piggy Webhooks uses temporary access token requests for a secret key-value from `piggysec.com/aws-secret-name` and `piggysec.com/aws-region` annotations
8) AWS secret manage return a secret key-value
9) The Piggy Webhooks parse a secret key-value, filter-out some restricted key and return to a container.

  - If `PIGGY_ALLOWED_SA` is found on key, the Piggy Webhooks will check the container requested service account. Return empty if name is not matched

10) The piggy-env received secret key-value then replace environment variable value if variable name with prefix `piggy:`

### proxy mode prerequisites

  - Ensure that the Piggy Webhooks has a permission to read secrets from AWS Secret Manager. The AWS [IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) is required

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

1) A container is running after object has been finalized
2) The Piggy Webhooks sends request to AWS STS for exchanging a temporary access token
3) AWS validates request and return a temporary access token
4) piggy-env uses temporary access token requests for a secret key-value from `PIGGY_AWS_SECRET_NAME` and `PIGGY_AWS_REGION` environment variables. These variable are injected by the Piggy Webhooks during object mutation
5) AWS secret manage return a secret key-value
6) The piggy-env received secret key-value then replace environment variable value if variable name with prefix `piggy:`

### Standalone mode prerequisites

  - Ensure that the application Pods has a permission to read secrets from AWS Secret Manager. The AWS [IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) is required
