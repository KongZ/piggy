package mutate

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/KongZ/piggy/piggy-webhooks/service"
	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestIsKubeNamespace(t *testing.T) {
	assert.True(t, IsKubeNamespace(metav1.NamespacePublic))
	assert.True(t, IsKubeNamespace(metav1.NamespaceSystem))
	assert.False(t, IsKubeNamespace("default"))
	assert.False(t, IsKubeNamespace("my-app"))
}

func TestNewMutating(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	m, err := NewMutating(ctx, client)
	assert.NoError(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, client, m.k8sClient)
}

func TestGenerateUid(t *testing.T) {
	m, _ := NewMutating(context.Background(), fake.NewSimpleClientset())
	uid1 := m.generateUid()
	uid2 := m.generateUid()
	assert.NotEmpty(t, uid1)
	assert.NotEqual(t, uid1, uid2)
}

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
	
	client := fake.NewSimpleClientset(cm, secret)
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
	
	client := fake.NewSimpleClientset(cm)
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
}

func TestApplyPiggy(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
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
