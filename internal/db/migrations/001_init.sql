-- Sources: registered producers of data (API clients, files, webhooks, etc.)
CREATE TABLE IF NOT EXISTS sources (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Batches: optional grouping for bulk imports or replay tracking
CREATE TABLE IF NOT EXISTS batches (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id   UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    status      TEXT NOT NULL DEFAULT 'open'
                CHECK (status IN ('open', 'closed', 'failed')),
    record_count INT NOT NULL DEFAULT 0,
    started_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ
);

-- Events: the core ingest table — one row per ingested record
CREATE TABLE IF NOT EXISTS events (
    id           BIGSERIAL PRIMARY KEY,
    source_id    UUID NOT NULL REFERENCES sources(id) ON DELETE RESTRICT,
    batch_id     UUID REFERENCES batches(id) ON DELETE SET NULL,
    event_type   TEXT NOT NULL DEFAULT 'record',
    external_id  TEXT,
    payload      JSONB NOT NULL,
    metadata     JSONB NOT NULL DEFAULT '{}',
    ingested_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT events_payload_is_object CHECK (jsonb_typeof(payload) = 'object')
);

CREATE INDEX IF NOT EXISTS idx_events_source_id ON events (source_id);
CREATE INDEX IF NOT EXISTS idx_events_batch_id ON events (batch_id) WHERE batch_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_events_ingested_at ON events (ingested_at DESC);
CREATE INDEX IF NOT EXISTS idx_events_event_type ON events (event_type);
CREATE INDEX IF NOT EXISTS idx_events_external_id ON events (source_id, external_id) WHERE external_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_events_payload_gin ON events USING GIN (payload);

INSERT INTO sources (name, description)
VALUES ('default', 'Default ingest source')
ON CONFLICT (name) DO NOTHING;
