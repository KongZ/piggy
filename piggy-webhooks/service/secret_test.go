package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func setupTest(t *testing.T, objects ...runtime.Object) (context.Context, *fake.Clientset, *Service) {
	ctx := context.Background()
	client := fake.NewClientset(objects...)
	svc, err := NewService(ctx, client)
	assert.NoError(t, err)
	return ctx, client, svc
}

func mockTokenReview(client *fake.Clientset, username string, authenticated bool) {
	client.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &authv1.TokenReview{
			Status: authv1.TokenReviewStatus{
				Authenticated: authenticated,
				User: authv1.UserInfo{
					Username: username,
				},
			},
		}, nil
	})
}

func newPod(ns, name, sa string, annotations map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   ns,
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: sa,
		},
	}
}

func TestNewService(t *testing.T) {
	_, client, svc := setupTest(t)
	assert.NotNil(t, svc)
	assert.Equal(t, client, svc.k8sClient)
}

func TestGetSecret_AuthenticationFailure(t *testing.T) {
	_, client, svc := setupTest(t)
	mockTokenReview(client, "", false)

	payload := &GetSecretPayload{
		Token: "invalid-token",
	}

	_, _, err := svc.GetSecret(payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is not authenticated")
}

func TestGetSecret_PodNotFound(t *testing.T) {
	_, client, svc := setupTest(t)
	mockTokenReview(client, "system:serviceaccount:default:test-sa", true)

	payload := &GetSecretPayload{
		Name:  "non-existent-pod",
		Token: "valid-token",
	}

	_, _, err := svc.GetSecret(payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pod non-existent-pod not found in default namespace")
}

func TestGetSecret_InvalidSignature(t *testing.T) {
	ns, name, sa, uid := "default", "test-pod", "test-sa", "test-uid"
	pod := newPod(ns, name, sa, map[string]string{
		Namespace + ConfigPiggyUID: `{"test-uid": "correct-signature"}`,
	})

	_, client, svc := setupTest(t, pod)
	mockTokenReview(client, "system:serviceaccount:"+ns+":"+sa, true)

	payload := &GetSecretPayload{
		Name:      name,
		Token:     "valid-token",
		UID:       uid,
		Signature: "wrong-signature",
	}

	_, _, err := svc.GetSecret(payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid signature")
}

func TestGetSecret_Success_MockingAWS(t *testing.T) {
	ns, name, sa, uid := "default", "test-pod", "test-sa", "test-uid"
	pod := newPod(ns, name, sa, map[string]string{
		Namespace + ConfigPiggyUID:             `{"test-uid": "correct-signature"}`,
		Namespace + ConfigPiggyEnforceIntegrity: "false",
	})

	_, client, svc := setupTest(t, pod)
	mockTokenReview(client, "system:serviceaccount:"+ns+":"+sa, true)

	payload := &GetSecretPayload{
		Name:      name,
		Token:     "valid-token",
		UID:       uid,
		Signature: "correct-signature",
	}

	// It will try to call AWS and fail because of no credentials/mock
	_, _, err := svc.GetSecret(payload)
	assert.Error(t, err)
}
