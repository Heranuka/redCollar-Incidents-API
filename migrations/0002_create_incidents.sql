-- +goose Up
CREATE TABLE IF NOT EXISTS incidents (
                                         id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                                         geo_point  GEOGRAPHY(POINT,4326) NOT NULL,
                                         radius_km  DOUBLE PRECISION      NOT NULL CHECK (radius_km > 0 AND radius_km <= 100),
                                         status     VARCHAR(20)           NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
                                         created_at TIMESTAMPTZ           NOT NULL DEFAULT NOW()
);

-- Индекс для гео‑поиска
CREATE INDEX IF NOT EXISTS incidents_geo_idx
    ON incidents
        USING GIST (geo_point);

-- Индекс по активным инцидентам
CREATE INDEX IF NOT EXISTS incidents_status_idx
    ON incidents (status)
    WHERE status = 'active';

-- +goose Down
DROP INDEX IF EXISTS incidents_status_idx;
DROP INDEX IF EXISTS incidents_geo_idx;
DROP TABLE IF EXISTS incidents;
