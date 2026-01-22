package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestAdmitHandler_MethodNotAllowed(t *testing.T) {
	handler := AdmitHandler(func(req *admissionv1.AdmissionRequest) (interface{}, error) {
		return nil, nil
	})

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

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
	req.Header.Set("Content-Type", JsonContentType)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	
	var responseReview admissionv1.AdmissionReview
	err := json.Unmarshal(rr.Body.Bytes(), &responseReview)
	assert.NoError(t, err)
	assert.True(t, responseReview.Response.Allowed)
	assert.NotNil(t, responseReview.Response.Patch)
}
