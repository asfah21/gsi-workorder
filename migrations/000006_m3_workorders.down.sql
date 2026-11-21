DROP TRIGGER IF EXISTS wo_updates_append_only ON wo_updates;
DROP FUNCTION IF EXISTS prevent_wo_updates_mutation();

DROP INDEX IF EXISTS idx_wo_updates_wo_created;
DROP INDEX IF EXISTS idx_work_orders_created_at;
DROP INDEX IF EXISTS idx_work_orders_supervisor_id;
DROP INDEX IF EXISTS idx_work_orders_assignee_status;

ALTER TABLE work_orders
    DROP CONSTRAINT IF EXISTS work_orders_maintenance_type_check;

ALTER TABLE work_orders
    DROP COLUMN IF EXISTS started_at,
    DROP COLUMN IF EXISTS supervisor_id,
    DROP COLUMN IF EXISTS maintenance_type;
