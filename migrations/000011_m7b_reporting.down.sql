DROP INDEX IF EXISTS idx_inventory_transactions_posted_at;
DROP INDEX IF EXISTS idx_wo_usage_used_at;
DROP INDEX IF EXISTS idx_wo_updates_status_created;
DROP INDEX IF EXISTS idx_pm_schedules_created_at;

DROP VIEW IF EXISTS v_inventory_movement;
DROP VIEW IF EXISTS v_sparepart_consumption;
DROP VIEW IF EXISTS v_waiting_part_units;
DROP VIEW IF EXISTS v_pm_compliance;
DROP VIEW IF EXISTS v_pm_due;
