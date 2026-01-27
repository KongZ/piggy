# Pod annotations

You can add annotations to Kubernetes Pod objects to customize Piggy's behavior.

## Annotations

| Name                                                                                       | Type    | Default     | Location | Notes |
| ------------------------------------------------------------------------------------------ | ------- | ----------- | -------- | ----- |
| [piggysec.com/aws-secret-name](#aws-secret-name)                                           | string  |             | Pods     |       |
| [piggysec.com/aws-region](#aws-region)                                                     | string  |             | Pods     |       |
| [piggysec.com/aws-secret-version](#aws-secret-version)                                     | string  | AWS_CURRENT | Pods     |       |
| [piggysec.com/piggy-env-image](#piggy-env-image)                                           | string  |             | Pods     |       |
| [piggysec.com/piggy-env-image-pull-policy](#piggy-env-image-pull-policy)                   | string  |             | Pods     |       |
| [piggysec.com/piggy-env-resource-cpu-request](#piggy-env-resource-cpu-request)             | string  |             | Pods     |       |
| [piggysec.com/piggy-env-resource-memory-request](#piggy-env-resource-memory-request)       | string  |             | Pods     |       |
| [piggysec.com/piggy-env-resource-cpu-limit](#piggy-env-resource-cpu-limit)                 | string  |             | Pods     |       |
| [piggysec.com/piggy-env-resource-memory-limit](#piggy-env-resource-memory-limit)           | string  |             | Pods     |       |
| [piggysec.com/piggy-psp-allow-privilege-escalation](#piggy-psp-allow-privilege-escalation) | boolean | false       | Pods     |       |
| [piggysec.com/piggy-address](#piggy-address)                                               | string  |             | Pods     |       |
| [piggysec.com/piggy-skip-verify-tls](#piggy-skip-verify-tls)                               | boolean | true        | Pods     |       |
| [piggysec.com/piggy-ignore-no-env](#piggy-ignore-no-env)                                   | boolean | false       | Pods     |       |
| [piggysec.com/piggy-enforce-integrity](#piggy-enforce-integrity)                           | boolean | true        | Pods     |       |
| [piggysec.com/debug](#debug)                                                               | boolean | false       | Pods     |       |
| [piggysec.com/standalone](#standalone)                                                     | boolean | false       | Pods     |       |
| [piggysec.com/image-pull-secret](#image-pull-secret)                                       | string  |             | Pods     |       |
| [piggysec.com/image-pull-secret-namespace](#image-pull-secret-namespace)                   | string  |             | Pods     |       |
| [piggysec.com/image-skip-verify-registry](#image-skip-verify-registry)                     | string  |             | Pods     |       |
| [piggysec.com/piggy-enforce-service-account](#piggy-enforce-service-account)               | bool    | false       | Pods     |       |
| [piggysec.com/piggy-default-secret-name-prefix](#piggy-default-secret-name-prefix)         | string  |             | Pods     |       |
| [piggysec.com/piggy-default-secret-name-suffix](#piggy-default-secret-name-suffix)         | string  |             | Pods     |       |
| [piggysec.com/piggy-dns-resolver](#piggy-dns-resolver)                                     | string  |             | Pods     |       |
| [piggysec.com/piggy-initial-delay](#piggy-initial-delay)                                   | string  |             | Pods     |       |
| [piggysec.com/piggy-number-of-retry](#piggy-number-of-retry)                               | int     | 0           | Pods     |       |

## AWS Secret Manager

  - <a name="aws-secret-name">`piggysec.com/aws-secret-name`</a> specifies an AWS secret name, e.g., "/myapp/name".
  - <a name="aws-region">`piggysec.com/aws-region`</a> specifies an AWS Secrets Manager region, e.g., "ap-southeast-1".
  - <a name="aws-secret-version">`piggysec.com/aws-secret-version`</a> specifies an AWS secret version. The default value is `AWS_CURRENT`.

## piggy-env settings

  - <a name="piggy-env-image">`piggysec.com/piggy-env-image`</a> overrides the piggy-env image location. If no value is specified, the piggy-env image location will be taken from the Piggy Webhooks settings in the Helm chart.
  - <a name="piggy-env-image-pull-policy">`piggysec.com/piggy-env-image-pull-policy`</a> overrides the piggy-env image pull policy. If no value is specified, the piggy-env image pull policy will be taken from the Piggy Webhooks settings in the Helm chart.
  - <a name="piggy-env-resource-cpu-request">`piggysec.com/piggy-env-resource-cpu-request`</a> overrides the piggy-env init-container resource CPU requests. Defaults to `50m`.
  - <a name="piggy-env-resource-memory-request">`piggysec.com/piggy-env-resource-memory-request`</a> overrides the piggy-env init-container resource memory requests. Defaults to `64Mi`.
  - <a name="piggy-env-resource-cpu-limit">`piggysec.com/piggy-env-resource-cpu-limit`</a> overrides the piggy-env init-container resource CPU limit. Defaults to `200m`.
  - <a name="piggy-env-resource-memory-limit">`piggysec.com/piggy-env-resource-memory-limit`</a> overrides the piggy-env init-container resource memory limit. Defaults to `64Mi`.
  - <a name="piggy-psp-allow-privilege-escalation">`piggysec.com/piggy-psp-allow-privilege-escalation`</a> allow a piggy-env init-container   to run as root. Default to `false`
  - <a name="piggy-address">`piggysec.com/piggy-address`</a> an endpoint of piggy-webhooks. This is required when it is running in proxy   mode.
  - <a name="piggy-skip-verify-tls">`piggysec.com/piggy-skip-verify-tls`</a> Do not verify TLS certificate between application and piggy-webhooks.
  - <a name="piggy-ignore-no-env">`piggysec.com/piggy-ignore-no-env`</a> does not terminate the container if no variables are found in Secrets Manager. Defaults to `false`. Setting this value to `false` (the default) is recommended for most applications; the container will not start if required environment variables are missing.
  - <a name="piggy-enforce-integrity">`piggysec.com/piggy-enforce-integrity`</a> enforces checking command integrity before injecting secrets. Defaults to `true`. Setting this value to `true` is recommended for most applications. Setting it to `false` will allow piggy-env to run with different arguments.
  - <a name="debug">`piggysec.com/debug`</a> allows to run piggy-env in debug mode. Default to `false`.
  - <a name="standalone">`piggysec.com/standalone`</a> allows to run piggy-env in standalone mode. Default to `false`. If this value is `true`, the [piggysec.com/piggy-address](#piggy-address) will not be used.
  - <a name="piggy-enforce-service-account">`piggysec.com/piggy-enforce-service-account`</a> Force to check `PIGGY_ALLOWED_SA` env value in AWS secret manager
  - <a name="piggy-default-secret-name-prefix">`piggysec.com/piggy-default-secret-name-prefix`</a>Set default prefix string for secret name
  - <a name="piggy-default-secret-name-suffix">`piggysec.com/piggy-default-secret-name-suffix`</a>Set default suffix string for secret name
  - <a name="piggy-dns-resolver">`piggysec.com/piggy-dns-resolver`</a>Set Golang DNS resolver such as `tcp`, `udp`. See [https://pkg.go.dev/net](https://pkg.go.dev/net)
  - <a name="piggy-initial-delay">`piggysec.com/piggy-initial-delay`</a> sets a delay in n[ns|us|ms|s|m|h] before starting to retrieve secrets. If you are using Istio/Envoy, you may need to set this value to `2s`. Envoy will block all outgoing requests from piggy-env until it is fully started. This delay allows Envoy to become operational before Piggy runs.
  - <a name="piggy-number-of-retry">`piggysec.com/piggy-number-of-retry`</a> sets the number of retries for retrieving secrets before giving up. Each retry will wait for 500 milliseconds. You can use this to resolve issues with delayed pod initialization, such as with Istio/Envoy.

## Container image settings

  - <a name="image-pull-secret">`piggysec.com/image-pull-secret`</a> specifies the name of the container image pull secret. Piggy will try to read the container image configuration by using secrets in the following order:
    1) `pod.spec.imagePullSecrets`
    2) `piggysec.com/image-pull-secret` annotation
    3) ServiceAccount permissions from the cloud provider
  - <a name="image-pull-secret-namespace">`piggysec.com/image-pull-secret-namespace`</a> specifies the namespace of the container image pull secret.
  - <a name="image-skip-verify-registry">`piggysec.com/image-skip-verify-registry`</a> skips registry verification when trying to read the image. Defaults to `true`.
