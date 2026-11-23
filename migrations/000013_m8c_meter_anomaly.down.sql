DROP INDEX IF EXISTS idx_unit_meter_readings_valid;

ALTER TABLE unit_meter_readings
    DROP COLUMN IF EXISTS anomaly_reason,
    DROP COLUMN IF EXISTS is_anomaly;
