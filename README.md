# Ingest

HTTP service that accepts JSON records and stores them in PostgreSQL. Includes a **Discord webhook-compatible** API so any tool that can POST to a Discord webhook URL works unchanged.

## Quick start

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
| `GET` | `/api/webhooks/{id}/{token}` | Webhook metadata |
| `PATCH` | `/api/webhooks/{id}/{token}/messages/{message.id}` | Update stored message |
| `DELETE` | `/api/webhooks/{id}/{token}/messages/{message.id}` | Delete stored message |

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
| `events` | One row per ingested record (`payload` JSONB) |
