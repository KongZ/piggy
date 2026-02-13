package mutate

import (
	"context"
	"crypto/tls"
	"net/http"

	"github.com/KongZ/piggy/piggy-webhooks/service"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
)

type containerInfo struct {
	Namespace          string
	ImagePullSecrets   []string
	ServiceAccountName string
	Image              string
}

// ImageRegistry object
type ImageRegistry struct {
	imageCache   map[string]*v1.Config
	config       *service.PiggyConfig
	imageFetcher func(ctx context.Context, config *service.PiggyConfig, container containerInfo) (*v1.Config, error)
}

// NewRegistry creates and initializes registry
func NewRegistry(config *service.PiggyConfig) *ImageRegistry {
	return &ImageRegistry{
		imageCache:   make(map[string]*v1.Config),
		config:       config,
		imageFetcher: getImageConfig,
	}
}

func isAllowedToCache(container corev1.Container) bool {
	if container.ImagePullPolicy == corev1.PullAlways {
		return false
	}
	if reference, err := name.ParseReference(container.Image); err == nil {
		return reference.Identifier() != "latest"
	}
	return false
}

func getImageConfig(ctx context.Context, config *service.PiggyConfig, container containerInfo) (*v1.Config, error) {
	log.Debug().Msgf("Reading image %s", container.Image)
	kc, err := k8schain.NewInCluster(ctx, k8schain.Options{
		Namespace:          container.Namespace,
		ServiceAccountName: container.ServiceAccountName,
		ImagePullSecrets:   container.ImagePullSecrets,
		UseMountSecrets:    true,
	})
	if err != nil {
		return nil, err
	}
	options := []remote.Option{
		remote.WithAuthFromKeychain(kc),
	}

	if config.ImageSkipVerifyRegistry {
		tr := &http.Transport{
			// #nosec G402 possible self-sign
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // nolint:gosec
		}
		options = append(options, remote.WithTransport(tr))
	}
	ref, err := name.ParseReference(container.Image)
	if err != nil {
		return nil, err
	}

	descriptor, err := remote.Get(ref, options...)
	if err != nil {
		return nil, err
	}

	image, err := descriptor.Image()
	if err != nil {
		return nil, err
	}

	configFile, err := image.ConfigFile()
	if err != nil {
		return nil, err
	}

	return &configFile.Config, nil
}

// GetImageConfig returns entrypoint and command of container
func (r *ImageRegistry) GetImageConfig(ctx context.Context, config *service.PiggyConfig, namespace string, container corev1.Container, podSpec corev1.PodSpec) (*v1.Config, error) {
	if imageConfig, found := r.imageCache[container.Image]; found {
		log.Debug().Msgf("found image %s in cache", container.Image)
		return imageConfig, nil
	}
	containerInfo := containerInfo{
		Namespace:          namespace,
		ServiceAccountName: podSpec.ServiceAccountName,
		Image:              container.Image,
	}
	if config.ImagePullSecretNamespace != "" {
		containerInfo.Namespace = config.ImagePullSecretNamespace
	}
	// ImagePullSecret
	// 1) pod.spec.imagePullSecrets
	// 2) config.ImagePullSecret
	// 3) ServiceAccount permission from cloud
	containerInfo.ImagePullSecrets = make([]string, len(podSpec.ImagePullSecrets))
	for _, imagePullSecret := range podSpec.ImagePullSecrets {
		if imagePullSecret.Name != "" {
			containerInfo.ImagePullSecrets = append(containerInfo.ImagePullSecrets, imagePullSecret.Name)
		}
	}
	if config.ImagePullSecret != "" {
		containerInfo.ImagePullSecrets = append(containerInfo.ImagePullSecrets, config.ImagePullSecret)
	}
	log.Debug().Msgf("Container info %+v", containerInfo)
	imageConfig, err := r.imageFetcher(ctx, config, containerInfo)
	if imageConfig != nil && isAllowedToCache(container) {
		r.imageCache[container.Image] = imageConfig
	}
	return imageConfig, err
}
