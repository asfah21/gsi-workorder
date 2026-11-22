-- Milestone 7A - unit history, component history, component intervals, and
-- downtime reporting. Additive only: no history table, no duplicated material
-- reservation logic. All history is derived from work orders, checklist,
-- usage, readings, and wo_updates.

-- Unit maintenance timeline. One row per work order, including open and
-- cancelled work orders so the unit profile remains an audit view.
CREATE VIEW v_unit_maintenance_history AS
SELECT
    wo.id AS wo_id,
    wo.unit_id,
    u.code AS unit_code,
    wo.code AS wo_code,
    wo.source,
    wo.maintenance_type,
    wo.status,
    COALESCE(wo.priority, '') AS priority,
    wo.title,
    COALESCE(wo.description, '') AS description,
    wo.p2h_report_id,
    wo.candidate_id,
    wo.pm_schedule_id,
    wo.assignee_id,
    COALESCE(assignee.name, '') AS assignee_name,
    wo.supervisor_id,
    COALESCE(supervisor.name, '') AS supervisor_name,
    wo.created_by,
    COALESCE(creator.name, '') AS created_by_name,
    wo.due_date,
    wo.started_at,
    wo.closed_at,
    wo.created_at,
    CASE
        WHEN wo.closed_at IS NOT NULL THEN
            GREATEST(
                0,
                EXTRACT(EPOCH FROM (wo.closed_at - COALESCE(wo.started_at, wo.created_at))) / 60
            )::numeric(14, 1)
        ELSE NULL
    END AS elapsed_minutes,
    COALESCE(checklist.checklist_total, 0) AS checklist_total,
    COALESCE(checklist.checklist_done, 0) AS checklist_done,
    COALESCE(checklist.checklist_not_ok, 0) AS checklist_not_ok,
    COALESCE(findings.findings_count, 0) AS findings_count,
    COALESCE(usage.usage_count, 0) AS usage_count,
    GREATEST(COALESCE(usage.usage_qty, 0), 0)::numeric(14, 3) AS usage_qty,
    COALESCE(material.status, '') AS material_status
FROM work_orders wo
JOIN units u ON u.id = wo.unit_id
LEFT JOIN users assignee ON assignee.id = wo.assignee_id
LEFT JOIN users supervisor ON supervisor.id = wo.supervisor_id
LEFT JOIN users creator ON creator.id = wo.created_by
LEFT JOIN LATERAL (
    SELECT
        count(*)::int AS checklist_total,
        count(*) FILTER (WHERE c.result <> 'pending')::int AS checklist_done,
        count(*) FILTER (WHERE c.result = 'not_ok')::int AS checklist_not_ok
    FROM work_order_checklist_items c
    WHERE c.wo_id = wo.id
) checklist ON true
LEFT JOIN LATERAL (
    SELECT count(*)::int AS findings_count
    FROM wo_findings f
    WHERE f.wo_id = wo.id
) findings ON true
LEFT JOIN LATERAL (
    SELECT
        count(DISTINCT uu.sparepart_id)
            FILTER (WHERE uu.movement_type = 'issue')::int AS usage_count,
        COALESCE(sum(
            CASE WHEN uu.movement_type = 'issue' THEN uu.qty ELSE -uu.qty END
        ), 0) AS usage_qty
    FROM work_order_sparepart_usage uu
    WHERE uu.wo_id = wo.id
) usage ON true
LEFT JOIN LATERAL (
    SELECT mr.status
    FROM material_requirements mr
    WHERE mr.wo_id = wo.id
      AND mr.status <> 'cancelled'
    ORDER BY mr.created_at DESC, mr.id DESC
    LIMIT 1
) material ON true;

