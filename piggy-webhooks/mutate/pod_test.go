package mutate

import (
	"context"
	"testing"

	"github.com/KongZ/piggy/piggy-webhooks/service"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestMutatePod_EmptyPod(t *testing.T) {
	m, _ := NewMutating(context.Background(), fake.NewClientset())
	config := &service.PiggyConfig{}
	pod := &corev1.Pod{}
	
	patch, err := m.MutatePod(config, pod)
	assert.NoError(t, err)
	assert.Nil(t, patch)
}

func TestMutatePod_InjectedPod(t *testing.T) {
	m, _ := NewMutating(context.Background(), fake.NewClientset())
	config := &service.PiggyConfig{
		AWSSecretName: "my-secret",
		PiggyImage:    "piggy-env:latest",
	}
	m.registry = NewRegistry(config)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				service.Namespace + service.AWSSecretName: "my-secret",
			},
		},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{},
			Containers: []corev1.Container{
				{
					Name: "app",
					Env: []corev1.EnvVar{
						{Name: "DB_PASS", Value: "piggy:db-pass"},
					},
				},
			},
		},
	}
	
	patch, err := m.MutatePod(config, pod)
	assert.NoError(t, err)
	assert.NotNil(t, patch)
}
