package mutate

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/KongZ/piggy/piggy-webhooks/service"
	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

// TestIsKubeNamespace verifies the identification of system namespaces.
func TestIsKubeNamespace(t *testing.T) {
	assert.True(t, IsKubeNamespace(metav1.NamespacePublic))
	assert.True(t, IsKubeNamespace(metav1.NamespaceSystem))
	assert.False(t, IsKubeNamespace("default"))
	assert.False(t, IsKubeNamespace("my-app"))
}

// TestNewMutating ensures that a new Mutating object is correctly initialized.
func TestNewMutating(t *testing.T) {
	ctx := context.Background()
	client := fake.NewClientset()
	m, err := NewMutating(ctx, client)
	assert.NoError(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, client, m.k8sClient)
}

// TestGenerateUid checks the randomness and format of generated UIDs.
func TestGenerateUid(t *testing.T) {
	m, _ := NewMutating(context.Background(), fake.NewClientset())
	uid1 := m.generateUID()
	uid2 := m.generateUID()
	assert.NotEmpty(t, uid1)
	assert.NotEqual(t, uid1, uid2)
}

// TestLookForValueFrom verifies that Piggy references are correctly identified in ConfigMaps and Secrets.
func TestLookForValueFrom(t *testing.T) {
	ctx := context.Background()
	ns := "default"
	cmName := "test-cm"
	secretName := "test-secret"

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: cmName, Namespace: ns},
		Data: map[string]string{
			"key1": "piggy:secret1",
			"key2": "normal-value",
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: ns},
		Data: map[string][]byte{
			"skey1": []byte("piggy:secret2"),
		},
	}

	client := fake.NewClientset(cm, secret)
	m, _ := NewMutating(ctx, client)

	// Test ConfigMap match
	envCM := corev1.EnvVar{
		Name: "VAR1",
		ValueFrom: &corev1.EnvVarSource{
			ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: cmName},
				Key:                  "key1",
			},
		},
	}
	result, err := m.LookForValueFrom(envCM, ns)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "piggy:secret1", result.Value)

	// Test ConfigMap no match (normal value)
	envCM2 := corev1.EnvVar{
		Name: "VAR2",
		ValueFrom: &corev1.EnvVarSource{
			ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: cmName},
				Key:                  "key2",
			},
		},
	}
	result, err = m.LookForValueFrom(envCM2, ns)
	assert.NoError(t, err)
	assert.Nil(t, result)

	// Test Secret match
	envSecret := corev1.EnvVar{
		Name: "VAR3",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
				Key:                  "skey1",
			},
		},
	}
	result, err = m.LookForValueFrom(envSecret, ns)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "piggy:secret2", result.Value)
}

// TestLookForEnvFrom ensures that all environment variables with Piggy prefixes
// are extracted from a ConfigMap reference.
func TestLookForEnvFrom(t *testing.T) {
	ctx := context.Background()
	ns := "default"
	cmName := "test-cm"

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: cmName, Namespace: ns},
		Data: map[string]string{
			"key1": "piggy:secret1",
			"key2": "normal-value",
		},
	}

	client := fake.NewClientset(cm)
	m, _ := NewMutating(ctx, client)

	envFrom := []corev1.EnvFromSource{
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: cmName},
			},
		},
	}

	results, err := m.LookForEnvFrom(envFrom, ns)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "key1", results[0].Name)
	assert.Equal(t, "piggy:secret1", results[0].Value)

	// Test Secret match
	secretName := "test-secret"
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: ns},
		Data: map[string][]byte{
			"skey1": []byte("piggy:secret2"),
		},
	}
	client = fake.NewClientset(secret)
	m, _ = NewMutating(ctx, client)

	envFrom = []corev1.EnvFromSource{
		{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
			},
		},
	}

	results, err = m.LookForEnvFrom(envFrom, ns)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "skey1", results[0].Name)
	assert.Equal(t, "piggy:secret2", results[0].Value)
}