-- Component replacement history. One event is one work order plus one
-- component. When component_id is null, the sparepart is used as the
-- fallback component key so existing M6 usage still appears in history.
-- Quantities are net of posted returns; fully returned events are omitted.
CREATE VIEW v_unit_component_history AS
WITH usage_rows AS (
    SELECT
        su.wo_id,
        wo.unit_id,
        su.sparepart_id,
        su.component_id,
        su.qty,
        su.hm,
        su.km,
        su.movement_type,
        su.used_at
    FROM work_order_sparepart_usage su
    JOIN work_orders wo ON wo.id = su.wo_id
    WHERE wo.unit_id IS NOT NULL
),
component_events AS (
    SELECT
        ur.unit_id,
        ur.wo_id,
        ur.component_id,
        CASE WHEN ur.component_id IS NULL THEN ur.sparepart_id END AS fallback_sparepart_id,
        min(ur.used_at) FILTER (WHERE ur.movement_type = 'issue') AS used_at,
        sum(CASE WHEN ur.movement_type = 'issue' THEN ur.qty ELSE -ur.qty END) AS net_qty,
        max(ur.hm) FILTER (WHERE ur.movement_type = 'issue') AS raw_hm,
        max(ur.km) FILTER (WHERE ur.movement_type = 'issue') AS raw_km,
        count(DISTINCT ur.sparepart_id)
            FILTER (WHERE ur.movement_type = 'issue')::int AS sparepart_count,
        string_agg(DISTINCT sp.code, ', ' ORDER BY sp.code) AS sparepart_codes,
        string_agg(DISTINCT sp.name, ', ' ORDER BY sp.name) AS sparepart_names
    FROM usage_rows ur
    JOIN spareparts sp ON sp.id = ur.sparepart_id
    GROUP BY
        ur.unit_id,
        ur.wo_id,
        ur.component_id,
        CASE WHEN ur.component_id IS NULL THEN ur.sparepart_id END
    HAVING sum(CASE WHEN ur.movement_type = 'issue' THEN ur.qty ELSE -ur.qty END) > 0
),
event_reading AS (
    SELECT
        ce.*,
        COALESCE(
            ce.raw_hm,
            (
                SELECT r.value
                FROM unit_meter_readings r
                WHERE r.unit_id = ce.unit_id
                  AND r.reading_type = 'hm'
                  AND r.recorded_at <= ce.used_at
                ORDER BY r.recorded_at DESC, r.id DESC
                LIMIT 1
            )
        ) AS hm,
        COALESCE(
            ce.raw_km,
            (
                SELECT r.value
                FROM unit_meter_readings r
                WHERE r.unit_id = ce.unit_id
                  AND r.reading_type = 'km'
                  AND r.recorded_at <= ce.used_at
                ORDER BY r.recorded_at DESC, r.id DESC
                LIMIT 1
            )
        ) AS km
    FROM component_events ce
    WHERE ce.used_at IS NOT NULL
),
with_previous AS (
    SELECT
        er.*,
        row_number() OVER (
            PARTITION BY er.unit_id, COALESCE(er.component_id, -er.fallback_sparepart_id)
            ORDER BY er.used_at ASC, er.wo_id ASC
        ) AS replacement_no,
        lag(er.used_at) OVER (
            PARTITION BY er.unit_id, COALESCE(er.component_id, -er.fallback_sparepart_id)
            ORDER BY er.used_at ASC, er.wo_id ASC
        ) AS previous_used_at,
        lag(er.hm) OVER (
            PARTITION BY er.unit_id, COALESCE(er.component_id, -er.fallback_sparepart_id)
            ORDER BY er.used_at ASC, er.wo_id ASC
        ) AS previous_hm,
        lag(er.km) OVER (
            PARTITION BY er.unit_id, COALESCE(er.component_id, -er.fallback_sparepart_id)
            ORDER BY er.used_at ASC, er.wo_id ASC
        ) AS previous_km
    FROM event_reading er
)
SELECT
    wp.unit_id,
    u.code AS unit_code,
    wp.wo_id,
    wo.code AS wo_code,
    wo.title AS wo_title,
    wo.maintenance_type,
    wo.status AS wo_status,
    wo.closed_at,
    wp.component_id,
    COALESCE(c.code, '') AS component_code,
    COALESCE(c.name, wp.sparepart_names) AS component_name,
    (wp.component_id IS NULL) AS is_fallback,
    wp.fallback_sparepart_id,
    COALESCE(wp.sparepart_codes, '') AS sparepart_codes,
    COALESCE(wp.sparepart_names, '') AS sparepart_names,
    wp.sparepart_count,
    wp.net_qty::numeric(14, 3) AS qty,
    wp.used_at,
    wp.hm,
    wp.km,
    wp.replacement_no,
    wp.previous_used_at,
    CASE
        WHEN wp.previous_used_at IS NOT NULL THEN
            round(EXTRACT(EPOCH FROM (wp.used_at - wp.previous_used_at)) / 86400.0, 1)
        ELSE NULL
    END AS days_since_previous,
    CASE
        WHEN wp.previous_hm IS NOT NULL AND wp.hm IS NOT NULL THEN wp.hm - wp.previous_hm
        ELSE NULL
    END AS hm_since_previous,
    CASE
        WHEN wp.previous_km IS NOT NULL AND wp.km IS NOT NULL THEN wp.km - wp.previous_km
        ELSE NULL
    END AS km_since_previous
