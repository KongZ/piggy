# Pod annotations

You can add annotations to kubernetes Pods objects to customize piggy behavior.

## Annotations

| Name                                                                                       | Type    | Default | Location | Notes |
|--------------------------------------------------------------------------------------------|---------|---------|----------|-------|
| [piggysec.com/aws-secret-name](#aws-secret-name)                                           | string  |         | Pods     |       |
| [piggysec.com/aws-region](#aws-region)                                                     | string  |         | Pods     |       |
| [piggysec.com/piggy-env-image](#piggy-env-image)                                           | string  |         | Pods     |       |
| [piggysec.com/piggy-env-image-pull-policy](#piggy-env-image-pull-policy)                   | string  |         | Pods     |       |
| [piggysec.com/piggy-env-resource-cpu-request](#piggy-env-resource-cpu-request)             | string  |         | Pods     |       |
| [piggysec.com/piggy-env-resource-memory-request](#piggy-env-resource-memory-request)       | string  |         | Pods     |       |
| [piggysec.com/piggy-env-resource-cpu-limit](#piggy-env-resource-cpu-limit)                 | string  |         | Pods     |       |
| [piggysec.com/piggy-env-resource-memory-limit](#piggy-env-resource-memory-limit)           | string  |         | Pods     |       |
| [piggysec.com/piggy-psp-allow-privilege-escalation](#piggy-psp-allow-privilege-escalation) | boolean | false   | Pods     |       |
| [piggysec.com/piggy-address](#piggy-address)                                               | string  |         | Pods     |       |
| [piggysec.com/piggy-skip-verify-tls](#piggy-skip-verify-tls)                               | boolean | true    | Pods     |       |
| [piggysec.com/piggy-ignore-no-env](#piggy-ignore-no-env)                                   | boolean | false   | Pods     |       |
| [piggysec.com/piggy-enforce-integrity](#piggy-enforce-integrity)                           | boolean | true    | Pods     |       |
| [piggysec.com/debug](#debug)                                                               | boolean | false   | Pods     |       |
| [piggysec.com/standalone](#standalone)                                                     | boolean | false   | Pods     |       |
| [piggysec.com/image-pull-secret](#image-pull-secret)                                       | string  |         | Pods     |       |
| [piggysec.com/image-pull-secret-namespace](#image-pull-secret-namespace)                   | string  |         | Pods     |       |
| [piggysec.com/image-skip-verify-registry](#image-skip-verify-registry)                     | string  |         | Pods     |       |
| [piggysec.com/piggy-enforce-service-account](#piggy-enforce-service-account)               | bool    | false   | Pods     |       |
| [piggysec.com/piggy-default-secret-name-prefix](#piggy-default-secret-name-prefix)         | string  |         | Pods     |       |
| [piggysec.com/piggy-default-secret-name-suffix](#piggy-default-secret-name-suffix)         | string  |         | Pods     |       |
| [piggysec.com/piggy-dns-resolver](#piggy-dns-resolver)                                     | string  |         | Pods     |       |
| [piggysec.com/piggy-delay-second](#piggy-delay-second)                                     | int     | 0       | Pods     |       |

## AWS Secret Manager

  - <a name="aws-secret-name">`piggysec.com/aws-secret-name`</a> specifies a AWS secret name e.g. "/myapp/name"
  - <a name="aws-region">`piggysec.com/aws-region`</a> specifies a AWS secret manager region e.g. "ap-southeast-1"

## piggy-env settings

  - <a name="piggy-env-image">`piggysec.com/piggy-env-image`</a> overrides a piggy-env image location. If no value specifies, the piggy-env   image location will be taken from piggy-webhooks settings on helm chart
  - <a name="piggy-env-image-pull-policy">`piggysec.com/piggy-env-image-pull-policy`</a> overrides a piggy-env image pull policy. If no   value specifies, the piggy-env image pull policy will be taken from piggy-webhooks settings on helm chart
  - <a name="piggy-env-resource-cpu-request">`piggysec.com/piggy-env-resource-cpu-request`</a> overrides a piggy-env init-container   resource CPU requests. Default to `50m`
  - <a name="piggy-env-resource-memory-request">`piggysec.com/piggy-env-resource-memory-request`</a> overrides a piggy-env init-container   resource memory requests. Default to `64Mi`
  - <a name="piggy-env-resource-cpu-limit">`piggysec.com/piggy-env-resource-cpu-limit`</a> overrides a piggy-env init-container resource   CPU limit. Default to `200m`
  - <a name="piggy-env-resource-memory-limit">`piggysec.com/piggy-env-resource-memory-limit`</a> overrides a piggy-env init-container   resource memory limit. Default to `64Mi`
  - <a name="piggy-psp-allow-privilege-escalation">`piggysec.com/piggy-psp-allow-privilege-escalation`</a> allow a piggy-env init-container   to run as root. Default to `false`
  - <a name="piggy-address">`piggysec.com/piggy-address`</a> an endpoint of piggy-webhooks. This is required when it is running in proxy   mode.
  - <a name="piggy-ignore-no-env">`piggysec.com/piggy-ignore-no-env`</a> do not terminate the container if no variable found on secret   manager. Default to `false`. Set this value to `false` is recommended in most application. The container will not start if environment   variable is missing.
  - <a name="piggy-enforce-integrity">`piggysec.com/piggy-enforce-integrity`</a> enforce checking command integrity before inject secrets   into. Default to `true`. Set this value to `true` is recommended in most application. Set to `false` will allow piggy-env to run on   different arguments
  - <a name="debug">`piggysec.com/debug`</a> allows to run piggy-env in debug mode. Default to `false`.
  - <a name="standalone">`piggysec.com/standalone`</a> allows to run piggy-env in standalone mode. Default to `false`. If this value is `true`, the [piggysec.com/piggy-address](#piggy-address) will not be used.
  - <a name="piggy-enforce-service-account">`piggysec.com/piggy-enforce-service-account`</a> Force to check `PIGGY_ALLOWED_SA` env value in AWS secret manager
  - <a name="piggy-default-secret-name-prefix">`piggysec.com/piggy-default-secret-name-prefix`</a>Set default prefix string for secret name
  - <a name="piggy-default-secret-name-suffix">`piggysec.com/piggy-default-secret-name-suffix`</a>Set default suffix string for secret name
  - <a name="piggy-dns-resolver">`piggysec.com/piggy-dns-resolver`</a>Set Golang DNS resolver such as `tcp`, `udp`. See [https://pkg.go.dev/net](https://pkg.go.dev/net)
  - <a name="piggy-delay-second">`piggysec.com/piggy-delay-second`</a>Set delay in second before start retrieving secrets. If you are using Istio Envoy, you may need to set this value to 1. The Envoy will block all outgoing requests from piggy-env until Envoy is fully started. Add this delay value to allow Envoy to operate before running piggy.
  
## Container image settings

  - <a name="image-pull-secret">`piggysec.com/image-pull-secret`</a> a name of container image pull secret. The piggy will try to read the   container image by using secret in the following order
    1) pod.spec.imagePullSecrets
    2) `piggysec.com/image-pull-secret` annotation
    3) ServiceAccount permission from cloud
  - <a name="image-pull-secret-namespace">`piggysec.com/image-pull-secret-namespace`</a> a name of container image pull secret namespace.
  - <a name="image-skip-verify-registry">`piggysec.com/image-skip-verify-registry`</a> skip verify registry when trying to read the image. Default to `true`
