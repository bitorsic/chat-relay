
# ChatRelay Slack Bot

ChatRelay is a high-performance Golang Slack bot that listens to user messages (via direct messages or mentions), forwards them to a backend, and streams the response back to Slack. It includes strong observability via OpenTelemetry, is built with concurrency in mind, and is easily extensible.

----------

## 📅 Project Overview

-   **Slack Bot:** Responds to @mentions and DMs.
    
-   **Chat Backend:** Simulated server with SSE or complete JSON response.
    
-   **Observability:** Full lifecycle traced with OpenTelemetry.
    
-   **Concurrency:** Goroutines used to handle Slack events, backend communication, and streaming.
    

----------

## ⚖️ Design Decisions

### Slack Connection: Socket Mode

-   **Why:** Enables local development without exposing public URLs.
    
-   **How:** Uses `slack-go/slack` and `socketmode`.
    

### Backend Streaming

-   SSE-based simulated streaming using `text/event-stream`.
    
-   Alternatively, responds with a full JSON payload.
    

### Concurrency

-   Slack bot runs in its own goroutine.
    
-   Backend server runs concurrently.
    
-   Response processing is asynchronous using goroutines and channels.
    

### Observability (OpenTelemetry)

-   Context propagated across HTTP boundary.
    
-   Spans: Receiving Slack event, sending to backend, processing response.
    
-   Exporter: Console-based `stdouttrace`.
    

----------

## 📁 Project Structure

```
cmd/main.go                  # Entry point, starts backend and Slack bot
defer graceful shutdown
internal/backend/server.go   # Mock backend server with SSE/JSON response
internal/backend/server_test.go # Basic unit test
otel/otel.go                 # OpenTelemetry setup
slack/slack.go               # Slack bot logic and response handler
.env.example                 # Example environment variables
go.mod / go.sum              # Dependencies
Dockerfile                   # Container config
```

----------

## 🌐 Setup and Running Instructions

### 1. Clone and Configure Environment

```
git clone <your-repo-url>
cd chat-relay
cp .env.example .env
```

### 2. Environment Variables

```
SLACK_APP_TOKEN=xapp-...
SLACK_BOT_TOKEN=xoxb-...
BACKEND_PORT=3000
BACKEND_URL=http://localhost:3000/
STREAMED_RESPONSE=true  # or false
# Optional for OTEL exporter (if used):
# OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
```

### 3. Slack App Setup ([https://api.slack.com/apps](https://api.slack.com/apps))

-   **Event Subscriptions:**
    
    -   Enable events and set `Socket Mode` ON
        
    -   Subscribe to events: `app_mention`, `message.im`
        
-   **Scopes Required:**
    
    -   `app_mentions:read`
        
    -   `chat:write`
        
    -   `channels:history`
        
    -   `groups:history`
        
    -   `im:history`
        
-   **Tokens:**
    
    -   Generate **Bot Token (xoxb)** and **App-Level Token (xapp)**
        

### 4. Run the Bot

```
go run cmd/main.go
```

### 5. Interacting With the Bot

-   **Mention in channel:**  `@ChatRelay tell me about goroutines`
    
-   **Direct message:**  `What is Go concurrency?`
    

----------

## 🔁 Mock Backend Behavior

-   Endpoint: `POST /v1/chat/stream`
    
-   Request format:
    
    ```
    {
      "user_id": "U12345",
      "query": "Tell me about goroutines"
    }
    ```
    
-   Behavior:
    
    -   If `STREAMED_RESPONSE=true`, replies with **SSE stream**:
        
        ```
        id: 1
        event: message_part
        data: {"text_chunk": "concurrent execution units in Go. "}
        
        id: 2
        event: message_part
        data: {"text_chunk": "They allow for massive parallelism."}
        
        id: 3
        event: stream_end
        data: {"status": "done"}
        ```
        
    -   Else, replies with full JSON:
        
        ```
        {
          "full_response": "Goroutines are lightweight..."
        }
        ```
        

----------

## 🐳 Dockerization

### 1. Build and Run Locally

```
docker build -t chatrelay .

docker run --env-file .env -p 8080:8080 chatrelay
```

### 2. **Docker Compose**

```
docker-compose up --build
```

----------

## ⚖️ Observability: Tracing & Logging

### Console Export (dev mode)

-   Traces printed using `stdouttrace.WithPrettyPrint()`.
    
-   Span context includes:
    
    -   Event type (mention/dm)
        
    -   Backend request/response lifecycle
        
    -   Response format (SSE or JSON)
        

### Context Propagation

-   Injects OTEL trace context in headers to backend.
    
-   Trace IDs logged with `log.Printf` for correlation.
    

### Export to Collector (optional)

To export telemetry data to Jaeger, OTLP, or a collector:

```
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
```

Then update `otel/otel.go` to use OTLP exporter.

----------

## 🚀 Scalability, Performance, and Observability

### Concurrency

-   Goroutines handle Slack events and backend streaming.
    
-   Non-blocking Slack updates via message edits.
    

### Resource Management

-   Lightweight HTTP server and Slack client.
    
-   Lazy response buffering.
    

### Bottlenecks & Mitigations

Bottleneck

Solution

Slack API Rate Limits

Message updates instead of floods

Backend Latency

Async SSE chunking with flushing

Crash Resilience

Graceful shutdown, error handling

### Horizontal Scalability

-   Stateless design: Multiple bot instances can be run in parallel.
    
-   Load balancer or Slack events can be distributed per bot instance.
    

### Stability

-   `signal.Notify` handles SIGTERM/SIGINT
    
-   OpenTelemetry spans flushed on shutdown
    

----------

## 📊 Testing

### Unit Tests

```
go test ./...
```

-   `server_test.go` includes:
    
    -   Test for SSE response
        
    -   Test for JSON response
        
-   `slack_test.go` includes:
    
    -   Integration-style test for `handleSlackResponse`, verifying backend call is made and execution completes without errors
        

These tests ensure core functionality is verified across the backend and Slack interaction boundaries.

For coverage:

```
go test -cover ./...
```

----------

## 📬 Slack Marketplace Publication Plan

### Technical & Procedural Steps

-   Implement **OAuth 2.0 Add to Slack** flow
    
-   Create **public redirect endpoint** (via ngrok or deployed service)
    
-   Secure token storage and environment handling
    
-   Validate all inputs from Slack
    
-   Complete **App Manifest** and submit for review
    

### Additional Requirements

-   Privacy Policy & Terms of Use
    
-   Support channel / contact email
    
-   Clear branding and command documentation
    

----------

## 🚀 Future Improvements

-   Structured logging with trace context
    
-   Slack interaction payload support
    
-   Retry policies for flaky backend responses
    
-   Metrics exporter (Prometheus, OTLP)
    

----------

Built with Go, Slack API, and OpenTelemetry ❤️