FROM with_previous wp
JOIN units u ON u.id = wp.unit_id
JOIN work_orders wo ON wo.id = wp.wo_id
LEFT JOIN components c ON c.id = wp.component_id;

-- Replacement frequency and interval per unit and component. The fallback
-- sparepart identity stays separate from a real component identity.
CREATE VIEW v_component_replacement_frequency AS
SELECT
    unit_id,
    unit_code,
    component_id,
    fallback_sparepart_id,
    component_code,
    component_name,
    is_fallback,
    count(*)::int AS replacement_count,
    min(used_at) AS first_replaced_at,
    max(used_at) AS last_replaced_at,
    round(avg(days_since_previous)::numeric, 1) AS avg_interval_days,
    round(avg(hm_since_previous)::numeric, 1) AS avg_interval_hm,
    round(avg(km_since_previous)::numeric, 1) AS avg_interval_km,
    sum(qty)::numeric(14, 3) AS total_qty,
    max(hm) AS last_hm,
    max(km) AS last_km
FROM v_unit_component_history
GROUP BY
    unit_id,
    unit_code,
    component_id,
    fallback_sparepart_id,
    component_code,
    component_name,
    is_fallback;

-- Completed downtime only. Inspection and cancelled work orders are excluded
-- by definition. Waiting-part time is derived from wo_updates and is reported
-- alongside, not added on top of, the work-order downtime interval.
CREATE VIEW v_unit_downtime AS
WITH waiting_rows AS (
    SELECT
        u.wo_id,
        u.created_at AS wait_start,
        lead(u.created_at) OVER (
            PARTITION BY u.wo_id
            ORDER BY u.created_at ASC, u.id ASC
        ) AS wait_end,
        u.status_to,
        wo.closed_at
    FROM wo_updates u
    JOIN work_orders wo ON wo.id = u.wo_id
),
waiting AS (
    SELECT
        wo_id,
        COALESCE(sum(
            GREATEST(
                0,
                EXTRACT(EPOCH FROM (
                    COALESCE(wait_end, closed_at) - wait_start
                )) / 60
            )
        ), 0)::numeric(14, 1) AS waiting_minutes
    FROM waiting_rows
    WHERE status_to = 'waiting_part'
    GROUP BY wo_id
)
SELECT
    wo.id AS wo_id,
    wo.unit_id,
    u.code AS unit_code,
    wo.code AS wo_code,
    wo.title,
    wo.maintenance_type,
    wo.priority,
    wo.assignee_id,
    COALESCE(assignee.name, '') AS assignee_name,
    wo.supervisor_id,
    COALESCE(supervisor.name, '') AS supervisor_name,
    wo.started_at,
    wo.closed_at AS downtime_end,
    CASE
        WHEN wo.maintenance_type = 'preventive' THEN wo.started_at
        ELSE COALESCE(wo.started_at, wo.created_at)
    END AS downtime_start,
    CASE
        WHEN wo.maintenance_type = 'preventive' THEN 'planned'
        ELSE 'unplanned'
    END AS downtime_class,
    CASE
        WHEN wo.maintenance_type = 'preventive' AND wo.started_at IS NULL THEN NULL
        ELSE GREATEST(
            0,
            EXTRACT(EPOCH FROM (wo.closed_at - COALESCE(wo.started_at, wo.created_at))) / 60
        )::numeric(14, 1)
    END AS downtime_minutes,
    COALESCE(waiting.waiting_minutes, 0) AS waiting_minutes
FROM work_orders wo
JOIN units u ON u.id = wo.unit_id
LEFT JOIN users assignee ON assignee.id = wo.assignee_id
LEFT JOIN users supervisor ON supervisor.id = wo.supervisor_id
LEFT JOIN waiting ON waiting.wo_id = wo.id
WHERE wo.status = 'done'
  AND wo.closed_at IS NOT NULL
  AND wo.maintenance_type IN ('preventive', 'corrective', 'breakdown');

CREATE INDEX IF NOT EXISTS idx_work_orders_unit_created_at
    ON work_orders (unit_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_work_orders_unit_closed_at
    ON work_orders (unit_id, closed_at DESC)
    WHERE closed_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_work_orders_maintenance_status
    ON work_orders (maintenance_type, status);
