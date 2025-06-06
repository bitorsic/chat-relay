package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type ChatRequest struct {
	UserID string `json:"user_id"`
	Query  string `json:"query"`
}

func NewBackendServer() *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/stream", handleChat)

	port := os.Getenv("BACKEND_PORT")
	if port == "" {
		port = "3000"
	}

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

	var streamed bool
	streamedString := os.Getenv("STREAMED_RESPONSE")
	if streamedString == "true" {
		streamed = true
	} else {
		streamed = false
	}

	var req ChatRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if streamed {
		chunks := []string{
			"Goroutines are lightweight threads. ",
			"They enable high concurrency in Go. ",
			"Channels help them communicate safely",
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		for i, chunk := range chunks {
			fmt.Fprintf(w, "id: %d\nevent: message_part\ndata: {\"text_chunk\": \"%s\"}\n\n", i+1, chunk)
			flusher.Flush()
			// time.Sleep(2 * time.Second)
		}

		fmt.Fprintf(w, "id: %d\nevent: stream_end\ndata: {\"status\": \"done\"}\n\n", len(chunks)+1)
		flusher.Flush()
		// time.Sleep(2 * time.Second)

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
