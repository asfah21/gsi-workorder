-- Reverse M1A. Storekeeper rows must be mapped back to an allowed M0 role
-- before the narrower M0 check constraint can be restored.

DROP INDEX IF EXISTS users_nik_normalized_uidx;

UPDATE users SET role = 'mechanic' WHERE role = 'storekeeper';

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('admin', 'planner', 'mechanic', 'viewer'));
