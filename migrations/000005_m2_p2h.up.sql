-- Milestone 2 - P2H sync, candidates, notifications, and P2H-to-WO approval.
-- M0 is locked; this migration only adds tables/columns/indexes.

-- Monthly WO counters. The code format is WO-YYYYMM-####, and the period is
-- calculated in the application timezone (Asia/Makassar), not UTC.
CREATE TABLE work_order_code_counters (
    period      text PRIMARY KEY,
    last_number integer NOT NULL CHECK (last_number >= 0),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- Approval note is separate from skip_reason because a candidate can be
-- approved with a note or skipped with a mandatory reason.
ALTER TABLE wo_candidates
    ADD COLUMN decision_note text;

-- Detail work queue and candidate filters.
CREATE INDEX idx_p2h_reports_detail_pending
    ON p2h_reports (id)
    WHERE detail_synced_at IS NULL;

CREATE INDEX idx_wo_candidates_unit_status
    ON wo_candidates (unit_id, status);

-- Notification list and unread badge queries are ordered by newest first.
CREATE INDEX idx_notifications_user_created_at
    ON notifications (user_id, created_at DESC);
