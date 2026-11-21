-- Milestone 4 - preventive maintenance core.
-- Additive only: M0 tables remain unchanged except extension columns on work_orders.

CREATE TABLE pm_plans (
    id                    bigserial PRIMARY KEY,
    name                  text NOT NULL,
    description           text,
    unit_id               bigint REFERENCES units(id),
    unit_model_id         bigint REFERENCES unit_models(id),
    status                text NOT NULL DEFAULT 'draft'
                          CHECK (status IN ('draft', 'active', 'inactive')),
    effective_from        date NOT NULL,
    effective_to          date,
    interval_hm           numeric(12,1) CHECK (interval_hm IS NULL OR interval_hm > 0),
    interval_km           numeric(12,1) CHECK (interval_km IS NULL OR interval_km > 0),
    interval_days         int CHECK (interval_days IS NULL OR interval_days > 0),
    interval_rule         text NOT NULL DEFAULT 'first_reached'
                          CHECK (interval_rule IN ('first_reached', 'all_required')),
    upcoming_days         int CHECK (upcoming_days IS NULL OR upcoming_days BETWEEN 0 AND 365),
    due_soon_days         int CHECK (due_soon_days IS NULL OR due_soon_days BETWEEN 0 AND 365),
    upcoming_hm           numeric(12,1) CHECK (upcoming_hm IS NULL OR upcoming_hm >= 0),
    upcoming_km           numeric(12,1) CHECK (upcoming_km IS NULL OR upcoming_km >= 0),
    due_soon_hm           numeric(12,1) CHECK (due_soon_hm IS NULL OR due_soon_hm >= 0),
    due_soon_km           numeric(12,1) CHECK (due_soon_km IS NULL OR due_soon_km >= 0),
    auto_generate_wo      boolean NOT NULL DEFAULT false,
    default_priority      text NOT NULL DEFAULT 'medium'
                          CHECK (default_priority IN ('low', 'medium', 'high', 'critical')),
    default_assignee_id   bigint REFERENCES users(id),
    default_supervisor_id bigint REFERENCES users(id),
    created_by            bigint REFERENCES users(id),
    updated_by            bigint REFERENCES users(id),
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT pm_plans_exactly_one_target CHECK (
        (unit_id IS NOT NULL)::int + (unit_model_id IS NOT NULL)::int = 1
    ),
    CONSTRAINT pm_plans_interval_required CHECK (
        COALESCE(interval_hm, 0) > 0
        OR COALESCE(interval_km, 0) > 0
        OR COALESCE(interval_days, 0) > 0
    ),
    CONSTRAINT pm_plans_effective_range CHECK (
        effective_to IS NULL OR effective_to >= effective_from
    )
);

CREATE INDEX idx_pm_plans_status ON pm_plans (status);
CREATE INDEX idx_pm_plans_unit ON pm_plans (unit_id);
CREATE INDEX idx_pm_plans_unit_model ON pm_plans (unit_model_id);

CREATE TABLE pm_plan_checklist_items (
    id            bigserial PRIMARY KEY,
    pm_plan_id    bigint NOT NULL REFERENCES pm_plans(id) ON DELETE CASCADE,
    sequence      int NOT NULL CHECK (sequence > 0),
    component_id  bigint REFERENCES components(id),
    task          text NOT NULL,
    standard      text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (pm_plan_id, sequence)
);

CREATE INDEX idx_pm_plan_checklist_plan ON pm_plan_checklist_items (pm_plan_id, sequence);

CREATE TABLE pm_schedules (
    id                 bigserial PRIMARY KEY,
    pm_plan_id         bigint NOT NULL REFERENCES pm_plans(id) ON DELETE CASCADE,
    unit_id            bigint NOT NULL REFERENCES units(id),
    cycle_no           int NOT NULL DEFAULT 1 CHECK (cycle_no > 0),
    status             text NOT NULL DEFAULT 'scheduled'
                       CHECK (status IN ('scheduled', 'due', 'overdue', 'wo_created', 'completed', 'skipped', 'cancelled')),
    due_hm             numeric(12,1) CHECK (due_hm IS NULL OR due_hm >= 0),
    due_km             numeric(12,1) CHECK (due_km IS NULL OR due_km >= 0),
    due_date           date,
    last_completed_hm  numeric(12,1) CHECK (last_completed_hm IS NULL OR last_completed_hm >= 0),
    last_completed_km  numeric(12,1) CHECK (last_completed_km IS NULL OR last_completed_km >= 0),
    last_completed_at  timestamptz,
    override_reason    text,
    overridden_by      bigint REFERENCES users(id),
    overridden_at      timestamptz,
    skip_reason        text,
    skipped_by         bigint REFERENCES users(id),
    skipped_at         timestamptz,
    cancel_reason      text,
    cancelled_by       bigint REFERENCES users(id),
    cancelled_at       timestamptz,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    UNIQUE (pm_plan_id, unit_id, cycle_no)
);

CREATE UNIQUE INDEX pm_schedules_active_uidx
    ON pm_schedules (pm_plan_id, unit_id)
    WHERE status IN ('scheduled', 'due', 'overdue', 'wo_created');
CREATE INDEX idx_pm_schedules_status_due ON pm_schedules (status, due_date);
CREATE INDEX idx_pm_schedules_unit_status ON pm_schedules (unit_id, status);

CREATE TABLE work_order_checklist_items (
    id                        bigserial PRIMARY KEY,
    wo_id                     bigint NOT NULL REFERENCES work_orders(id) ON DELETE CASCADE,
    pm_plan_checklist_item_id bigint REFERENCES pm_plan_checklist_items(id) ON DELETE SET NULL,
    sequence                  int NOT NULL CHECK (sequence > 0),
    component_id              bigint REFERENCES components(id),
    task                      text NOT NULL,
    standard                  text,
    result                    text NOT NULL DEFAULT 'pending'
                              CHECK (result IN ('pending', 'ok', 'not_ok', 'na')),
    checked_by                bigint REFERENCES users(id),
    checked_at                timestamptz,
    created_at                timestamptz NOT NULL DEFAULT now(),
    updated_at                timestamptz NOT NULL DEFAULT now(),
    UNIQUE (wo_id, sequence)
);

CREATE INDEX idx_wo_checklist_wo ON work_order_checklist_items (wo_id, sequence);

ALTER TABLE work_orders
    ADD COLUMN IF NOT EXISTS pm_schedule_id bigint REFERENCES pm_schedules(id);

ALTER TABLE work_orders DROP CONSTRAINT IF EXISTS work_orders_source_check;
ALTER TABLE work_orders
    ADD CONSTRAINT work_orders_source_check
    CHECK (source IN ('p2h', 'manual', 'pm'));

CREATE INDEX IF NOT EXISTS idx_work_orders_pm_schedule ON work_orders (pm_schedule_id);
CREATE UNIQUE INDEX IF NOT EXISTS work_orders_pm_schedule_active_uidx
    ON work_orders (pm_schedule_id)
    WHERE pm_schedule_id IS NOT NULL AND status <> 'cancelled';

ALTER TABLE notifications ADD COLUMN IF NOT EXISTS dedupe_key text;
CREATE UNIQUE INDEX IF NOT EXISTS notifications_user_dedupe_uidx
    ON notifications (user_id, dedupe_key)
    WHERE dedupe_key IS NOT NULL;

INSERT INTO app_settings (key, value, updated_at)
VALUES
    ('pm_auto_wo', 'false', now()),
    ('pm_upcoming_days', '30', now()),
    ('pm_due_soon_days', '7', now())
ON CONFLICT (key) DO NOTHING;
