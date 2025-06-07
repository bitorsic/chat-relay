package backend

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleChatStreamed(t *testing.T) {
	t.Setenv("STREAMED_RESPONSE", "true")

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

	if !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("expected Content-Type text/event-stream, received %v", response.Header.Get("Content-Type"))
	}
}

func TestHandleChatFull(t *testing.T) {
	t.Setenv("STREAMED_RESPONSE", "false")

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

	if !strings.HasPrefix(response.Header.Get("Content-Type"), "application/json") {
		t.Fatalf("expected Content-Type application/json, received %v", response.Header.Get("Content-Type"))
	}

	var data map[string]string

	err := json.NewDecoder(response.Body).Decode(&data)
	if err != nil {
		t.Fatalf("received invalid JSON")
	}

	_, ok := data["full_response"]
	if !ok {
		t.Fatalf("received JSON does not have the \"full_response\" field")
	}
}
