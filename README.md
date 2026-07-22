# ChatRelay Slack Bot

ChatRelay is a Go Slack bot that listens for direct messages and @mentions, forwards them to a Gemini-backed backend, and returns either a streamed SSE response or a full JSON response back to Slack. The backend, Slack client, and tracing are all started by the same Go process.

## What It Does

- Responds to Slack `app_mention` events in channels and `message.im` events in DMs.
- Calls a local HTTP backend at `POST /v1/chat/stream`.
- Uses Google Gemini (`gemini-1.5-flash`) for both streamed and non-streamed responses.
- Updates the Slack message while streamed chunks arrive, or posts the full response directly.
- Emits OpenTelemetry traces to stdout with pretty printing.

## Runtime Layout

The application starts two things:

- A Slack Socket Mode client for event handling.
- An HTTP backend server that listens on `BACKEND_PORT` and serves `/v1/chat/stream`.

The Slack side sends the user message to the backend with JSON like:

```json
{
  "user_id": "U12345",
  "query": "Tell me about goroutines"
}
```

The backend returns:

- `text/event-stream` when `STREAMED_RESPONSE=true`
- `application/json` when `STREAMED_RESPONSE` is anything else or unset

## Project Structure

```text
cmd/main.go                     # Entry point: starts tracing, Gemini, backend, and Slack bot
internal/backend/server.go      # HTTP handler for /v1/chat/stream
internal/backend/genai.go       # Gemini helpers for stream and full response modes
internal/backend/server_test.go # Backend tests for streamed and JSON responses
internal/config/genai.go        # Gemini client and generation config setup
internal/slack/slack.go         # Slack Socket Mode client and response handling
internal/slack/slack_test.go    # Slack handler test
internal/otel/otel.go           # stdout OpenTelemetry tracer setup
Dockerfile                      # Container image build
docker-compose.yml              # Local compose setup
go.mod                          # Go module definition
```

## Requirements

- Go 1.24 or newer
- A Google Gemini API key
- A Slack app configured for Socket Mode
- Docker, if you want to run the container image

## Configuration

Create a `.env` file in the repository root with the following values:

```bash
SLACK_APP_TOKEN=xapp-...
SLACK_BOT_TOKEN=xoxb-...
GEMINI_API_KEY=your-gemini-api-key
BACKEND_PORT=3000
BACKEND_URL=http://localhost:3000/
STREAMED_RESPONSE=true
```

Notes:

- `BACKEND_URL` should end with a trailing slash because the code appends `v1/chat/stream` directly.
- `BACKEND_PORT` defaults to `3000` if it is not set.
- `STREAMED_RESPONSE` defaults to JSON mode unless it is exactly `true`.

## Slack App Setup

1. Create or open your app at [Slack API](https://api.slack.com/apps).
2. Enable Socket Mode.
3. Enable Event Subscriptions.
4. Subscribe to `app_mention` and `message.im`.
5. Add these bot token scopes:
   - `app_mentions:read`
   - `chat:write`
   - `im:history`
   - `channels:history` if you want channel mentions
   - `groups:history` if you want private channel mentions
6. Generate both tokens:
   - Bot token: `xoxb-...`
   - App-level token: `xapp-...`

## Run Locally

1. Clone the repository.
2. Create your `.env` file with the values above.
3. Start the app:

```bash
go run ./cmd
```

4. Send a DM to the bot or mention it in a channel.

Example prompts:

- `@ChatRelay tell me about goroutines`
- `What is Go concurrency?`

## Backend Behavior

The backend exposes a single endpoint:

- `POST /v1/chat/stream`

When `STREAMED_RESPONSE=true`, the response is SSE with events in this shape:

```text
id: 1
event: message_part
data: {"text_chunk":"..."}

id: 2
event: stream_end
data: {"status":"done"}
```

When streaming is disabled, the backend returns:

```json
{
  "full_response": "..."
}
```

## Docker

Build the image:

```bash
docker build -t chatrelay .
```

Run the container on the backend port used by the app:

```bash
docker run --env-file .env -p 3000:3000 chatrelay
```

Use Compose for local development:

```bash
docker-compose up --build
```

If you change `BACKEND_PORT`, update the port mapping to match.

## Observability

- Traces are exported to stdout with `stdouttrace.WithPrettyPrint()`.
- Slack events, backend requests, and response handling are traced.
- The Slack client injects trace context into the backend request headers.
- Shutdown is handled on `SIGINT` and `SIGTERM`.

## Testing

Run the test suite with:

```bash
go test ./...
```

Coverage can be collected with:

```bash
go test -cover ./...
```

## Notes

- The backend uses Gemini through the `google.golang.org/genai` client.
- The current OpenTelemetry setup uses the stdout exporter; OTLP export is not wired in.

Built with Go, Slack API, Gemini AI, and OpenTelemetry.
