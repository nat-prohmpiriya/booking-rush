-- 000001_create_tenants.down.sql
DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;
DROP TABLE IF EXISTS tenants;
DROP FUNCTION IF EXISTS update_updated_at_column();
