package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"io"

	"github.com/KongZ/piggy/piggy-webhooks/mutate"
	"github.com/rs/zerolog/log"
	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type admitFunc func(*admissionv1.AdmissionRequest) (interface{}, error)

func doServeAdmitFunc(w http.ResponseWriter, r *http.Request, admit admitFunc) ([]byte, error) {
	// Step 1: Request validation. Only handle POST requests with a body and json content type.
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return nil, fmt.Errorf("invalid method %s, only POST requests are allowed", r.Method)
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("could not read request body: %v", err)
	}

	if contentType := r.Header.Get("Content-Type"); contentType != JsonContentType {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("unsupported content type %s, only %s is supported", contentType, JsonContentType)
	}

	// Step 2: Parse the AdmissionReview request.
	var admissionReviewReq admissionv1.AdmissionReview
	if _, _, err := mutate.UniversalDeserializer.Decode(body, nil, &admissionReviewReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("could not deserialize request: %v", err)
	} else if admissionReviewReq.Request == nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, errors.New("malformed admission review: request is nil")
	}

	// Step 3: Construct the AdmissionReview response.
	pt := admissionv1.PatchTypeJSONPatch
	admissionReviewResponse := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{APIVersion: "admission.k8s.io/v1", Kind: "AdmissionReview"},
		Response: &admissionv1.AdmissionResponse{
			UID:       admissionReviewReq.Request.UID,
			PatchType: &pt,
		},
	}

	// // Apply the admit() function only for non-Kubernetes namespaces. For objects in Kubernetes namespaces, return
	// // an empty set of patch operations.
	// if common.IsKubeNamespace(admissionReviewReq.Request.Namespace) {
	// 	return nil, nil
	// }

	mutatedObj, err := admit(admissionReviewReq.Request)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return nil, fmt.Errorf("error while admitting request: %w", err)
	}
	if mutatedObj == nil {
		log.Debug().Msgf("Nothing to mutate")
		return nil, nil
	}

	reqObject := admissionReviewReq.Request.Object

	mutatedJSON, err := json.Marshal(mutatedObj)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return nil, fmt.Errorf("could not marshal into JSON mutated object: %w", err)
	}
	patch, err := jsonpatch.CreatePatch(reqObject.Raw, mutatedJSON)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return nil, fmt.Errorf("could not create JSON patch: %w", err)
	}

	if err == nil {
		// Encode the patch operations to JSON and return a positive response.
		patchBytes, err := json.Marshal(patch)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return nil, fmt.Errorf("could not marshal JSON patch: %v", err)
		}
		admissionReviewResponse.Response.Allowed = true
		admissionReviewResponse.Response.Patch = patchBytes
	} else {
		// If the handler returned an error, incorporate the error message into the response and deny the object
		// creation.
		admissionReviewResponse.Response.Allowed = false
		admissionReviewResponse.Response.Result = &metav1.Status{
			Message: err.Error(),
		}
	}

	// Return the AdmissionReview with a response as JSON.
	bytes, err := json.Marshal(&admissionReviewResponse)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return nil, fmt.Errorf("marshaling response: %v", err)
	}
	return bytes, nil
}

// AdmitFuncHandler takes an admitFunc and wraps it into a http.Handler by means of calling serveAdmitFunc.
func AdmitHandler(admit admitFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// dump, err := httputil.DumpRequest(r, true)
		// if err != nil {
		// 	log.Error().Msgf("%v", err)
		// 	return
		// }
		// log.Info().Msgf("%q", dump)
		log.Debug().Msgf("Handling webhook request ...")

		var writeErr error
		if bytes, err := doServeAdmitFunc(w, r, admit); err == nil {
			log.Debug().Msgf("Webhook request handled successfully")
			_, writeErr = w.Write(bytes)
		} else {
			log.Error().Msgf("Error handling webhook request: %v", err)
			_, writeErr = w.Write([]byte(err.Error()))
		}

		if writeErr != nil {
			log.Error().Msgf("Could not write response: %v", writeErr)
		}
	})
}
