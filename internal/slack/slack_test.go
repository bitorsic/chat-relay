package slack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// just testing if the backend gets called
func TestHandleSlackResponse_JSON(t *testing.T) {
	backendCalled := false

	handler := func(w http.ResponseWriter, r *http.Request) {
		backendCalled = true
		resp := map[string]string{"full_response": "Hello from backend"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	os.Setenv("BACKEND_URL", server.URL+"/")

	api := slack.New("xoxb-fake-token", slack.OptionLog(nil), slack.OptionDebug(false))
	client := socketmode.New(api)

	handleSlackResponse(context.Background(), "test-channel", "U123", "hello", (*socketmode.Client)(client))

	if !backendCalled {
		t.Error("Expected backend to be called, but it wasn't")
	}
}
