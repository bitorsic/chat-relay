package backend

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type ChatRequest struct {
	UserID string `json:"user_id"`
	Query  string `json:"query"`
}

func NewBackendServer(port string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/stream", handleChat)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	return server
}

func StartBackend(server *http.Server) error {
	log.Println("Listening on port", server.Addr)

	return server.ListenAndServe()
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	response := map[string]string{
		"full_response": "Goroutines are lightweight threads. They enable high concurrency in Go. Channels help them communicate safely",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func StopBackend(server *http.Server, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Println("[!] Stopping backend...")
	return server.Shutdown(ctx)
}
