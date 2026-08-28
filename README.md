# Ingest

HTTP service that accepts JSON records and stores them in PostgreSQL. Includes a **Discord webhook-compatible** API so any tool that can POST to a Discord webhook URL works unchanged.

## Quick start

```bash
make deploy
```

Starts Postgres and the ingest app in Docker. No manual steps.

Alternative for local Go development (Postgres in Docker, app on host):

```bash
make dev
```

## Manual setup

```bash
docker compose up -d
cp .env.example .env
go run ./cmd/ingest
```

## Discord webhook ingest (recommended)

This mirrors Discord's execute-webhook API:

| Method | Path | Behavior |
|--------|------|----------|
| `POST` | `/api/webhooks/{id}/{token}` | Ingest message (returns `204`, or `200` + message with `?wait=true`) |

Only valid requests are stored. Invalid payloads, unknown webhooks, and empty messages return an error with no database write.

Each stored event includes a `request_info` JSON column with method, path, query params, headers, remote address, and user agent.

### 1. Create a webhook

```bash
curl -X POST http://localhost:8080/v1/webhooks \
  -H 'Content-Type: application/json' \
  -d '{"name": "Alerts", "source": "webhook"}'
```

Response:

```json
{
  "id": "223704706495545344",
  "token": "3d89bb7572e0fb30...",
  "name": "Alerts",
  "channel_id": "199737254929760256",
  "url": "/api/webhooks/223704706495545344/3d89bb7572e0fb30...",
  "source": "webhook"
}
```

### 2. Send data (Discord-compatible)

```bash
# Default: 204 No Content (same as Discord)
curl -X POST "http://localhost:8080/api/webhooks/{id}/{token}" \
  -H 'Content-Type: application/json' \
  -d '{"content": "Server is up", "embeds": [{"title": "Status", "description": "All good"}]}'

# With ?wait=true: returns the created message object (same as Discord)
curl -X POST "http://localhost:8080/api/webhooks/{id}/{token}?wait=true" \
  -H 'Content-Type: application/json' \
  -d '{"content": "Hello from webhook"}'
```

Supported body fields: `content`, `embeds`, `username`, `avatar_url`, `tts`, `allowed_mentions`, `components`, `flags`, `thread_name`, `attachments`, `poll`. At least one of `content`, `embeds`, `components`, `file`, or `poll` is required — matching Discord's rules.

`multipart/form-data` with `payload_json` + file fields is also accepted (file metadata is stored; binary content is not persisted).

Point any Discord webhook client at your server instead of `discord.com`:

```
https://your-host/api/webhooks/{id}/{token}
```

## Realtime event stream

Subscribe to new events as they are ingested via **Server-Sent Events (SSE)**:

```bash
curl -N http://localhost:8080/v1/events/stream
```

Optional filters:

```bash
curl -N "http://localhost:8080/v1/events/stream?source=webhook&event_type=discord.webhook"
```

**Browser / Node client:**

```javascript
const es = new EventSource("http://localhost:8080/v1/events/stream?source=webhook");

es.addEventListener("message", (e) => {
  const event = JSON.parse(e.data);
  console.log("new event", event.id, event.payload);
});

es.onerror = () => console.error("stream disconnected");
```

Each `message` event contains the full row: `id`, `source`, `event_type`, `payload`, `request_info`, `ingested_at`, etc.

Under the hood Postgres `NOTIFY` fires on every insert, and the server pushes it to connected clients with minimal delay.

## Payload encryption

Payloads are encrypted at ingest time with a **hybrid RSA + AES-GCM** envelope before being written to Postgres. Only your local private key can decrypt them.

### 1. Generate keys (local, one-time)

```bash
go run ./cmd/keygen
```

This creates:

- `keys/private.pem` — **keep on your machine, never deploy**
- `keys/public.pem` — deploy to the server

### 2. Configure the server

```bash
export INGEST_PUBLIC_KEY_FILE=keys/public.pem
# or paste PEM into INGEST_PUBLIC_KEY
go run ./cmd/ingest
```

On Railway, set `INGEST_PUBLIC_KEY` to the contents of `public.pem`.

### 3. Decrypt locally

Pipe an encrypted `payload` from the DB or SSE stream:

```bash
curl -N http://localhost:8080/v1/events/stream | while read -r line; do
  echo "$line"
done

# decrypt an envelope JSON file:
cat envelope.json | go run ./cmd/decrypt -key keys/private.pem
```

Stored envelope format:

```json
{"v":1,"ek":"...","n":"...","c":"..."}
```

## Generic ingest API

### Health

```bash
curl http://localhost:8080/health
```

### Ingest a record

```bash
curl -X POST http://localhost:8080/v1/ingest \
  -H 'Content-Type: application/json' \
  -d '{
    "source": "default",
    "event_type": "user.signup",
    "external_id": "usr_123",
    "payload": {"email": "alice@example.com", "plan": "pro"},
    "metadata": {"ip": "10.0.0.1"}
  }'
```

## Schema

See `internal/db/migrations/` for full DDL.

| Table | Purpose |
|-------|---------|
| `sources` | Named producers (API, webhook, file import) |
| `webhooks` | Discord-style webhook credentials (`id` + `token`) |
| `batches` | Optional grouping for bulk/replay tracking |
| `events` | One row per valid ingested record (`payload`, `request_info` JSONB) |
