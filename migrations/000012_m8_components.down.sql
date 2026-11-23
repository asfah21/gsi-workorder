DROP VIEW IF EXISTS v_unit_component_lifetime;
DROP TABLE IF EXISTS unit_component_installations;

ALTER TABLE components
    DROP CONSTRAINT IF EXISTS components_component_class_check,
    DROP CONSTRAINT IF EXISTS components_default_lifetime_hm_check,
    DROP CONSTRAINT IF EXISTS components_default_lifetime_days_check,
    DROP CONSTRAINT IF EXISTS components_default_lifetime_km_check,
    DROP COLUMN IF EXISTS component_class,
    DROP COLUMN IF EXISTS default_lifetime_hm,
    DROP COLUMN IF EXISTS default_lifetime_days,
    DROP COLUMN IF EXISTS default_lifetime_km;
