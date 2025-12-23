-- +goose Up
CREATE TABLE IF NOT EXISTS location_checks (
                                               id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                                               user_id      UUID              NOT NULL,
                                               lat          DOUBLE PRECISION  NOT NULL CHECK (lat >= -90  AND lat <= 90),
                                               lng          DOUBLE PRECISION  NOT NULL CHECK (lng >= -180 AND lng <= 180),
                                               incident_ids UUID[]            NOT NULL DEFAULT '{}',
                                               checked_at   TIMESTAMPTZ       NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS location_checks_user_time_idx
    ON location_checks (user_id, checked_at);

CREATE INDEX IF NOT EXISTS location_checks_time_idx
    ON location_checks (checked_at);

-- +goose Down
DROP INDEX IF EXISTS location_checks_time_idx;
DROP INDEX IF EXISTS location_checks_user_time_idx;
DROP TABLE IF EXISTS location_checks;
