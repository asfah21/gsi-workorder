DROP VIEW IF EXISTS v_shortage_aging;
DROP VIEW IF EXISTS v_material_readiness;

DROP TABLE IF EXISTS work_order_sparepart_usage;
DROP INDEX IF EXISTS idx_inventory_items_reservation;
ALTER TABLE inventory_transaction_items DROP COLUMN IF EXISTS reservation_id;
DROP TABLE IF EXISTS inventory_reservations;
DROP TABLE IF EXISTS material_requirement_updates;
DROP TABLE IF EXISTS material_requirement_items;
DROP TABLE IF EXISTS material_requirements;
DROP SEQUENCE IF EXISTS material_requirement_code_seq;
