-- Milestone 5 - sparepart master, PM material template, and internal inventory.
-- Additive only: M0-M4 tables remain unchanged.
-- No price, cost, vendor, procurement, reservation, or material-readiness data is added here.

CREATE SEQUENCE inventory_transaction_code_seq START WITH 1;
CREATE SEQUENCE stock_opname_code_seq START WITH 1;

CREATE TABLE spareparts (
    id          bigserial PRIMARY KEY,
    code        text NOT NULL UNIQUE,
    name        text NOT NULL,
    category    text,
    uom         text NOT NULL DEFAULT 'pcs',
    description text,
    is_active   boolean NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_spareparts_name ON spareparts (lower(name));
CREATE INDEX idx_spareparts_category ON spareparts (category);
CREATE INDEX idx_spareparts_active ON spareparts (is_active);

CREATE TABLE sparepart_numbers (
    id           bigserial PRIMARY KEY,
    sparepart_id bigint NOT NULL REFERENCES spareparts(id) ON DELETE CASCADE,
    part_number  text NOT NULL,
    brand        text,
    number_type  text NOT NULL DEFAULT 'oem'
                 CHECK (number_type IN ('oem', 'aftermarket', 'superseded')),
    is_primary   boolean NOT NULL DEFAULT false,
    notes        text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE NULLS NOT DISTINCT (part_number, brand)
);

CREATE UNIQUE INDEX sparepart_primary_number_uidx
    ON sparepart_numbers (sparepart_id)
    WHERE is_primary;
CREATE INDEX idx_sparepart_numbers_part ON sparepart_numbers (lower(part_number));
CREATE INDEX idx_sparepart_numbers_sparepart ON sparepart_numbers (sparepart_id);

CREATE TABLE sparepart_alternatives (
    id                       bigserial PRIMARY KEY,
    sparepart_id             bigint NOT NULL REFERENCES spareparts(id),
    alternative_sparepart_id bigint NOT NULL REFERENCES spareparts(id),
    relation_type            text NOT NULL DEFAULT 'equivalent'
                             CHECK (relation_type IN ('equivalent', 'superseded', 'substitute')),
    verification_status      text NOT NULL DEFAULT 'pending_verification'
                             CHECK (verification_status IN
                               ('pending_verification', 'approved_compatible',
                                'conditional_compatible', 'not_compatible')),
    notes                    text,
    verified_by              bigint REFERENCES users(id),
    verified_at              timestamptz,
    created_at               timestamptz NOT NULL DEFAULT now(),
    updated_at               timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT sparepart_alternatives_not_self CHECK (sparepart_id <> alternative_sparepart_id),
    CONSTRAINT sparepart_alternatives_verified CHECK (
        verification_status = 'pending_verification'
        OR (verified_by IS NOT NULL AND verified_at IS NOT NULL)
    ),
    CONSTRAINT sparepart_alternatives_conditional_note CHECK (
        verification_status <> 'conditional_compatible' OR COALESCE(notes, '') <> ''
    ),
    UNIQUE (sparepart_id, alternative_sparepart_id, relation_type)
);

CREATE INDEX idx_sparepart_alternatives_alt ON sparepart_alternatives (alternative_sparepart_id);
CREATE INDEX idx_sparepart_alternatives_status ON sparepart_alternatives (verification_status);

CREATE TABLE sparepart_compatibility (
    id              bigserial PRIMARY KEY,
    sparepart_id    bigint NOT NULL REFERENCES spareparts(id) ON DELETE CASCADE,
    unit_model_id   bigint REFERENCES unit_models(id),
    engine_model_id bigint REFERENCES engine_models(id),
    configuration   text,
    status          text NOT NULL DEFAULT 'pending_verification'
                    CHECK (status IN ('pending_verification', 'approved_compatible',
                                      'conditional_compatible', 'not_compatible')),
    condition_note  text,
    source          text,
    notes           text,
    verified_by     bigint REFERENCES users(id),
    verified_at     timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT sparepart_compatibility_target CHECK (
        unit_model_id IS NOT NULL OR engine_model_id IS NOT NULL OR COALESCE(configuration, '') <> ''
    ),
    CONSTRAINT sparepart_compatibility_conditional_note CHECK (
        status <> 'conditional_compatible' OR COALESCE(condition_note, '') <> ''
    ),
    CONSTRAINT sparepart_compatibility_verified CHECK (
        status = 'pending_verification' OR (verified_by IS NOT NULL AND verified_at IS NOT NULL)
    ),
    UNIQUE NULLS NOT DISTINCT (sparepart_id, unit_model_id, engine_model_id, configuration)
);

CREATE INDEX idx_sparepart_compat_unit_model ON sparepart_compatibility (unit_model_id, status);
CREATE INDEX idx_sparepart_compat_engine_model ON sparepart_compatibility (engine_model_id, status);
CREATE INDEX idx_sparepart_compat_sparepart ON sparepart_compatibility (sparepart_id, status);

CREATE TABLE pm_plan_materials (
    id           bigserial PRIMARY KEY,
    pm_plan_id   bigint NOT NULL REFERENCES pm_plans(id) ON DELETE CASCADE,
    sparepart_id bigint NOT NULL REFERENCES spareparts(id),
    qty          numeric(14,3) NOT NULL CHECK (qty > 0),
    is_optional  boolean NOT NULL DEFAULT false,
    notes        text,
    UNIQUE (pm_plan_id, sparepart_id)
);

CREATE INDEX idx_pm_plan_materials_plan ON pm_plan_materials (pm_plan_id);
CREATE INDEX idx_pm_plan_materials_sparepart ON pm_plan_materials (sparepart_id);

CREATE TABLE warehouses (
    id            bigserial PRIMARY KEY,
    code          text NOT NULL UNIQUE,
    name          text NOT NULL,
    site_location text,
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_warehouses_active ON warehouses (is_active);

CREATE TABLE warehouse_locations (
    id            bigserial PRIMARY KEY,
    warehouse_id  bigint NOT NULL REFERENCES warehouses(id),
    code          text NOT NULL,
    name          text NOT NULL,
    location_type text NOT NULL DEFAULT 'bin'
                  CHECK (location_type IN ('rack', 'bin', 'yard', 'staging')),
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (warehouse_id, code)
);

CREATE INDEX idx_warehouse_locations_warehouse ON warehouse_locations (warehouse_id, is_active);

CREATE TABLE inventory_stock_policies (
    id                    bigserial PRIMARY KEY,
    sparepart_id          bigint NOT NULL REFERENCES spareparts(id),
    warehouse_location_id bigint NOT NULL REFERENCES warehouse_locations(id),
    min_qty               numeric(14,3) NOT NULL DEFAULT 0 CHECK (min_qty >= 0),
    critical_qty          numeric(14,3) NOT NULL DEFAULT 0 CHECK (critical_qty >= 0),
    reorder_qty           numeric(14,3) CHECK (reorder_qty IS NULL OR reorder_qty >= 0),
    is_active             boolean NOT NULL DEFAULT true,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    UNIQUE (sparepart_id, warehouse_location_id),
    CHECK (critical_qty <= min_qty)
);

CREATE INDEX idx_inventory_stock_policies_location
    ON inventory_stock_policies (warehouse_location_id, is_active);

CREATE TABLE inventory_transactions (
    id              bigserial PRIMARY KEY,
    code            text NOT NULL UNIQUE,
    type            text NOT NULL CHECK (type IN
                      ('receipt', 'issue', 'return', 'reserve', 'release',
                       'transfer', 'adjustment', 'opname')),
    status          text NOT NULL DEFAULT 'posted'
                    CHECK (status IN ('posted', 'reversed')),
    reference_type  text,
    reference_id    bigint,
    reference_no    text,
    reversal_of_id  bigint REFERENCES inventory_transactions(id),
    idempotency_key text,
    posted_by       bigint REFERENCES users(id),
    posted_at       timestamptz NOT NULL DEFAULT now(),
    notes           text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    CHECK (reversal_of_id IS NULL OR reversal_of_id <> id)
);

CREATE UNIQUE INDEX uq_inventory_transactions_idempotency
    ON inventory_transactions (idempotency_key)
    WHERE idempotency_key IS NOT NULL;
CREATE UNIQUE INDEX uq_inventory_transactions_reversal
    ON inventory_transactions (reversal_of_id)
    WHERE reversal_of_id IS NOT NULL;
CREATE INDEX idx_inventory_transactions_posted ON inventory_transactions (posted_at DESC);
CREATE INDEX idx_inventory_transactions_type_posted ON inventory_transactions (type, posted_at DESC);
CREATE INDEX idx_inventory_transactions_ref ON inventory_transactions (reference_type, reference_id);

CREATE TABLE inventory_transaction_items (
    id               bigserial PRIMARY KEY,
    transaction_id   bigint NOT NULL REFERENCES inventory_transactions(id) ON DELETE RESTRICT,
    sparepart_id     bigint NOT NULL REFERENCES spareparts(id),
    from_location_id bigint REFERENCES warehouse_locations(id),
    to_location_id   bigint REFERENCES warehouse_locations(id),
    qty              numeric(14,3) NOT NULL CHECK (qty > 0),
    notes            text,
    CONSTRAINT inventory_transaction_items_location CHECK (
        from_location_id IS NOT NULL OR to_location_id IS NOT NULL
    ),
    CONSTRAINT inventory_transaction_items_distinct_locations CHECK (
        from_location_id IS NULL OR to_location_id IS NULL OR from_location_id <> to_location_id
    )
);

CREATE INDEX idx_inventory_items_transaction ON inventory_transaction_items (transaction_id);
CREATE INDEX idx_inventory_items_sparepart ON inventory_transaction_items (sparepart_id);
CREATE INDEX idx_inventory_items_from ON inventory_transaction_items (from_location_id);
CREATE INDEX idx_inventory_items_to ON inventory_transaction_items (to_location_id);

CREATE TABLE inventory_balances (
    sparepart_id          bigint NOT NULL REFERENCES spareparts(id),
    warehouse_location_id bigint NOT NULL REFERENCES warehouse_locations(id),
    on_hand               numeric(14,3) NOT NULL DEFAULT 0 CHECK (on_hand >= 0),
    reserved              numeric(14,3) NOT NULL DEFAULT 0 CHECK (reserved >= 0),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (sparepart_id, warehouse_location_id),
    CHECK (reserved <= on_hand)
);

CREATE INDEX idx_inventory_balances_location ON inventory_balances (warehouse_location_id);

CREATE TABLE stock_opname_sessions (
    id            bigserial PRIMARY KEY,
    code          text NOT NULL UNIQUE,
    warehouse_id  bigint NOT NULL REFERENCES warehouses(id),
    status        text NOT NULL DEFAULT 'draft'
                  CHECK (status IN ('draft', 'counting', 'review', 'posted', 'cancelled')),
    started_at    timestamptz,
    posted_at     timestamptz,
    posted_by     bigint REFERENCES users(id),
    notes         text,
    cancel_reason text,
    cancelled_by  bigint REFERENCES users(id),
    cancelled_at  timestamptz,
    created_by    bigint REFERENCES users(id),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_stock_opname_sessions_status ON stock_opname_sessions (status, created_at DESC);
CREATE INDEX idx_stock_opname_sessions_warehouse ON stock_opname_sessions (warehouse_id);

CREATE TABLE stock_opname_items (
    id           bigserial PRIMARY KEY,
    session_id   bigint NOT NULL REFERENCES stock_opname_sessions(id) ON DELETE CASCADE,
    sparepart_id bigint NOT NULL REFERENCES spareparts(id),
    location_id  bigint NOT NULL REFERENCES warehouse_locations(id),
    system_qty   numeric(14,3) NOT NULL CHECK (system_qty >= 0),
    counted_qty  numeric(14,3) CHECK (counted_qty IS NULL OR counted_qty >= 0),
    difference   numeric(14,3) GENERATED ALWAYS AS (counted_qty - system_qty) STORED,
    notes        text,
    UNIQUE (session_id, sparepart_id, location_id)
);

CREATE INDEX idx_stock_opname_items_session ON stock_opname_items (session_id);

CREATE VIEW v_stock_balance AS
WITH stock_rows AS (
    SELECT b.sparepart_id,
           b.warehouse_location_id,
           b.on_hand,
           b.reserved,
           b.updated_at AS balance_updated_at
    FROM inventory_balances b
    UNION ALL
    SELECT p.sparepart_id,
           p.warehouse_location_id,
           0::numeric(14,3),
           0::numeric(14,3),
           NULL::timestamptz
    FROM inventory_stock_policies p
    WHERE NOT EXISTS (
        SELECT 1
        FROM inventory_balances b
        WHERE b.sparepart_id = p.sparepart_id
          AND b.warehouse_location_id = p.warehouse_location_id
    )
)
SELECT r.sparepart_id,
       s.code AS sparepart_code,
       s.name AS sparepart_name,
       COALESCE(s.category, '') AS category,
       s.uom,
       s.is_active AS sparepart_active,
       r.warehouse_location_id,
       w.id AS warehouse_id,
       w.code AS warehouse_code,
       w.name AS warehouse_name,
       wl.code AS location_code,
       wl.name AS location_name,
       wl.location_type,
       r.on_hand,
       r.reserved,
       (r.on_hand - r.reserved) AS available,
       p.id AS policy_id,
       COALESCE(p.min_qty, 0) AS min_qty,
       COALESCE(p.critical_qty, 0) AS critical_qty,
       p.reorder_qty,
       CASE
           WHEN p.id IS NULL OR p.is_active = false THEN 'no_policy'
           WHEN (r.on_hand - r.reserved) <= p.critical_qty THEN 'critical'
           WHEN (r.on_hand - r.reserved) < p.min_qty THEN 'low'
           ELSE 'ok'
       END AS stock_status,
       r.balance_updated_at
FROM stock_rows r
JOIN spareparts s ON s.id = r.sparepart_id
JOIN warehouse_locations wl ON wl.id = r.warehouse_location_id
JOIN warehouses w ON w.id = wl.warehouse_id
LEFT JOIN inventory_stock_policies p
       ON p.sparepart_id = r.sparepart_id
      AND p.warehouse_location_id = r.warehouse_location_id;

CREATE VIEW v_low_stock AS
SELECT *
FROM v_stock_balance
WHERE stock_status IN ('low', 'critical')
  AND policy_id IS NOT NULL;
