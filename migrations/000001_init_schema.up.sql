-- GSI Work Order & Maintenance
-- Milestone 0 - initial schema.
--
-- Source: PRD Appendix A2.
-- Extension: wo_candidates.notified_at is required by PRD Appendix A3 point 5
-- (24-hour reminder, one notification per candidate).

CREATE TABLE users (
    id            bigserial PRIMARY KEY,
    odoo_user_id  bigint UNIQUE,
    nik           text,
    name          text,
    email         text,
    role          text NOT NULL DEFAULT 'mechanic'
                  CHECK (role IN ('admin', 'planner', 'mechanic', 'viewer')),
    is_active     boolean NOT NULL DEFAULT true,
    last_login_at timestamptz,
    synced_at     timestamptz
);

CREATE TABLE employees (
    id               bigserial PRIMARY KEY,
    odoo_employee_id bigint UNIQUE,
    nik              text,
    name             text,
    job_title        text,
    department       text,
    site_location    text,
    user_id          bigint REFERENCES users(id),
    is_active        boolean NOT NULL DEFAULT true,
    synced_at        timestamptz
);

CREATE TABLE units (
    id           bigserial PRIMARY KEY,
    odoo_unit_id bigint UNIQUE,
    code         text NOT NULL,
    unit_type    text,
    merk         text,
    class_unit   text,
    license_plate text,
    site_location text,
    is_breakdown boolean NOT NULL DEFAULT false,
    synced_at    timestamptz
);

CREATE TABLE p2h_reports (
    id                bigserial PRIMARY KEY,
    odoo_report_id    bigint UNIQUE NOT NULL,
    name              text,
    report_date       date,
    unit_id           bigint REFERENCES units(id),
    unit_type         text,
    shift             text,
    foreman_name      text,
    employee_name     text,
    location          text,
    state             text,
    approval_status   text,
    km_akhir          numeric,
    hm_awal           numeric,
    notes             text,
    checked_items     int,
    findings_count    int,
    detail_synced_at  timestamptz,
    raw               jsonb,
    synced_at         timestamptz
);

CREATE TABLE p2h_findings (
    id            bigserial PRIMARY KEY,
    odoo_line_id  bigint UNIQUE NOT NULL,
    report_id     bigint NOT NULL REFERENCES p2h_reports(id) ON DELETE CASCADE,
    question_id   bigint,
    component     text,
    category      text,
    category_norm text,
    condition     text CHECK (condition IN ('rusak', 'tidak_ada')),
    remark        text
);

CREATE TABLE wo_candidates (
    id                 bigserial PRIMARY KEY,
    p2h_report_id      bigint UNIQUE NOT NULL REFERENCES p2h_reports(id),
    unit_id            bigint REFERENCES units(id),
    suggested_priority text CHECK (suggested_priority IN ('low', 'medium', 'high', 'critical')),
    findings_count     int,
    rusak_count        int,
    status             text NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending', 'approved', 'skipped', 'auto_created', 'archived')),
    decided_by         bigint REFERENCES users(id),
    decided_at         timestamptz,
    skip_reason        text,
    -- Intentionally no FK: work_orders.candidate_id references this table.
    -- Consistency is enforced transactionally when a candidate is approved.
    wo_id              bigint,
    notified_at        timestamptz,
    created_at         timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE work_orders (
    id            bigserial PRIMARY KEY,
    code          text UNIQUE NOT NULL,
    unit_id       bigint REFERENCES units(id),
    source        text NOT NULL CHECK (source IN ('p2h', 'manual')),
    p2h_report_id bigint UNIQUE REFERENCES p2h_reports(id),
    candidate_id  bigint REFERENCES wo_candidates(id),
    title         text NOT NULL,
    description   text,
    priority      text CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    status        text NOT NULL DEFAULT 'open'
                  CHECK (status IN ('open', 'in_progress', 'waiting_part', 'done', 'cancelled')),
    assignee_id   bigint REFERENCES users(id),
    created_by    bigint REFERENCES users(id),
    due_date      date,
    closed_at     timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE wo_findings (
    wo_id      bigint NOT NULL REFERENCES work_orders(id) ON DELETE CASCADE,
    finding_id bigint NOT NULL REFERENCES p2h_findings(id),
    PRIMARY KEY (wo_id, finding_id)
);

CREATE TABLE wo_updates (
    id          bigserial PRIMARY KEY,
    wo_id       bigint NOT NULL REFERENCES work_orders(id) ON DELETE CASCADE,
    user_id     bigint REFERENCES users(id),
    note        text,
    status_from text,
    status_to   text,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE wo_attachments (
    id          bigserial PRIMARY KEY,
    wo_id       bigint NOT NULL REFERENCES work_orders(id) ON DELETE CASCADE,
    file_path   text,
    mime        text,
    size_bytes  int,
    uploaded_by bigint REFERENCES users(id),
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE notifications (
    id         bigserial PRIMARY KEY,
    user_id    bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       text,
    title      text,
    body       text,
    ref_type   text,
    ref_id     bigint,
    read_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE app_settings (
    key        text PRIMARY KEY,
    value      text,
    updated_by bigint REFERENCES users(id),
    updated_at timestamptz
);

CREATE TABLE app_settings_history (
    id         bigserial PRIMARY KEY,
    key        text,
    old_value  text,
    new_value  text,
    changed_by bigint REFERENCES users(id),
    changed_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sync_runs (
    id          bigserial PRIMARY KEY,
    kind        text,
    status      text,
    started_at  timestamptz,
    finished_at timestamptz,
    reports     int NOT NULL DEFAULT 0,
    details     int NOT NULL DEFAULT 0,
    findings    int NOT NULL DEFAULT 0,
    candidates  int NOT NULL DEFAULT 0,
    work_orders int NOT NULL DEFAULT 0,
    error       text
);

CREATE TABLE sync_state (
    key        text PRIMARY KEY,
    value      text,
    updated_at timestamptz
);

CREATE INDEX idx_work_orders_status_priority ON work_orders (status, priority);
CREATE INDEX idx_work_orders_unit_id ON work_orders (unit_id);
CREATE INDEX idx_wo_candidates_status_created_at ON wo_candidates (status, created_at);
CREATE INDEX idx_p2h_reports_unit_date ON p2h_reports (unit_id, report_date DESC);
CREATE INDEX idx_p2h_findings_report_id ON p2h_findings (report_id);
CREATE INDEX idx_p2h_findings_category_norm ON p2h_findings (category_norm);
CREATE INDEX idx_notifications_user_read ON notifications (user_id, read_at);
