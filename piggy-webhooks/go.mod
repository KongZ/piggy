module github.com/KongZ/piggy/piggy-webhooks

go 1.16

require (
	github.com/aws/aws-sdk-go v1.39.5
	github.com/google/go-containerregistry v0.5.2-0.20210609162550-f0ce2270b3b4
	github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20210709161016-b448abac9a70
	github.com/mattbaird/jsonpatch v0.0.0-20200820163806-098863c1fc24
	github.com/rs/zerolog v1.23.0
	k8s.io/api v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
	sigs.k8s.io/controller-runtime v0.9.2
)