// TestApplyPiggy checks the high-level mutation logic for a pod admission request.
func TestApplyPiggy(t *testing.T) {
	ctx := context.Background()
	client := fake.NewClientset()
	m, _ := NewMutating(ctx, client)

	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
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
	rawPod, _ := json.Marshal(pod)

	req := &admissionv1.AdmissionRequest{
		Resource: metav1.GroupVersionResource{Version: "v1", Resource: "pods"},
		Object: runtime.RawExtension{
			Raw: rawPod,
		},
		Namespace: "default",
	}

	result, err := m.ApplyPiggy(req)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// ApplyPiggy returns a mutated pod directly if it matches the resource
	mutatedPod, ok := result.(*corev1.Pod)
	assert.True(t, ok)
	assert.Equal(t, "test-pod", mutatedPod.Name)
}

// TestLookForValueFrom_ErrorCases verifies handling of missing ConfigMaps/Secrets in ValueFrom.
func TestLookForValueFrom_ErrorCases(t *testing.T) {
	ctx := context.Background()
	client := fake.NewClientset()
	m, _ := NewMutating(ctx, client)

	// ConfigMap not found (should return nil, nil)
	envCM := corev1.EnvVar{
		ValueFrom: &corev1.EnvVarSource{
			ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "missing"},
				Key:                  "k",
			},
		},
	}
	res, err := m.LookForValueFrom(envCM, "default")
	assert.NoError(t, err)
	assert.Nil(t, res)

	// Secret not found (should return nil, nil)
	envSec := corev1.EnvVar{
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "missing"},
				Key:                  "k",
			},
		},
	}
	res, err = m.LookForValueFrom(envSec, "default")
	assert.NoError(t, err)
	assert.Nil(t, res)
}

// TestLookForEnvFrom_ErrorCases verifies handling of missing or optional sources in EnvFrom.
func TestLookForEnvFrom_ErrorCases(t *testing.T) {
	ctx := context.Background()
	client := fake.NewClientset()
	m, _ := NewMutating(ctx, client)

	optional := true
	envFrom := []corev1.EnvFromSource{
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: "missing"},
				Optional:             &optional,
			},
		},
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: "missing-but-handled"},
			},
		},
	}

	results, err := m.LookForEnvFrom(envFrom, "default")
	assert.NoError(t, err)
	assert.Empty(t, results)
}

// TestMergeConfig ensures that Piggy configuration is correctly merged from pod annotations and environment variables.
func TestMergeConfig(t *testing.T) {
	m, _ := NewMutating(context.Background(), fake.NewClientset())
	config := &service.PiggyConfig{}

	// Set an env var for fallback
	err := os.Setenv("PIGGY_ADDRESS", "http://env-address")
	assert.NoError(t, err)
	defer func() { _ = os.Unsetenv("PIGGY_ADDRESS") }()

	annotations := map[string]string{
		service.Namespace + service.ConfigPiggyEnvImage:                    "custom-image",
		service.Namespace + service.ConfigPiggyEnvImagePullPolicy:          "IfNotPresent",
		service.Namespace + service.ConfigPiggyEnvResourceCPURequest:       "100m",
		service.Namespace + service.ConfigPiggyEnvResourceMemoryRequest:    "128Mi",
		service.Namespace + service.ConfigPiggyEnvResourceCPULimit:         "500m",
		service.Namespace + service.ConfigPiggyEnvResourceMemoryLimit:      "256Mi",
		service.Namespace + service.ConfigPiggyPSPAllowPrivilegeEscalation: "true",
		service.Namespace + service.AWSSecretName:                          "my-secret",
		service.Namespace + service.ConfigDebug:                            "true",
	}

	m.mergeConfig(config, annotations)
	assert.Equal(t, "custom-image", config.PiggyImage)
	assert.Equal(t, corev1.PullIfNotPresent, config.PiggyImagePullPolicy)
	assert.Equal(t, "100m", config.PiggyResourceCPURequest.String())
	assert.Equal(t, "my-secret", config.AWSSecretName)
	assert.True(t, config.PiggyPspAllowPrivilegeEscalation)
	assert.True(t, config.Debug)
	assert.Equal(t, "http://env-address", config.PiggyAddress)
}
