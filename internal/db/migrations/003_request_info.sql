ALTER TABLE events
    ADD COLUMN IF NOT EXISTS request_info JSONB NOT NULL DEFAULT '{}';

CREATE INDEX IF NOT EXISTS idx_events_request_info_gin ON events USING GIN (request_info);
