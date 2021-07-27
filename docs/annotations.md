#  Pod annotations
You can add annotations to kubernetes Pods objects to customize piggy behavior.

## Annotations
|Name                       | Type |Default|Location|Notes|
|---------------------------|------|-------|--------|------|
|[piggy.kong-z.com/aws-secret-name](#aws-secret-name)|string||Pods||
|[piggy.kong-z.com/aws-region](#aws-region)|string||Pods||
|[piggy.kong-z.com/piggy-env-image](#piggy-env-image)|string||Pods||
|[piggy.kong-z.com/piggy-env-image-pull-policy](#piggy-env-image-pull-policy)|string||Pods||
|[piggy.kong-z.com/piggy-env-resource-cpu-request](#piggy-env-resource-cpu-request)|string||Pods||
|[piggy.kong-z.com/piggy-env-resource-memory-request](#piggy-env-resource-memory-request)|string||Pods||
|[piggy.kong-z.com/piggy-env-resource-cpu-limit](#piggy-env-resource-cpu-limit)|string||Pods||
|[piggy.kong-z.com/piggy-env-resource-memory-limit](#piggy-env-resource-memory-limit)|string||Pods||
|[piggy.kong-z.com/piggy-psp-allow-privilege-escalation](#piggy-psp-allow-privilege-escalation)|boolean|false|Pods||
|[piggy.kong-z.com/piggy-address](#piggy-address)|string||Pods||
|[piggy.kong-z.com/piggy-skip-verify-tls](#piggy-skip-verify-tls)|boolean|true|Pods||
|[piggy.kong-z.com/piggy-ignore-no-env](#piggy-ignore-no-env)|boolean|false|Pods||
|[piggy.kong-z.com/debug](#debug)|boolean|false|Pods||
|[piggy.kong-z.com/standalone](#standalone)|boolean|false|Pods||
|[piggy.kong-z.com/image-pull-secret](#image-pull-secret)|string||Pods||
|[piggy.kong-z.com/image-pull-secret-namespace](#image-pull-secret-namespace)|string||Pods||
|[piggy.kong-z.com/image-skip-verify-registry](#image-skip-verify-registry)|string||Pods||

## AWS Secret Manager

- <a name="aws-secret-name">`piggy.kong-z.com/aws-secret-name`</a> specifies a AWS secret name e.g. "/myapp/name"
- <a name="aws-region">`piggy.kong-z.com/aws-region`</a> specifies a AWS secret manager region e.g. "ap-southeast-1"

## piggy-env settings

- <a name="piggy-env-image">`piggy.kong-z.com/piggy-env-image`</a> overrides a piggy-env image location. If no value specifies, the piggy-env image location will be taken from piggy-webhooks settings on helm chart
- <a name="piggy-env-image-pull-policy">`piggy.kong-z.com/piggy-env-image-pull-policy`</a> overrides a piggy-env image pull policy. If no value specifies, the piggy-env image pull policy will be taken from piggy-webhooks settings on helm chart
- <a name="piggy-env-resource-cpu-request">`piggy.kong-z.com/piggy-env-resource-cpu-request`</a> overrides a piggy-env init-container resource CPU requests. Default to `50m`
- <a name="piggy-env-resource-memory-request">`piggy.kong-z.com/piggy-env-resource-memory-request`</a> overrides a piggy-env init-container resource memory requests. Default to `64Mi`
- <a name="piggy-env-resource-cpu-limit">`piggy.kong-z.com/piggy-env-resource-cpu-limit`</a> overrides a piggy-env init-container resource CPU limit. Default to `200m`
- <a name="piggy-env-resource-memory-limit">`piggy.kong-z.com/piggy-env-resource-memory-limit`</a> overrides a piggy-env init-container resource memory limit. Default to `64Mi`
- <a name="piggy-psp-allow-privilege-escalation">`piggy.kong-z.com/piggy-psp-allow-privilege-escalation`</a> allow a piggy-env init-container to run as root. Default to `false`
- <a name="piggy-address">`piggy.kong-z.com/piggy-address`</a> an endpoint of piggy-webhooks. This is required when it is running in lookup mode.
- <a name="piggy-ignore-no-env">`piggy.kong-z.com/piggy-ignore-no-env`</a> do not terminate the container if no variable found on secret manager. Default to `false`. Set this value to `false` is recommended in most application. The container will not start if environment variable is missing.
- <a name="debug">`piggy.kong-z.com/debug`</a> allows to run piggy-env in debug mode. Default to `false`.
- <a name="standalone">`piggy.kong-z.com/standalone`</a> allows to run piggy-env in standalone mode. Default to `false`. If this value is `true`, the [piggy.kong-z.com/piggy-address](#piggy-address) will not be used.

## Container image settings

- <a name="image-pull-secret">`piggy.kong-z.com/image-pull-secret`</a> a name of container image pull secret. The piggy will try to read the container image by using secret in the following order 
  1) pod.spec.imagePullSecrets
	2) `piggy.kong-z.com/image-pull-secret` annotation
	3) ServiceAccount permission from cloud
- <a name="image-pull-secret-namespace">`piggy.kong-z.com/image-pull-secret-namespace`</a> a name of container image pull secret namespace.
- <a name="image-skip-verify-registry">`piggy.kong-z.com/image-skip-verify-registry`</a> skip verify registry when trying to read the image. Default to `true`
