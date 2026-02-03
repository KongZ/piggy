package mutate

import (
	"context"
	"testing"

	"github.com/KongZ/piggy/piggy-webhooks/service"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// TestGetSecurityContext verifies that the security context is correctly derived from PiggyConfig and PodSecurityContext.
func TestGetSecurityContext(t *testing.T) {
	config := &service.PiggyConfig{
		PiggyPspAllowPrivilegeEscalation: false,
	}

	// Case 1: No pod security context
	sc := getSecurityContext(config, nil)
	assert.False(t, *sc.AllowPrivilegeEscalation)
	assert.Nil(t, sc.RunAsUser)

	// Case 2: With pod security context
	runAsUser := int64(1000)
	psc := &corev1.PodSecurityContext{
		RunAsUser: &runAsUser,
	}
	sc = getSecurityContext(config, psc)
	assert.Equal(t, &runAsUser, sc.RunAsUser)
}

// TestMutateCommand ensures that the container command and arguments are correctly modified for Piggy.
func TestMutateCommand(t *testing.T) {
	ctx := context.Background()
	client := fake.NewClientset()
	m, _ := NewMutating(ctx, client)
	m.registry = NewRegistry(&service.PiggyConfig{})

	config := &service.PiggyConfig{}
	pod := &corev1.Pod{}

	// Case 1: Already mutated
	c1 := &corev1.Container{
		Command: []string{"/piggy/piggy-env"},
		Args:    []string{"--", "ls", "-la"},
	}
	args, mutated, err := m.mutateCommand(config, c1, pod)
	assert.NoError(t, err)
	assert.False(t, mutated)
	assert.Equal(t, []string{"ls", "-la"}, args)

	// Case 2: Empty command, mock registry
	c2 := &corev1.Container{
		Image: "my-image",
	}
	m.registry.imageCache["my-image"] = &v1.Config{
		Entrypoint: []string{"sh"},
		Cmd:        []string{"-c", "echo hello"},
	}
	args, mutated, err = m.mutateCommand(config, c2, pod)
	assert.NoError(t, err)
	assert.True(t, mutated)
	assert.Equal(t, []string{"sh", "-c", "echo hello"}, args)
	assert.Equal(t, []string{"/piggy/piggy-env"}, c2.Command)
	assert.Equal(t, []string{"--", "sh", "-c", "echo hello"}, c2.Args)

	// Case 3: Only Command
	c3 := &corev1.Container{
		Command: []string{"python", "app.py"},
	}
	args, mutated, err = m.mutateCommand(config, c3, pod)
	assert.NoError(t, err)
	assert.True(t, mutated)
	assert.Equal(t, []string{"python", "app.py"}, args)
}

// TestMutateContainer_Skip verifies that containers without Piggy annotations are skipped during mutation.
func TestMutateContainer_Skip(t *testing.T) {
	ctx := context.Background()
	client := fake.NewClientset()
	m, _ := NewMutating(ctx, client)

	config := &service.PiggyConfig{}
	pod := &corev1.Pod{}
	container := &corev1.Container{
		Name: "test",
		Env: []corev1.EnvVar{
			{Name: "NORMAL", Value: "value"},
		},
	}

	sig, mutated, err := m.mutateContainer("uid", config, container, pod)
	assert.NoError(t, err)
	assert.False(t, mutated)
	assert.Empty(t, sig)
}

// TestMutateContainer_EnvFrom ensures that Piggy correctly handles and expands environment variables from EnvFrom sources.
func TestMutateContainer_EnvFrom(t *testing.T) {
	ctx := context.Background()
	ns := "default"
	cmName := "test-cm"
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: cmName, Namespace: ns},
		Data: map[string]string{
			"key1": "piggy:secret1",
		},
	}
	client := fake.NewClientset(cm)
	m, _ := NewMutating(ctx, client)
	m.registry = NewRegistry(&service.PiggyConfig{})

	config := &service.PiggyConfig{
		AWSSecretName: "my-aws-secret",
	}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: ns}}
	container := &corev1.Container{
		Name: "test",
		EnvFrom: []corev1.EnvFromSource{
			{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: cmName},
				},
			},
		},
	}

	sig, mutated, err := m.mutateContainer("uid", config, container, pod)
	assert.NoError(t, err)
	assert.True(t, mutated)
	assert.NotEmpty(t, sig)

	// Verify injection
	found := false
	for _, env := range container.Env {
		if env.Name == "PIGGY_AWS_SECRET_NAME" {
			assert.Equal(t, "my-aws-secret", env.Value)
			found = true
		}
	}
	assert.True(t, found)
}

// TestMutatePod_GranularIdempotency verifies that repeated mutations on the same pod are handled correctly without duplicate resource injection.
func TestMutatePod_GranularIdempotency(t *testing.T) {
	ctx := context.Background()
	client := fake.NewClientset()
	m, _ := NewMutating(ctx, client)
	m.registry = NewRegistry(&service.PiggyConfig{})

	config := &service.PiggyConfig{
		AWSSecretName: "my-secret",
		PiggyImage:    "piggy-image:v1",
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "app",
					Env: []corev1.EnvVar{
						{Name: "DB", Value: "piggy:db"},
					},
				},
			},
		},
	}

	// First mutation
	_, err := m.MutatePod(config, pod)
	assert.NoError(t, err)
	assert.Len(t, pod.Spec.Volumes, 1)
	assert.Len(t, pod.Spec.InitContainers, 1)
	assert.Equal(t, "install-piggy-env", pod.Spec.InitContainers[0].Name)
	assert.Equal(t, corev1.ContainerRestartPolicyAlways, *pod.Spec.InitContainers[0].RestartPolicy)

	// Second mutation (reinvocation)
	_, err = m.MutatePod(config, pod)
	assert.NoError(t, err)
	// Should not duplicate
	assert.Len(t, pod.Spec.Volumes, 1)
	assert.Len(t, pod.Spec.InitContainers, 1)
}
