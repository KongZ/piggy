package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KongZ/piggy/piggy-webhooks/service"
	"github.com/stretchr/testify/assert"
)

// TestSecretHandler_Success verifies that the secret handler correctly processes
// valid requests and returns the expected environment variables.
func TestSecretHandler_Success(t *testing.T) {
	// Mock secret mapping function
	secretFunc := func(payload *service.GetSecretPayload) (*service.SanitizedEnv, service.Info, error) {
		env := &service.SanitizedEnv{
			"DB_PASS": "secret-value",
		}
		info := service.Info{
			Namespace:      "default",
			Name:           "test-pod",
			ServiceAccount: "test-sa",
		}
		return env, info, nil
	}

	handler := SecretHandler(secretFunc)

	payload := service.GetSecretPayload{
		Name: "test-pod",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", JSONContentType)
	req.Header.Set("X-Token", "valid-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var responseEnv service.SanitizedEnv
	err := json.Unmarshal(rr.Body.Bytes(), &responseEnv)
	assert.NoError(t, err)
	assert.Equal(t, "secret-value", responseEnv["DB_PASS"])
}

// TestSecretHandler_MissingToken ensures that requests lacking an authorization token
// are rejected with an unauthorized status.
func TestSecretHandler_MissingToken(t *testing.T) {
	handler := SecretHandler(nil)

	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", JSONContentType)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestSecretHandler_Errors verifies the secret handler's error response for various invalid requests.
func TestSecretHandler_Errors(t *testing.T) {
	handler := SecretHandler(nil)

	// Case 1: Invalid Method
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)

	// Case 2: Invalid Content-Type
	req, _ = http.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-Token", "valid-token")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	// Case 3: Malformed JSON
	req, _ = http.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", JSONContentType)
	req.Header.Set("X-Token", "valid-token")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	// Case 4: Forbidden
	secretFuncForbidden := func(payload *service.GetSecretPayload) (*service.SanitizedEnv, service.Info, error) {
		return nil, service.Info{}, service.ErrorAuthorized
	}
	handler = SecretHandler(secretFuncForbidden)
	req, _ = http.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"name":"pod"}`))
	req.Header.Set("Content-Type", JSONContentType)
	req.Header.Set("X-Token", "valid-token")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)

	// Case 5: Generic Error
	secretFuncError := func(payload *service.GetSecretPayload) (*service.SanitizedEnv, service.Info, error) {
		return nil, service.Info{}, assert.AnError
	}
	handler = SecretHandler(secretFuncError)
	req, _ = http.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"name":"pod"}`))
	req.Header.Set("Content-Type", JSONContentType)
	req.Header.Set("X-Token", "valid-token")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	// Case 6: Empty Name
	handler = SecretHandler(nil)
	req, _ = http.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"name":""}`))
	req.Header.Set("Content-Type", JSONContentType)
	req.Header.Set("X-Token", "valid-token")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
