-- Milestone 6 - material requirement, readiness, reservation, and usage.
-- Additive only: M0-M5 tables remain unchanged except the new nullable
-- inventory_transaction_items.reservation_id link.

CREATE SEQUENCE material_requirement_code_seq START WITH 1;

CREATE TABLE material_requirements (
    id                bigserial PRIMARY KEY,
    code              text NOT NULL UNIQUE,
    wo_id             bigint NOT NULL REFERENCES work_orders(id),
    source            text NOT NULL DEFAULT 'manual'
                      CHECK (source IN ('manual', 'pm')),
    status            text NOT NULL DEFAULT 'draft'
                      CHECK (status IN ('draft', 'planned', 'checking_stock',
                                        'ready', 'partially_ready', 'shortage',
                                        'reserved', 'issued', 'completed', 'cancelled')),
    status_changed_at timestamptz NOT NULL DEFAULT now(),
    shortage_since    timestamptz,
    requested_by      bigint REFERENCES users(id),
    requested_at      timestamptz,
    notes             text,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uq_material_requirements_non_cancelled_wo
    ON material_requirements (wo_id)
    WHERE status <> 'cancelled';
CREATE INDEX idx_material_requirements_status
    ON material_requirements (status, status_changed_at);
CREATE INDEX idx_material_requirements_shortage
    ON material_requirements (shortage_since)
    WHERE shortage_since IS NOT NULL;
CREATE INDEX idx_material_requirements_wo
    ON material_requirements (wo_id);

CREATE TABLE material_requirement_items (
    id                      bigserial PRIMARY KEY,
    material_requirement_id bigint NOT NULL
                            REFERENCES material_requirements(id) ON DELETE CASCADE,
    sparepart_id            bigint NOT NULL REFERENCES spareparts(id),
    qty_required            numeric(14,3) NOT NULL CHECK (qty_required > 0),
    qty_reserved            numeric(14,3) NOT NULL DEFAULT 0 CHECK (qty_reserved >= 0),
    qty_issued              numeric(14,3) NOT NULL DEFAULT 0 CHECK (qty_issued >= 0),
    qty_returned            numeric(14,3) NOT NULL DEFAULT 0 CHECK (qty_returned >= 0),
    is_optional             boolean NOT NULL DEFAULT false,
    preferred_warehouse_id  bigint REFERENCES warehouses(id),
    notes                   text,
    UNIQUE (material_requirement_id, sparepart_id),
    CHECK (qty_reserved <= qty_required),
    CHECK (qty_returned <= qty_issued),
    CHECK (qty_issued - qty_returned <= qty_required),
    CHECK (qty_reserved + (qty_issued - qty_returned) <= qty_required)
);

CREATE INDEX idx_material_items_sparepart
    ON material_requirement_items (sparepart_id);
CREATE INDEX idx_material_items_requirement
    ON material_requirement_items (material_requirement_id);

CREATE TABLE material_requirement_updates (
    id                      bigserial PRIMARY KEY,
    material_requirement_id bigint NOT NULL
                            REFERENCES material_requirements(id) ON DELETE RESTRICT,
    user_id                 bigint REFERENCES users(id),
    status_from             text,
    status_to               text NOT NULL,
    note                    text,
    created_at              timestamptz NOT NULL DEFAULT now(),
    CHECK (status_from IS NULL OR status_from IN ('draft', 'planned', 'checking_stock',
           'ready', 'partially_ready', 'shortage', 'reserved', 'issued',
           'completed', 'cancelled')),
    CHECK (status_to IN ('draft', 'planned', 'checking_stock',
           'ready', 'partially_ready', 'shortage', 'reserved', 'issued',
           'completed', 'cancelled'))
);

CREATE INDEX idx_material_updates_requirement
    ON material_requirement_updates (material_requirement_id, created_at DESC);

CREATE TABLE inventory_reservations (
    id                           bigserial PRIMARY KEY,
    material_requirement_item_id bigint NOT NULL
                                 REFERENCES material_requirement_items(id) ON DELETE RESTRICT,
    sparepart_id                 bigint NOT NULL REFERENCES spareparts(id),
    warehouse_location_id        bigint NOT NULL REFERENCES warehouse_locations(id),
    qty                          numeric(14,3) NOT NULL CHECK (qty >= 0),
    status                       text NOT NULL DEFAULT 'active'
                                 CHECK (status IN ('active', 'consumed', 'released', 'cancelled')),
    reserved_by                  bigint REFERENCES users(id),
    reserved_at                  timestamptz NOT NULL DEFAULT now(),
    released_at                  timestamptz,
    consumed_at                  timestamptz,
    notes                        text,
    CHECK ((status = 'active' AND qty > 0) OR (status <> 'active' AND qty = 0))
);

CREATE UNIQUE INDEX uq_inventory_reservations_active
    ON inventory_reservations (material_requirement_item_id, warehouse_location_id)
    WHERE status = 'active';
CREATE INDEX idx_inventory_reservations_stock
    ON inventory_reservations (sparepart_id, warehouse_location_id, status);
CREATE INDEX idx_inventory_reservations_requirement
    ON inventory_reservations (material_requirement_item_id, status);

ALTER TABLE inventory_transaction_items
    ADD COLUMN reservation_id bigint REFERENCES inventory_reservations(id);

CREATE INDEX idx_inventory_items_reservation
    ON inventory_transaction_items (reservation_id);

CREATE TABLE work_order_sparepart_usage (
    id                            bigserial PRIMARY KEY,
    wo_id                         bigint NOT NULL REFERENCES work_orders(id) ON DELETE RESTRICT,
    material_requirement_item_id  bigint REFERENCES material_requirement_items(id),
    sparepart_id                  bigint NOT NULL REFERENCES spareparts(id),
    component_id                  bigint REFERENCES components(id),
    warehouse_location_id         bigint NOT NULL REFERENCES warehouse_locations(id),
    qty                           numeric(14,3) NOT NULL CHECK (qty > 0),
    hm                            numeric(12,1) CHECK (hm IS NULL OR hm >= 0),
    km                            numeric(12,1) CHECK (km IS NULL OR km >= 0),
    movement_type                 text NOT NULL CHECK (movement_type IN ('issue', 'return')),
    inventory_transaction_item_id bigint NOT NULL
                                  REFERENCES inventory_transaction_items(id),
    return_of_usage_id            bigint REFERENCES work_order_sparepart_usage(id),
    used_by                       bigint REFERENCES users(id),
    used_at                       timestamptz NOT NULL DEFAULT now(),
    notes                         text,
    UNIQUE (inventory_transaction_item_id),
    CHECK (return_of_usage_id IS NULL OR return_of_usage_id <> id)
);

CREATE INDEX idx_wo_usage_wo ON work_order_sparepart_usage (wo_id);
CREATE INDEX idx_wo_usage_unit_time
    ON work_order_sparepart_usage (sparepart_id, used_at DESC);
CREATE INDEX idx_wo_usage_component
    ON work_order_sparepart_usage (component_id, used_at DESC);
CREATE INDEX idx_wo_usage_return_of
    ON work_order_sparepart_usage (return_of_usage_id)
    WHERE return_of_usage_id IS NOT NULL;

CREATE VIEW v_material_readiness AS
WITH item_availability AS (
    SELECT mri.id AS item_id,
           mri.material_requirement_id,
           mri.sparepart_id,
           mri.qty_required,
           mri.qty_reserved,
           mri.qty_issued,
           mri.qty_returned,
           (mri.qty_issued - mri.qty_returned) AS net_issued,
           mri.is_optional,
           mri.preferred_warehouse_id,
           COALESCE(mri.notes, '') AS notes,
           COALESCE(a.available, 0)::numeric(14,3) AS available
    FROM material_requirement_items mri
    LEFT JOIN LATERAL (
        SELECT SUM(v.available)::numeric(14,3) AS available
        FROM v_stock_balance v
        WHERE v.sparepart_id = mri.sparepart_id
          AND (mri.preferred_warehouse_id IS NULL OR v.warehouse_id = mri.preferred_warehouse_id)
    ) a ON true
),
item_calc AS (
    SELECT ia.*,
           (ia.available + ia.qty_reserved + ia.net_issued)::numeric(14,3) AS coverage,
           GREATEST(ia.qty_required - (ia.available + ia.qty_reserved + ia.net_issued), 0)::numeric(14,3) AS shortage
    FROM item_availability ia
),
item_agg AS (
    SELECT material_requirement_id,
           count(*) FILTER (WHERE NOT is_optional) AS required_count,
           count(*) FILTER (WHERE NOT is_optional AND coverage <= 0) AS zero_coverage_count,
           count(*) FILTER (WHERE NOT is_optional AND coverage > 0 AND coverage < qty_required) AS partial_count,
           count(*) FILTER (WHERE NOT is_optional AND coverage >= qty_required) AS full_count,
           COALESCE(sum(qty_reserved), 0)::numeric(14,3) AS reserved_total,
           COALESCE(sum(net_issued), 0)::numeric(14,3) AS issued_total,
           COALESCE(bool_or(NOT is_optional AND coverage <= 0), false) AS has_zero_required
    FROM item_calc
    GROUP BY material_requirement_id
)
SELECT mr.id AS material_requirement_id,
       mr.code AS requirement_code,
       mr.wo_id,
       mr.status AS requirement_status,
       mr.shortage_since,
       mr.source,
       mr.requested_by,
       mr.requested_at,
       COALESCE(mr.notes, '') AS requirement_notes,
       ic.item_id,
       ic.sparepart_id,
       s.code AS sparepart_code,
       s.name AS sparepart_name,
       s.uom,
       ic.qty_required::float8,
       ic.qty_reserved::float8,
       ic.qty_issued::float8,
       ic.qty_returned::float8,
       ic.net_issued::float8,
       ic.available::float8,
       ic.coverage::float8,
       ic.shortage::float8,
       ic.is_optional,
       ic.preferred_warehouse_id,
       COALESCE(w.code, '') AS preferred_warehouse_code,
       ic.notes,
       CASE
           WHEN mr.status = 'cancelled' THEN 'cancelled'
           WHEN mr.status = 'completed' THEN 'completed'
           WHEN a.issued_total > 0 AND NOT a.has_zero_required THEN 'issued'
           WHEN a.reserved_total > 0 AND NOT a.has_zero_required AND a.partial_count = 0 THEN 'reserved'
           WHEN a.has_zero_required THEN 'shortage'
           WHEN a.required_count = 0 THEN
               CASE WHEN mr.status IN ('draft', 'planned') THEN mr.status ELSE 'ready' END
           WHEN a.partial_count > 0 THEN 'partially_ready'
           WHEN a.full_count = a.required_count THEN 'ready'
           ELSE mr.status
       END AS derived_status
FROM material_requirements mr
JOIN item_calc ic ON ic.material_requirement_id = mr.id
JOIN item_agg a ON a.material_requirement_id = mr.id
JOIN spareparts s ON s.id = ic.sparepart_id
LEFT JOIN warehouses w ON w.id = ic.preferred_warehouse_id;

CREATE VIEW v_shortage_aging AS
SELECT mr.id AS material_requirement_id,
       mr.code AS requirement_code,
       mr.wo_id,
       wo.code AS wo_code,
       wo.title AS wo_title,
       wo.unit_id,
       COALESCE(u.code, '') AS unit_code,
       mr.shortage_since,
       r.item_id,
       r.sparepart_id,
       r.sparepart_code,
       r.sparepart_name,
       r.uom,
       r.qty_required,
       r.coverage,
       r.shortage,
       GREATEST(
           (now() AT TIME ZONE 'Asia/Makassar')::date
           - (mr.shortage_since AT TIME ZONE 'Asia/Makassar')::date,
           0
       ) AS age_days,
       CASE
           WHEN (now() AT TIME ZONE 'Asia/Makassar')::date
                - (mr.shortage_since AT TIME ZONE 'Asia/Makassar')::date > 14 THEN '>14'
           WHEN (now() AT TIME ZONE 'Asia/Makassar')::date
                - (mr.shortage_since AT TIME ZONE 'Asia/Makassar')::date > 7 THEN '8-14'
           WHEN (now() AT TIME ZONE 'Asia/Makassar')::date
                - (mr.shortage_since AT TIME ZONE 'Asia/Makassar')::date > 3 THEN '4-7'
           ELSE '0-3'
       END AS age_bucket
FROM material_requirements mr
JOIN work_orders wo ON wo.id = mr.wo_id
LEFT JOIN units u ON u.id = wo.unit_id
JOIN v_material_readiness r ON r.material_requirement_id = mr.id
WHERE mr.status = 'shortage'
  AND mr.shortage_since IS NOT NULL
  AND r.shortage > 0
  AND NOT r.is_optional;
