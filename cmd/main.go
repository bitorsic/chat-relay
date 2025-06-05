package main

import (
	"chat-relay/internal/backend"
	"chat-relay/internal/config"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func init() {
	godotenv.Load()
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	backendServer := backend.NewBackendServer(cfg.BackendPort)

	go func() {
		err := backend.StartBackend(backendServer)
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Backend failed: %v", err)
		}
	}()

	// listening for SIGINT and performing graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	err = backend.StopBackend(backendServer, 5*time.Second)
	if err != nil {
		log.Printf("Backend shutdown failed: %v", err)
	}

	log.Println("Graceful Shutdown Successful")
}
