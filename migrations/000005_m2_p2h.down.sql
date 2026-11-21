DROP INDEX IF EXISTS idx_notifications_user_created_at;
DROP INDEX IF EXISTS idx_wo_candidates_unit_status;
DROP INDEX IF EXISTS idx_p2h_reports_detail_pending;

ALTER TABLE wo_candidates
    DROP COLUMN IF EXISTS decision_note;

DROP TABLE IF EXISTS work_order_code_counters;
