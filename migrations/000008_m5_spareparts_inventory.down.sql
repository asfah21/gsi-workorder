DROP VIEW IF EXISTS v_low_stock;
DROP VIEW IF EXISTS v_stock_balance;

DROP TABLE IF EXISTS stock_opname_items;
DROP TABLE IF EXISTS stock_opname_sessions;
DROP TABLE IF EXISTS inventory_balances;
DROP TABLE IF EXISTS inventory_transaction_items;
DROP TABLE IF EXISTS inventory_transactions;
DROP TABLE IF EXISTS inventory_stock_policies;
DROP TABLE IF EXISTS warehouse_locations;
DROP TABLE IF EXISTS warehouses;
DROP TABLE IF EXISTS pm_plan_materials;
DROP TABLE IF EXISTS sparepart_compatibility;
DROP TABLE IF EXISTS sparepart_alternatives;
DROP TABLE IF EXISTS sparepart_numbers;
DROP TABLE IF EXISTS spareparts;

DROP SEQUENCE IF EXISTS stock_opname_code_seq;
DROP SEQUENCE IF EXISTS inventory_transaction_code_seq;
