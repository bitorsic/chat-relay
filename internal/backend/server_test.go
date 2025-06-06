package backend

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleChat(t *testing.T) {
	reqBody := ChatRequest{
		UserID: "abc",
		Query:  "What are goroutines?",
	}

	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/stream", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleChat(w, req)

	response := w.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status code 200, received %d", response.StatusCode)
	}
}
