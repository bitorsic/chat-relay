package backend

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type ChatRequest struct {
	UserID string `json:"user_id"`
	Query  string `json:"query"`
}

func StartBackend(port string) error {
	http.HandleFunc("/v1/chat/stream", handleChat)
	fmt.Println("[+] Listening on port", port)

	return http.ListenAndServe(":"+port, nil)
}

func handleChat(w http.ResponseWriter, r *http.Request) {
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
