-- Milestone 8C - mark lower P2H meter readings as anomalies instead of letting
-- them overwrite the unit's current HM/KM cache. Additive only.

ALTER TABLE unit_meter_readings
    ADD COLUMN is_anomaly boolean NOT NULL DEFAULT false,
    ADD COLUMN anomaly_reason text;

CREATE INDEX idx_unit_meter_readings_valid
    ON unit_meter_readings (unit_id, reading_type, recorded_at DESC)
    WHERE is_anomaly = false;
