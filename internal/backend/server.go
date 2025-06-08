package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
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
	// extracting the context from headers for otel tracing
	propagator := propagation.TraceContext{}
	ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

	// starting the tracer and the new span
	tracer := otel.Tracer("mock-backend")
	ctx, span := tracer.Start(ctx, "ProcessingChatRequest")
	defer span.End()

	if r.Method != http.MethodPost {
		span.SetStatus(codes.Error, "Could not connect to the backend")

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

	span.SetAttributes(attribute.Bool("streamed", streamed))

	var req ChatRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		span.SetStatus(codes.Error, "Invalid HTTP request")
		span.RecordError(err)

		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if streamed {
		aiStream := getGenAIStream(ctx, req.Query)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")

		flusher, ok := w.(http.Flusher)
		if !ok {
			span.SetStatus(codes.Error, "Streaming not supported")

			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		var chunkID int = 0

		// for i, chunk := range chunks {
		for chunk, err := range aiStream {
			if err != nil {
				span.SetStatus(codes.Error, "Could not generate AI Response")
				span.RecordError(err)

				http.Error(w, "could not generate response properly", http.StatusInternalServerError)
				return
			}
			chunkID++

			part := chunk.Candidates[0].Content.Parts[0]
			textChunk := part.Text

			// constructing JSON this way to avoid parsing errors
			payload := map[string]string{
				"text_chunk": textChunk,
			}

			jsonBytes, err := json.Marshal(payload)
			if err != nil {
				span.SetStatus(codes.Error, "Could not construct JSON properly")
				span.RecordError(err)

				http.Error(w, "could not construct json properly", http.StatusInternalServerError)
				return
			}

			fmt.Fprintf(w, "id: %d\nevent: message_part\ndata: %s\n\n", chunkID, string(jsonBytes))
			flusher.Flush()
			// time.Sleep(2 * time.Second)

			span.AddEvent("SentTextChunk", trace.WithAttributes(
				attribute.Int("chunk_id", chunkID),
				attribute.String("event", "message_part"),
			))
		}

		span.SetAttributes(attribute.Int("chunks_count", chunkID))

		chunkID++

		fmt.Fprintf(w, "id: %d\nevent: stream_end\ndata: {\"status\": \"done\"}\n\n", chunkID)
		flusher.Flush()
		// time.Sleep(2 * time.Second)

		span.AddEvent("SentStatusChunk", trace.WithAttributes(
			attribute.Int("chunk_id", chunkID),
			attribute.String("event", "stream_end"),
			attribute.String("status", "done"),
		))

		return
	}

	aiResponse, err := getGenAIFullResponse(ctx, req.Query)
	if err != nil {
		span.SetStatus(codes.Error, "Could not generate AI Response")
		span.RecordError(err)

		http.Error(w, "could not generate response properly", http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"full_response": aiResponse,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	span.AddEvent("SentFullJSONResponse")
}

func StopBackend(server *http.Server, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Println("[!] Stopping backend...")
	return server.Shutdown(ctx)
}
