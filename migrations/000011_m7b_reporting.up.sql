-- Milestone 7B - dashboard and reporting views. Additive only.
-- Every row is derived from actual transaction tables; no static totals, no
-- duplicated reservation/usage logic, and no procurement or valuation data.

-- Flattened PM schedule rows for due/overdue dashboards and compliance reports.
-- Thresholds come from app_settings with per-plan overrides; no hardcoded interval.
CREATE VIEW v_pm_due AS
SELECT
    s.id AS schedule_id,
    s.pm_plan_id,
    p.name AS plan_name,
    s.unit_id,
    u.code AS unit_code,
    um.id AS unit_model_id,
    concat_ws(' ', um.manufacturer, um.model) AS unit_model_name,
    s.cycle_no,
    s.status,
    s.due_date,
    s.due_hm,
    s.due_km,
    s.last_completed_at,
    s.last_completed_hm,
    s.last_completed_km,
    s.created_at,
    w.id AS work_order_id,
    COALESCE(w.code, '') AS work_order_code
FROM pm_schedules s
JOIN pm_plans p ON p.id = s.pm_plan_id
JOIN units u ON u.id = s.unit_id
LEFT JOIN unit_models um ON um.id = u.unit_model_id
LEFT JOIN LATERAL (
    SELECT wo.id, wo.code
    FROM work_orders wo
    WHERE wo.pm_schedule_id = s.id
      AND wo.status <> 'cancelled'
    ORDER BY wo.created_at DESC, wo.id DESC
    LIMIT 1
) w ON true;

-- Compliance classification mirrors the M4 /pm/compliance endpoint. A completed
-- schedule is on time when completion is not past the calendar/HM/KM due value.
CREATE VIEW v_pm_compliance AS
SELECT
    s.id AS schedule_id,
    s.pm_plan_id,
    p.name AS plan_name,
    s.unit_id,
    u.code AS unit_code,
    um.id AS unit_model_id,
    concat_ws(' ', um.manufacturer, um.model) AS unit_model_name,
    s.cycle_no,
    s.status,
    s.created_at,
    s.due_date,
    s.due_hm,
    s.due_km,
    s.last_completed_at,
    s.last_completed_hm,
    s.last_completed_km,
    (s.status = 'completed') AS is_completed,
    (
        s.status = 'completed'
        AND (s.due_date IS NULL OR (s.last_completed_at AT TIME ZONE 'Asia/Makassar')::date <= s.due_date)
        AND (s.due_hm IS NULL OR s.last_completed_hm IS NULL OR s.last_completed_hm <= s.due_hm)
        AND (s.due_km IS NULL OR s.last_completed_km IS NULL OR s.last_completed_km <= s.due_km)
    ) AS is_on_time,
    (s.status = 'overdue') AS is_overdue,
    (s.status = 'skipped') AS is_missed
FROM pm_schedules s
JOIN pm_plans p ON p.id = s.pm_plan_id
JOIN units u ON u.id = s.unit_id
LEFT JOIN unit_models um ON um.id = u.unit_model_id;

-- Work orders currently waiting for material. Duration is derived from the
-- latest transition into waiting_part; the same wo_updates audit table is used
-- by the unit downtime view, so no waiting-material table is introduced.
CREATE VIEW v_waiting_part_units AS
SELECT DISTINCT ON (wo.id)
    wo.id AS wo_id,
    wo.code AS wo_code,
    wo.unit_id,
    u.code AS unit_code,
    wo.title,
    wo.priority,
    wo.assignee_id,
    COALESCE(assignee.name, '') AS assignee_name,
    wo.supervisor_id,
    COALESCE(supervisor.name, '') AS supervisor_name,
    upd.created_at AS waiting_since,
    GREATEST(
        0,
        EXTRACT(EPOCH FROM (now() - upd.created_at)) / 60
    )::numeric(14, 1) AS waiting_minutes,
    COALESCE(mr.status, '') AS material_status
FROM work_orders wo
JOIN units u ON u.id = wo.unit_id
JOIN wo_updates upd
    ON upd.wo_id = wo.id
   AND upd.status_to = 'waiting_part'
