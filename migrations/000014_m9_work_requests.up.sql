-- Milestone 9A - Work Request: dispatch role, manual requests, validation,
-- conversion, work categories, and backlog. Existing P2H candidates keep
-- working; manual requests have p2h_report_id = NULL.

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('admin', 'planner', 'mechanic', 'storekeeper', 'viewer', 'dispatch'));

ALTER TABLE wo_candidates
    ALTER COLUMN p2h_report_id DROP NOT NULL,
    ADD COLUMN source text NOT NULL DEFAULT 'p2h',
    ADD COLUMN title text,
    ADD COLUMN description text,
    ADD COLUMN work_category text,
    ADD COLUMN is_breakdown boolean NOT NULL DEFAULT false,
    ADD COLUMN created_by bigint REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN validated_by bigint REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN validated_at timestamptz,
    ADD COLUMN converted_by bigint REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN converted_at timestamptz;

ALTER TABLE wo_candidates DROP CONSTRAINT IF EXISTS wo_candidates_source_check;
ALTER TABLE wo_candidates ADD CONSTRAINT wo_candidates_source_check
    CHECK (source IN ('p2h', 'manual'));

ALTER TABLE wo_candidates DROP CONSTRAINT IF EXISTS wo_candidates_work_category_check;
ALTER TABLE wo_candidates ADD CONSTRAINT wo_candidates_work_category_check
    CHECK (work_category IS NULL OR work_category IN ('Service', 'Scheduled', 'Unscheduled', 'Incident', 'UBOP'));

ALTER TABLE wo_candidates DROP CONSTRAINT IF EXISTS wo_candidates_status_check;
ALTER TABLE wo_candidates ADD CONSTRAINT wo_candidates_status_check
    CHECK (status IN ('pending', 'validated', 'converted', 'approved', 'auto_created', 'skipped', 'archived'));

ALTER TABLE wo_candidates DROP CONSTRAINT IF EXISTS wo_candidates_manual_fields_check;
ALTER TABLE wo_candidates ADD CONSTRAINT wo_candidates_manual_fields_check
    CHECK (
        source <> 'manual'
        OR (
            unit_id IS NOT NULL
            AND title IS NOT NULL
            AND btrim(title) <> ''
            AND work_category IS NOT NULL
        )
    );

CREATE INDEX idx_wo_candidates_source_status_created
    ON wo_candidates (source, status, created_at DESC);

-- Backlog default: non-breakdown Work Request that has not started yet
-- (pending/validated). Deferred requests remain pending/validated and thus
-- stay in this queue.
CREATE INDEX idx_wo_candidates_backlog
    ON wo_candidates (status, created_at DESC)
    WHERE is_breakdown = false AND status IN ('pending', 'validated');
