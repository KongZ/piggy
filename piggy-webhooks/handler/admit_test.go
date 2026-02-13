package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KongZ/piggy/piggy-webhooks/mutate"
	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

// TestAdmitHandler_MethodNotAllowed verifies that non-POST requests are rejected.
func TestAdmitHandler_MethodNotAllowed(t *testing.T) {
	handler := AdmitHandler(func(req *admissionv1.AdmissionRequest) (interface{}, error) {
		return nil, nil
	})

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

// TestAdmitHandler_InvalidContentType ensures that requests with non-JSON content types are rejected.
func TestAdmitHandler_InvalidContentType(t *testing.T) {
	handler := AdmitHandler(func(req *admissionv1.AdmissionRequest) (interface{}, error) {
		return nil, nil
	})

	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// TestAdmitHandler_Success verifies the normal mutation flow through the HTTP handler.
func TestAdmitHandler_Success(t *testing.T) {
	// Mock admit function
	admit := func(req *admissionv1.AdmissionRequest) (interface{}, error) {
		// Return a simple patch or some object
		return map[string]interface{}{"metadata": map[string]interface{}{"labels": map[string]string{"piggy-injected": "true"}}}, nil
	}

	handler := AdmitHandler(admit)

	rawPod, _ := json.Marshal(metav1.ObjectMeta{Name: "test-pod"})
	review := admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			UID: "test-uid",
			Object: runtime.RawExtension{
				Raw: rawPod,
			},
		},
	}
	body, _ := json.Marshal(review)

	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", JSONContentType)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var responseReview admissionv1.AdmissionReview
	err := json.Unmarshal(rr.Body.Bytes(), &responseReview)
	assert.NoError(t, err)
	assert.True(t, responseReview.Response.Allowed)
	assert.NotNil(t, responseReview.Response.Patch)
}

// TestAdmitHandler_Idempotency ensures that multiple calls to the admission webhook
// do not result in duplicate resource injections in the pod template.
func TestAdmitHandler_Idempotency(t *testing.T) {
	// Initialize real Mutating object with a fake clientset
	ctx := context.Background()
	client := fake.NewClientset()
	m, _ := mutate.NewMutating(ctx, client)

	// Create a pod that requires mutation
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
			Annotations: map[string]string{
				"piggysec.com/aws-secret-name": "test-secret",
			},
		},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{},
			Containers: []corev1.Container{
				{
					Name: "app",
					Env: []corev1.EnvVar{
						{Name: "MY_SECRET", Value: "piggy:secret-key"},
					},
				},
			},
		},
	}

	rawPod, _ := json.Marshal(pod)
	review := admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			UID:      "test-uid",
			Resource: metav1.GroupVersionResource{Version: "v1", Resource: "pods"},
			Object: runtime.RawExtension{
				Raw: rawPod,
			},
			Namespace: "default",
		},
	}
	body, _ := json.Marshal(review)

	handler := AdmitHandler(m.ApplyPiggy)

	// First mutation
	req1, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBuffer(body))
	req1.Header.Set("Content-Type", JSONContentType)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	assert.Equal(t, http.StatusOK, rr1.Code)
	var responseReview1 admissionv1.AdmissionReview
	err := json.Unmarshal(rr1.Body.Bytes(), &responseReview1)
	assert.NoError(t, err)
	assert.True(t, responseReview1.Response.Allowed)

	// Apply the patch to get the mutated pod for the second call
	// For simplicity in unit test, we'll manually simulate the "mutated pod" being the input for the next call
	// as if reinvocationPolicy: IfNeeded triggered it again with the already mutated state.

	// We need to parse the patch and apply it, or just use m.ApplyPiggy directly on the mutated object
	// In a real K8s scenario, the second call's Request.Object is the result of the first call's mutation.

	// Let's decode the mutated pod from the first response if possible,
	// but AdmissionReview doesn't return the full object in the response, only the patch.

	// So we'll use a helper to apply the patch or just manually create the "expect-to-be-mutated" pod.
	// Actually, the easiest way to test idempotency here is to call m.ApplyPiggy(req) where req.Object is already mutated.

	mutatedPodInterface, err := m.ApplyPiggy(review.Request)
	assert.NoError(t, err)
	mutatedPod := mutatedPodInterface.(*corev1.Pod)

	// Verify first mutation added exactly 1 init container and 1 volume
	assert.Len(t, mutatedPod.Spec.InitContainers, 1)
	assert.Equal(t, "install-piggy-env", mutatedPod.Spec.InitContainers[0].Name)

	foundVolume := false
	for _, v := range mutatedPod.Spec.Volumes {
		if v.Name == "piggy-env" {
			foundVolume = true
			break
		}
	}
	assert.True(t, foundVolume)

	// Second mutation (simulating reinvocation)
	rawPod2, _ := json.Marshal(mutatedPod)
	review2 := review
	review2.Request.Object.Raw = rawPod2

	mutatedPodInterface2, err := m.ApplyPiggy(review2.Request)
	assert.NoError(t, err)
	mutatedPod2 := mutatedPodInterface2.(*corev1.Pod)

	// VERIFY IDEMPOTENCY: Should still have exactly 1 init container and 1 volume
	assert.Len(t, mutatedPod2.Spec.InitContainers, 1, "Init containers should not be duplicated")

	piggyVolumeCount := 0
	for _, v := range mutatedPod2.Spec.Volumes {
		if v.Name == "piggy-env" {
			piggyVolumeCount++
		}
	}
	assert.Equal(t, 1, piggyVolumeCount, "Piggy volume should not be duplicated")
}

// TestAdmitHandler_Errors verifies the admission handler's error response for various invalid requests.
func TestAdmitHandler_Errors(t *testing.T) {
	// Case 1: Malformed JSON
	handler := AdmitHandler(nil)
	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", JSONContentType)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	// Case 2: Missing request object
	review := admissionv1.AdmissionReview{Request: nil}
	body, _ := json.Marshal(review)
	req, _ = http.NewRequest(http.MethodPost, "/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", JSONContentType)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	// Case 3: Admit function error
	admitErr := func(req *admissionv1.AdmissionRequest) (interface{}, error) {
		return nil, errors.New("admit error")
	}
	handler = AdmitHandler(admitErr)
	rawPod, _ := json.Marshal(metav1.ObjectMeta{Name: "test-pod"})
	review = admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			UID: "test-uid",
			Object: runtime.RawExtension{
				Raw: rawPod,
			},
		},
	}
	body, _ = json.Marshal(review)
	req, _ = http.NewRequest(http.MethodPost, "/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", JSONContentType)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)

}
