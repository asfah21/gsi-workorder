-- Milestone 8B - component master extensions, per-unit component installations,
-- and component lifetime view. Additive only: existing component and unit rows
-- are preserved.

ALTER TABLE components
    ADD COLUMN component_class text,
    ADD COLUMN default_lifetime_hm numeric(12,1),
    ADD COLUMN default_lifetime_days integer,
    ADD COLUMN default_lifetime_km numeric(12,1);

ALTER TABLE components
    ADD CONSTRAINT components_component_class_check
        CHECK (component_class IS NULL OR component_class IN ('major', 'minor')),
    ADD CONSTRAINT components_default_lifetime_hm_check
        CHECK (default_lifetime_hm IS NULL OR default_lifetime_hm >= 0),
    ADD CONSTRAINT components_default_lifetime_days_check
        CHECK (default_lifetime_days IS NULL OR default_lifetime_days >= 0),
    ADD CONSTRAINT components_default_lifetime_km_check
        CHECK (default_lifetime_km IS NULL OR default_lifetime_km >= 0);

CREATE TABLE unit_component_installations (
    id             bigserial PRIMARY KEY,
    unit_id        bigint NOT NULL REFERENCES units(id),
    component_id   bigint NOT NULL REFERENCES components(id),
    serial_number  text NOT NULL CHECK (btrim(serial_number) <> ''),
    installed_at   timestamptz NOT NULL,
    installed_hm   numeric(12,1) NOT NULL DEFAULT 0 CHECK (installed_hm >= 0),
    installed_km   numeric(12,1) NOT NULL DEFAULT 0 CHECK (installed_km >= 0),
    removed_at     timestamptz,
    removed_hm     numeric(12,1),
    removed_km     numeric(12,1),
    lifetime_hm    numeric(12,1),
    lifetime_days  integer,
    lifetime_km    numeric(12,1),
    status         text NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'removed', 'transferred', 'scrapped')),
    source_wo_id   bigint REFERENCES work_orders(id) ON DELETE SET NULL,
    notes          text,
    created_by     bigint REFERENCES users(id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT unit_component_installations_removed_at_check
        CHECK (removed_at IS NULL OR removed_at >= installed_at),
    CONSTRAINT unit_component_installations_removed_hm_check
        CHECK (removed_hm IS NULL OR removed_hm >= installed_hm),
    CONSTRAINT unit_component_installations_removed_km_check
        CHECK (removed_km IS NULL OR removed_km >= installed_km),
    CONSTRAINT unit_component_installations_lifetime_hm_check
        CHECK (lifetime_hm IS NULL OR lifetime_hm >= 0),
    CONSTRAINT unit_component_installations_lifetime_days_check
        CHECK (lifetime_days IS NULL OR lifetime_days >= 0),
    CONSTRAINT unit_component_installations_lifetime_km_check
        CHECK (lifetime_km IS NULL OR lifetime_km >= 0),
    CONSTRAINT unit_component_installations_status_dates_check
        CHECK (
            (status = 'active' AND removed_at IS NULL)
            OR (status <> 'active' AND removed_at IS NOT NULL)
        )
);

-- A unit can only have one active installation for a component type. This is
-- the M8 default; multi-instance component types can be split into distinct
-- component codes if PAM later confirms they are needed.
CREATE UNIQUE INDEX uq_unit_component_installations_active
    ON unit_component_installations (unit_id, component_id)
    WHERE status = 'active';

-- A physical component serial cannot be active on two units at the same time.
CREATE UNIQUE INDEX uq_unit_component_installations_active_serial
    ON unit_component_installations (lower(btrim(serial_number)))
    WHERE status = 'active';

CREATE INDEX idx_unit_component_installations_unit
    ON unit_component_installations (unit_id, status, component_id);
CREATE INDEX idx_unit_component_installations_component
    ON unit_component_installations (component_id, status);
CREATE INDEX idx_unit_component_installations_serial
    ON unit_component_installations (lower(btrim(serial_number)));

-- Lifetime is computed from the current unit HM/KM for active installations and
-- from the removal reading for closed installations. Remaining values can be
-- negative so the UI can show an overdue amount instead of silently clamping it.
CREATE VIEW v_unit_component_lifetime AS
SELECT
    i.id AS installation_id,
    i.unit_id,
    u.code AS unit_code,
    u.current_hm AS unit_current_hm,
    u.current_km AS unit_current_km,
    i.component_id,
    c.code AS component_code,
    c.name AS component_name,
    COALESCE(c.category, '') AS category,
    COALESCE(c.component_class, '') AS component_class,
    i.serial_number,
    i.installed_at,
    i.installed_hm,
    i.installed_km,
    i.removed_at,
    i.removed_hm,
    i.removed_km,
    i.status,
    COALESCE(i.lifetime_hm, c.default_lifetime_hm) AS lifetime_hm,
    COALESCE(i.lifetime_days, c.default_lifetime_days) AS lifetime_days,
    COALESCE(i.lifetime_km, c.default_lifetime_km) AS lifetime_km,
    used.used_hm,
    used.used_km,
    used.used_days,
    CASE
        WHEN COALESCE(i.lifetime_hm, c.default_lifetime_hm) IS NULL THEN NULL
        ELSE COALESCE(i.lifetime_hm, c.default_lifetime_hm) - used.used_hm
    END AS remaining_hm,
    CASE
        WHEN COALESCE(i.lifetime_days, c.default_lifetime_days) IS NULL THEN NULL
        ELSE COALESCE(i.lifetime_days, c.default_lifetime_days) - used.used_days
    END AS remaining_days,
    CASE
        WHEN COALESCE(i.lifetime_km, c.default_lifetime_km) IS NULL THEN NULL
        ELSE COALESCE(i.lifetime_km, c.default_lifetime_km) - used.used_km
    END AS remaining_km
FROM unit_component_installations i
JOIN units u ON u.id = i.unit_id
JOIN components c ON c.id = i.component_id
CROSS JOIN LATERAL (
    SELECT
        GREATEST(
            CASE
                WHEN i.status = 'active' THEN u.current_hm
                ELSE COALESCE(i.removed_hm, u.current_hm)
            END - i.installed_hm,
            0
        )::numeric(12,1) AS used_hm,
        GREATEST(
            CASE
                WHEN i.status = 'active' THEN u.current_km
                ELSE COALESCE(i.removed_km, u.current_km)
            END - i.installed_km,
            0
        )::numeric(12,1) AS used_km,
        GREATEST(
            EXTRACT(EPOCH FROM (COALESCE(i.removed_at, now()) - i.installed_at)) / 86400,
            0
        )::numeric(12,1) AS used_days
) used;
