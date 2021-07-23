package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	listenAddress := ":8080"
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		for _, env := range os.Environ() {
			w.Write([]byte(env))
			w.Write([]byte("\n"))
		}
	}))
	ch := make(chan struct{})
	server := http.Server{Addr: listenAddress, Handler: mux}
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGTERM)
		<-sigint
		// We received an interrupt signal, shut down.
		if err := server.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("HTTP server Shutdown: %v", err)
		}
		close(ch)
	}()
	if err := server.ListenAndServe(); err != nil {
		log.Printf("Error serving: %s", err)
	}
}
