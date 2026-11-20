-- Milestone 1A - authentication/RBAC foundation.
--
-- M0 is locked; this migration only extends existing tables.
-- NIK is used as the local bridge between Odoo login and the local user row.
-- Reject duplicates before creating the unique expression index so the
-- migration fails with an actionable message instead of a generic index error.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM users
        WHERE nik IS NOT NULL
          AND btrim(nik) <> ''
        GROUP BY lower(btrim(nik))
        HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'users.nik contains duplicates after normalization; resolve duplicates before applying M1A';
    END IF;
END;
$$;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('admin', 'planner', 'mechanic', 'storekeeper', 'viewer'));

CREATE UNIQUE INDEX IF NOT EXISTS users_nik_normalized_uidx
    ON users (lower(btrim(nik)))
    WHERE nik IS NOT NULL AND btrim(nik) <> '';
