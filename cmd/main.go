package main

import (
	"chat-relay/internal/backend"
	"chat-relay/internal/slack"
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
	backendServer := backend.NewBackendServer()

	go func() {
		err := backend.StartBackend(backendServer)
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Backend failed: %v", err)
		}
	}()

	go slack.StartSlackBot()

	// listening for SIGINT and performing graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	err := backend.StopBackend(backendServer, 5*time.Second)
	if err != nil {
		log.Printf("Backend shutdown failed: %v", err)
	}

	log.Println("Graceful Shutdown Successful")
}
