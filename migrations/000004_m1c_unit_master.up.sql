-- Milestone 1C - unit master, engine master, and manual HM/KM readings.
-- M0 is locked; this migration only adds tables/columns and indexes.

CREATE TABLE unit_models (
    id            bigserial PRIMARY KEY,
    manufacturer  text NOT NULL,
    model         text NOT NULL,
    category      text,
    notes         text,
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (manufacturer, model)
);

CREATE TABLE engine_models (
    id              bigserial PRIMARY KEY,
    manufacturer    text NOT NULL,
    model           text NOT NULL,
    fuel_type       text,
    displacement_cc int CHECK (displacement_cc IS NULL OR displacement_cc > 0),
    notes           text,
    is_active       boolean NOT NULL DEFAULT true,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (manufacturer, model)
);

CREATE TABLE components (
    id          bigserial PRIMARY KEY,
    code        text NOT NULL UNIQUE,
    name        text NOT NULL,
    category    text,
    is_active   boolean NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE units
    ADD COLUMN unit_model_id bigint REFERENCES unit_models(id),
    ADD COLUMN engine_model_id bigint REFERENCES engine_models(id),
    ADD COLUMN serial_number text,
    ADD COLUMN engine_serial_number text,
    ADD COLUMN current_hm numeric(12,1) NOT NULL DEFAULT 0 CHECK (current_hm >= 0),
    ADD COLUMN current_km numeric(12,1) NOT NULL DEFAULT 0 CHECK (current_km >= 0),
    ADD COLUMN hm_updated_at timestamptz,
    ADD COLUMN km_updated_at timestamptz,
    ADD COLUMN is_active boolean NOT NULL DEFAULT true;

-- Unit codes are the natural key used by the UI/manual master and later by
-- P2H unit upserts. Reject duplicates with an actionable migration error.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM units
        WHERE btrim(code) <> ''
        GROUP BY lower(btrim(code))
        HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'units.code contains duplicates after normalization; resolve duplicates before applying M1C';
    END IF;
END;
$$;

CREATE UNIQUE INDEX units_code_normalized_uidx
    ON units (lower(btrim(code)))
    WHERE btrim(code) <> '';

CREATE INDEX idx_units_unit_model ON units (unit_model_id);
CREATE INDEX idx_units_engine_model ON units (engine_model_id);
CREATE INDEX idx_units_active ON units (is_active);

CREATE TABLE unit_meter_readings (
    id           bigserial PRIMARY KEY,
    unit_id      bigint NOT NULL REFERENCES units(id),
    reading_type text NOT NULL CHECK (reading_type IN ('hm', 'km')),
    value        numeric(12,1) NOT NULL CHECK (value >= 0),
    recorded_at  timestamptz NOT NULL,
    source_type  text NOT NULL CHECK (source_type IN ('p2h', 'wo', 'manual', 'sync')),
    source_id    bigint,
    notes        text,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uq_unit_meter_readings_source
    ON unit_meter_readings (unit_id, reading_type, source_type, source_id)
    WHERE source_id IS NOT NULL;

CREATE INDEX idx_unit_meter_readings_unit_time
    ON unit_meter_readings (unit_id, reading_type, recorded_at DESC);
