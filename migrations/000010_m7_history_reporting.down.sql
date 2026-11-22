DROP INDEX IF EXISTS idx_work_orders_maintenance_status;
DROP INDEX IF EXISTS idx_work_orders_unit_closed_at;
DROP INDEX IF EXISTS idx_work_orders_unit_created_at;

DROP VIEW IF EXISTS v_unit_downtime;
DROP VIEW IF EXISTS v_component_replacement_frequency;
DROP VIEW IF EXISTS v_unit_component_history;
DROP VIEW IF EXISTS v_unit_maintenance_history;
