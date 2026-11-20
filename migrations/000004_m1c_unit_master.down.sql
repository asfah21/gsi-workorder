DROP TABLE IF EXISTS unit_meter_readings;
DROP INDEX IF EXISTS idx_units_active;
DROP INDEX IF EXISTS idx_units_engine_model;
DROP INDEX IF EXISTS idx_units_unit_model;
DROP INDEX IF EXISTS units_code_normalized_uidx;

ALTER TABLE units
    DROP COLUMN IF EXISTS is_active,
    DROP COLUMN IF EXISTS km_updated_at,
    DROP COLUMN IF EXISTS hm_updated_at,
    DROP COLUMN IF EXISTS current_km,
    DROP COLUMN IF EXISTS current_hm,
    DROP COLUMN IF EXISTS engine_serial_number,
    DROP COLUMN IF EXISTS serial_number,
    DROP COLUMN IF EXISTS engine_model_id,
    DROP COLUMN IF EXISTS unit_model_id;

DROP TABLE IF EXISTS components;
DROP TABLE IF EXISTS engine_models;
DROP TABLE IF EXISTS unit_models;
