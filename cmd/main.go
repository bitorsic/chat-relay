package main

import (
	"chat-relay/internal/backend"
	"chat-relay/internal/config"
	"log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	go func() {
		err := backend.StartBackend(cfg.BackendPort)
		if err != nil {
			log.Printf("Backend failed: %v", err)
		}
	}()

}