LEFT JOIN users assignee ON assignee.id = wo.assignee_id
LEFT JOIN users supervisor ON supervisor.id = wo.supervisor_id
LEFT JOIN LATERAL (
    SELECT mr.status
    FROM material_requirements mr
    WHERE mr.wo_id = wo.id
      AND mr.status <> 'cancelled'
    ORDER BY mr.created_at DESC, mr.id DESC
    LIMIT 1
) mr ON true
WHERE wo.status = 'waiting_part'
ORDER BY wo.id, upd.created_at DESC, upd.id DESC;

-- Actual sparepart consumption. Returns are negative so net usage stays
-- visible without double counting; unissued material requirements are absent
-- by construction because they have no work_order_sparepart_usage row.
CREATE VIEW v_sparepart_consumption AS
SELECT
    su.id AS usage_id,
    su.wo_id,
    wo.code AS wo_code,
    wo.unit_id,
    u.code AS unit_code,
    um.id AS unit_model_id,
    concat_ws(' ', um.manufacturer, um.model) AS unit_model_name,
    su.sparepart_id,
    sp.code AS sparepart_code,
    sp.name AS sparepart_name,
    sp.uom,
    su.component_id,
    c.code AS component_code,
    c.name AS component_name,
    su.movement_type,
    CASE WHEN su.movement_type = 'issue' THEN su.qty ELSE -su.qty END AS net_qty,
    su.used_at,
    su.hm,
    su.km,
    wl.id AS warehouse_location_id,
    wl.warehouse_id,
    w.code AS warehouse_code,
    w.name AS warehouse_name,
    wl.code AS location_code,
    wl.name AS location_name
FROM work_order_sparepart_usage su
JOIN work_orders wo ON wo.id = su.wo_id
JOIN spareparts sp ON sp.id = su.sparepart_id
LEFT JOIN units u ON u.id = wo.unit_id
LEFT JOIN unit_models um ON um.id = u.unit_model_id
LEFT JOIN components c ON c.id = su.component_id
LEFT JOIN warehouse_locations wl ON wl.id = su.warehouse_location_id
LEFT JOIN warehouses w ON w.id = wl.warehouse_id;

-- Physical inventory movement ledger. Reserve/release are planning entries
-- and are excluded; reversed posting remains visible with its status so the
-- audit trail is not silently dropped.
CREATE VIEW v_inventory_movement AS
SELECT
    t.id AS transaction_id,
    t.code AS transaction_code,
    t.type,
    t.status,
    t.reference_type,
    t.reference_no,
    t.posted_at,
    t.reversal_of_id,
    t.posted_by,
    COALESCE(posted_by_user.name, '') AS posted_by_name,
    it.id AS item_id,
    it.sparepart_id,
    sp.code AS sparepart_code,
    sp.name AS sparepart_name,
    sp.uom,
    it.qty,
    COALESCE(it.notes, '') AS notes,
    fl.id AS from_location_id,
    fl.code AS from_location_code,
    fw.code AS from_warehouse_code,
    tl.id AS to_location_id,
    tl.code AS to_location_code,
    tw.code AS to_warehouse_code
FROM inventory_transactions t
JOIN inventory_transaction_items it ON it.transaction_id = t.id
JOIN spareparts sp ON sp.id = it.sparepart_id
LEFT JOIN users posted_by_user ON posted_by_user.id = t.posted_by
LEFT JOIN warehouse_locations fl ON fl.id = it.from_location_id
LEFT JOIN warehouses fw ON fw.id = fl.warehouse_id
LEFT JOIN warehouse_locations tl ON tl.id = it.to_location_id
LEFT JOIN warehouses tw ON tw.id = tl.warehouse_id
WHERE t.type IN ('receipt', 'issue', 'return', 'transfer', 'adjustment', 'opname');

CREATE INDEX IF NOT EXISTS idx_pm_schedules_created_at
    ON pm_schedules (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wo_updates_status_created
    ON wo_updates (status_to, created_at DESC)
    WHERE status_to = 'waiting_part';
CREATE INDEX IF NOT EXISTS idx_wo_usage_used_at
    ON work_order_sparepart_usage (used_at DESC);
CREATE INDEX IF NOT EXISTS idx_inventory_transactions_posted_at
    ON inventory_transactions (posted_at DESC)
    WHERE status = 'posted';
