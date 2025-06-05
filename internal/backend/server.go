package backend

import (
	"context"
	"encoding/json"
	"fmt"
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

var streamed bool

func StartBackend(server *http.Server, streamedResponse bool) error {
	streamed = streamedResponse
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
		}

		fmt.Fprintf(w, "id: %d\nevent: stream_end\ndata: {\"status\": \"done\"}\n\n", len(chunks)+1)
		flusher.Flush()

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
