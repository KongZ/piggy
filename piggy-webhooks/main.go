package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/KongZ/piggy/piggy-webhooks/handler"
	"github.com/KongZ/piggy/piggy-webhooks/mutate"
	"github.com/KongZ/piggy/piggy-webhooks/service"
	"k8s.io/client-go/kubernetes"
	kubernetesConfig "sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func newClient() (kubernetes.Interface, error) {
	kubeConfig, err := kubernetesConfig.GetConfig()
	if err != nil {
		return nil, err
	}
	k8sClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}
	return k8sClient, nil
}

func main() {
	var err error
	debug, _ := strconv.ParseBool(service.GetEnv("DEBUG", "false"))
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	certPath := service.GetEnv("TLS_CERT_FILE", "")
	keyPath := service.GetEnv("TLS_PRIVATE_KEY_FILE", "")
	listenAddress := service.GetEnv("LISTEN_ADDRESS", ":8080")
	k8s, err := newClient()
	if err != nil {
		log.Fatal().Msgf("error creating client: %s", err)
	}
	mut, err := mutate.NewMutating(context.Background(), k8s)
	if err != nil {
		log.Fatal().Msgf("error creating webhook: %s", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	mux.Handle("/mutate", handler.AdmitHandler(mut.ApplyPiggy))
	svc, err := service.NewService(context.Background(), k8s)
	if err != nil {
		log.Fatal().Msgf("error creating service: %s", err)
	}
	mux.Handle("/secret", handler.SecretHandler(svc.GetSecret))
	ch := make(chan struct{})
	server := http.Server{Addr: listenAddress, Handler: mux}
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGTERM)
		<-sigint
		// We received an interrupt signal, shut down.
		if err := server.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Error().Msgf("HTTP server Shutdown: %v", err)
		}
		close(ch)
	}()
	if certPath == "" && keyPath == "" {
		log.Info().Msgf("Listening on http://%s", listenAddress)
		err = server.ListenAndServe()
	} else {
		log.Info().Msgf("Listening on https://%s", listenAddress)
		err = server.ListenAndServeTLS(certPath, keyPath)
	}
	if err != nil {
		log.Fatal().Msgf("Error serving webhooks: %s", err)
	}
}
