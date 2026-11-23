-- Revert M9 Work Request schema. Dispatch users fall back to planner. New
-- statuses are mapped to the legacy set; manual requests are kept, so
-- p2h_report_id remains nullable when manual rows exist.

UPDATE users SET role = 'planner' WHERE role = 'dispatch';
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('admin', 'planner', 'mechanic', 'storekeeper', 'viewer'));

UPDATE wo_candidates SET status = 'approved' WHERE status = 'converted';
UPDATE wo_candidates SET status = 'pending' WHERE status = 'validated';

ALTER TABLE wo_candidates DROP CONSTRAINT IF EXISTS wo_candidates_status_check;
ALTER TABLE wo_candidates ADD CONSTRAINT wo_candidates_status_check
    CHECK (status IN ('pending', 'approved', 'skipped', 'auto_created', 'archived'));

ALTER TABLE wo_candidates DROP CONSTRAINT IF EXISTS wo_candidates_manual_fields_check;
ALTER TABLE wo_candidates DROP CONSTRAINT IF EXISTS wo_candidates_work_category_check;
ALTER TABLE wo_candidates DROP CONSTRAINT IF EXISTS wo_candidates_source_check;

DROP INDEX IF EXISTS idx_wo_candidates_backlog;
DROP INDEX IF EXISTS idx_wo_candidates_source_status_created;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM wo_candidates WHERE p2h_report_id IS NULL) THEN
        ALTER TABLE wo_candidates ALTER COLUMN p2h_report_id SET NOT NULL;
    END IF;
END;
$$;

ALTER TABLE wo_candidates
    DROP COLUMN IF EXISTS converted_at,
    DROP COLUMN IF EXISTS converted_by,
    DROP COLUMN IF EXISTS validated_at,
    DROP COLUMN IF EXISTS validated_by,
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS is_breakdown,
    DROP COLUMN IF EXISTS work_category,
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS title,
    DROP COLUMN IF EXISTS source;
