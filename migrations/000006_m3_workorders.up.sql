-- M3 baseline work-order lifecycle fields.
-- Keeps M0 schema additive and leaves source=p2h/manual unchanged.

ALTER TABLE work_orders
    ADD COLUMN IF NOT EXISTS maintenance_type text NOT NULL DEFAULT 'corrective',
    ADD COLUMN IF NOT EXISTS supervisor_id bigint REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS started_at timestamptz;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'work_orders_maintenance_type_check'
    ) THEN
        ALTER TABLE work_orders
            ADD CONSTRAINT work_orders_maintenance_type_check
            CHECK (maintenance_type IN ('preventive', 'corrective', 'breakdown', 'inspection'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_work_orders_assignee_status
    ON work_orders (assignee_id, status);
CREATE INDEX IF NOT EXISTS idx_work_orders_supervisor_id
    ON work_orders (supervisor_id);
CREATE INDEX IF NOT EXISTS idx_work_orders_created_at
    ON work_orders (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wo_updates_wo_created
    ON wo_updates (wo_id, created_at);

-- wo_updates is append-only. Application code never mutates it, and this
-- trigger prevents accidental updates/deletes outside the application.
CREATE OR REPLACE FUNCTION prevent_wo_updates_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'wo_updates is append-only';
END;
$$;

DROP TRIGGER IF EXISTS wo_updates_append_only ON wo_updates;
CREATE TRIGGER wo_updates_append_only
    BEFORE UPDATE OR DELETE ON wo_updates
    FOR EACH ROW
    EXECUTE FUNCTION prevent_wo_updates_mutation();
