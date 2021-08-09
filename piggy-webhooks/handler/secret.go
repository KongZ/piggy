package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/KongZ/piggy/piggy-webhooks/service"
	"github.com/rs/zerolog/log"
)

type getSecretFunc func(*service.GetSecretPayload) (*service.SanitizedEnv, error)

func doServeSecretFunc(w http.ResponseWriter, r *http.Request, secretFunc getSecretFunc) ([]byte, error) {
	// Step 1: Request validation. Only handle POST requests with a body and json content type.
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return nil, fmt.Errorf("invalid method %s, only POST requests are allowed", r.Method)
	}
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("only 'application/x-www-form-urlencoded' is supported")
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("could not read request body: %v", err)
	}

	if contentType := r.Header.Get("Content-Type"); contentType != JsonContentType {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("unsupported content type %s, only %s is supported", contentType, JsonContentType)
	}

	serviceToken := r.Header.Get("X-Token")
	if len(serviceToken) == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		return nil, fmt.Errorf("token is not supplied: %v", err)
	}

	// Step 2: Parse the request.
	var payload service.GetSecretPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("could not deserialize request: %v", err)
	} else if payload.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("malformed payload: request is nil")
	}
	payload.Token = serviceToken
	// Serve request
	env, err := secretFunc(&payload)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("could not get secret: %v", err)
	}

	// Return the secrets with a response as JSON.
	bytes, err := json.Marshal(&env)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return nil, fmt.Errorf("marshaling response: %v", err)
	}
	return bytes, nil
}

// SecretHandler retreive and return secret from secret manager
func SecretHandler(secret getSecretFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// dump, err := httputil.DumpRequest(r, true)
		// if err != nil {
		// 	log.Error().Msgf("%v", err)
		// 	return
		// }
		// log.Info().Msgf("%q", dump)
		log.Debug().Msgf("Handling secret request ...")

		var writeErr error
		if bytes, err := doServeSecretFunc(w, r, secret); err == nil {
			log.Debug().Msgf("Secret request handled successfully")
			_, writeErr = w.Write(bytes)
		} else {
			log.Error().Msgf("Error handling secret request: %v", err)
			_, writeErr = w.Write([]byte(err.Error()))
		}
		if writeErr != nil {
			log.Error().Msgf("Could not write response: %v", writeErr)
		}
	})
}
