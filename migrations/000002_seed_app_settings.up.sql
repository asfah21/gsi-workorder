-- Default settings for Milestone 0.
-- p2h_auto_wo MUST default to false: P2H findings create candidates, not WOs.
INSERT INTO app_settings (key, value, updated_at)
VALUES
    ('p2h_auto_wo', 'false', now()),
    ('candidate_sla_hours', '24', now()),
    ('candidate_backfill_days', '30', now())
ON CONFLICT (key) DO NOTHING;
