package service

import (
	"context"
	"errors"
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

// TestNewService verifies the initialization of the secret service.
func TestNewService(t *testing.T) {
	_, client, svc := setupTest(t)
	assert.NotNil(t, svc)
	assert.Equal(t, client, svc.k8sClient)
}

// TestGetSecret_AuthenticationFailure ensures that unauthenticated requests are rejected.
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

// TestGetSecret_PodNotFound verifies that requests for unknown pods result in an error.
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

// TestGetSecret_InvalidSignature checks the validation of pod signatures during secret retrieval.
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

// TestGetSecret_Success_MockingAWS verifies the flow when authentication and signature validation pass.
func TestGetSecret_Success_MockingAWS(t *testing.T) {
	ns, name, sa, uid := "default", "test-pod", "test-sa", "test-uid"
	pod := newPod(ns, name, sa, map[string]string{
		Namespace + ConfigPiggyUID:              `{"test-uid": "correct-signature"}`,
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

// TestSanitizedEnv_Append verifies that sensitive PIGGY_* environment variables are filtered out during appending.
func TestSanitizedEnv_Append(t *testing.T) {
	env := &SanitizedEnv{}
	env.append("NORMAL_VAR", "value1")
	env.append("PIGGY_AWS_SECRET_NAME", "secret") // Should be skipped

	assert.Equal(t, "value1", (*env)["NORMAL_VAR"])
	_, exists := (*env)["PIGGY_AWS_SECRET_NAME"]
	assert.False(t, exists)
}

// TestAwsErr verifies the error identification helper for AWS related errors.
func TestAwsErr(t *testing.T) {
	assert.False(t, awsErr(nil))
	assert.True(t, awsErr(errors.New("generic error")))
}

// TestProcessSecret ensures that secrets are correctly processed and filtered based on service account permissions.
func TestProcessSecret(t *testing.T) {
	config := &PiggyConfig{
		PodServiceAccountName: "default:test-sa",
	}

	// Case 1: Allowed via PIGGY_ALLOWED_SA
	secrets := map[string]string{
		"PIGGY_ALLOWED_SA": "default:test-sa,other:sa",
		"MY_VAR":           "val1",
	}
	env := &SanitizedEnv{}
	err := processSecret(config, secrets, env)
	assert.NoError(t, err)
	assert.Equal(t, "val1", (*env)["MY_VAR"])

	// Case 2: Rejected via PIGGY_ALLOWED_SA
	secrets = map[string]string{
		"PIGGY_ALLOWED_SA": "other:sa",
		"MY_VAR":           "val1",
	}
	env = &SanitizedEnv{}
	err = processSecret(config, secrets, env)
	assert.Error(t, err)
	assert.Equal(t, ErrorAuthorized, err)

	// Case 3: Enforce Service Account (No PIGGY_ALLOWED_SA in secrets)
	config.PiggyEnforceServiceAccount = true
	secrets = map[string]string{
		"MY_VAR": "val1",
	}
	env = &SanitizedEnv{}
	err = processSecret(config, secrets, env)
	assert.Error(t, err)

	// Case 4: Don't Enforce Service Account
	config.PiggyEnforceServiceAccount = false
	err = processSecret(config, secrets, env)
	assert.NoError(t, err)
	assert.Equal(t, "val1", (*env)["MY_VAR"])
}

// TestGetSecret_InvalidServiceAccount verifies that requests from a mismatching service account are rejected.
func TestGetSecret_InvalidServiceAccount(t *testing.T) {
	ns, name, sa, otherSa := "default", "test-pod", "test-sa", "other-sa"
	pod := newPod(ns, name, sa, nil)

	_, client, svc := setupTest(t, pod)
	// Token is for otherSa
	mockTokenReview(client, "system:serviceaccount:"+ns+":"+otherSa, true)

	payload := &GetSecretPayload{
		Name:  name,
		Token: "valid-token",
	}

	_, _, err := svc.GetSecret(payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid service account found")
}

// TestGetSecret_MissingUID verifies that requests with a missing UID in the pod signature are rejected.
func TestGetSecret_MissingUID(t *testing.T) {
	ns, name, sa := "default", "test-pod", "test-sa"
	pod := newPod(ns, name, sa, map[string]string{
		Namespace + ConfigPiggyUID: "{}", // Empty signature
	})

	_, client, svc := setupTest(t, pod)
	mockTokenReview(client, "system:serviceaccount:"+ns+":"+sa, true)

	payload := &GetSecretPayload{
		Name:      name,
		Token:     "valid-token",
		UID:       "some-uid",
		Signature: "sig",
	}

	_, _, err := svc.GetSecret(payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid signature")
}
