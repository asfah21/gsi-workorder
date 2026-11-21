DROP INDEX IF EXISTS work_orders_pm_schedule_active_uidx;
DROP INDEX IF EXISTS idx_work_orders_pm_schedule;

DELETE FROM work_orders WHERE source = 'pm';

ALTER TABLE work_orders DROP CONSTRAINT IF EXISTS work_orders_source_check;
ALTER TABLE work_orders
    ADD CONSTRAINT work_orders_source_check
    CHECK (source IN ('p2h', 'manual'));
ALTER TABLE work_orders DROP COLUMN IF EXISTS pm_schedule_id;

DROP INDEX IF EXISTS notifications_user_dedupe_uidx;
ALTER TABLE notifications DROP COLUMN IF EXISTS dedupe_key;

DROP TABLE IF EXISTS work_order_checklist_items;
DROP TABLE IF EXISTS pm_schedules;
DROP TABLE IF EXISTS pm_plan_checklist_items;
DROP TABLE IF EXISTS pm_plans;

DELETE FROM app_settings
WHERE key IN ('pm_auto_wo', 'pm_upcoming_days', 'pm_due_soon_days');
