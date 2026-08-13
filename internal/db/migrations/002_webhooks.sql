CREATE TABLE IF NOT EXISTS webhooks (
    id          TEXT PRIMARY KEY,
    token       TEXT NOT NULL,
    source_id   UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    name        TEXT NOT NULL DEFAULT 'Captain Hook',
    avatar      TEXT,
    channel_id  TEXT NOT NULL,
    guild_id    TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT webhooks_id_token_unique UNIQUE (id, token)
);

CREATE INDEX IF NOT EXISTS idx_webhooks_source_id ON webhooks (source_id);
