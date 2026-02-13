package mutate

import (
	"context"
	"testing"

	"github.com/KongZ/piggy/piggy-webhooks/service"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

// TestNewRegistry ensures correct initialization of the image registry client and cache.
func TestNewRegistry(t *testing.T) {
	config := &service.PiggyConfig{}
	r := NewRegistry(config)
	assert.NotNil(t, r)
	assert.Equal(t, config, r.config)
	assert.NotNil(t, r.imageCache)
}

// TestIsAllowedToCache checks the logic for determining which container images can be cached.
func TestIsAllowedToCache(t *testing.T) {
	containerAlways := corev1.Container{ImagePullPolicy: corev1.PullAlways}
	assert.False(t, isAllowedToCache(containerAlways))

	containerLatest := corev1.Container{Image: "my-image:latest"}
	assert.False(t, isAllowedToCache(containerLatest))

	containerV1 := corev1.Container{Image: "my-image:v1"}
	assert.True(t, isAllowedToCache(containerV1))

	containerInvalid := corev1.Container{Image: "::invalid::"}
	assert.False(t, isAllowedToCache(containerInvalid))
}

// TestGetImageConfig_Cache verifies that cached image configurations are returned correctly.
func TestGetImageConfig_Cache(t *testing.T) {
	config := &service.PiggyConfig{}
	r := NewRegistry(config)

	imageName := "my-image:v1"
	cachedConfig := &v1.Config{Entrypoint: []string{"/app"}}
	r.imageCache[imageName] = cachedConfig

	ctx := context.Background()
	container := corev1.Container{Image: imageName}
	podSpec := corev1.PodSpec{}

	result, err := r.GetImageConfig(ctx, config, "default", container, podSpec)
	assert.NoError(t, err)
	assert.Equal(t, cachedConfig, result)
}

// TestGetImageConfig_Fetch verifies fetching logic by mocking the image fetcher.
func TestGetImageConfig_Fetch(t *testing.T) {
	config := &service.PiggyConfig{}
	r := NewRegistry(config)

	expectedConfig := &v1.Config{Entrypoint: []string{"/bin/sh"}}

	// Mock the fetcher
	r.imageFetcher = func(ctx context.Context, config *service.PiggyConfig, container containerInfo) (*v1.Config, error) {
		return expectedConfig, nil
	}

	container := corev1.Container{Image: "new-image:v1"}
	// Ensure not in cache
	assert.Nil(t, r.imageCache[container.Image])

	result, err := r.GetImageConfig(context.Background(), config, "default", container, corev1.PodSpec{})
	assert.NoError(t, err)
	assert.Equal(t, expectedConfig, result)

	// Verify it got cached
	assert.Equal(t, expectedConfig, r.imageCache[container.Image])

	// Test case with ImagePullSecrets
	config.ImagePullSecret = "global-secret"
	config.ImagePullSecretNamespace = "piggy-ns"
	containerWithSecrets := corev1.Container{Image: "secret-image:v1"}

	podSpec := corev1.PodSpec{
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: "pod-secret"},
		},
	}

	// Mock fetcher to verify secrets are passed correctly
	r.imageFetcher = func(ctx context.Context, config *service.PiggyConfig, container containerInfo) (*v1.Config, error) {
		assert.Equal(t, "piggy-ns", container.Namespace)
		assert.Contains(t, container.ImagePullSecrets, "global-secret")
		assert.Contains(t, container.ImagePullSecrets, "pod-secret")
		return expectedConfig, nil
	}

	result, err = r.GetImageConfig(context.Background(), config, "default", containerWithSecrets, podSpec)
	assert.NoError(t, err)
	assert.Equal(t, expectedConfig, result)
}
