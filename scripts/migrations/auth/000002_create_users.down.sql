-- 000002_create_users.down.sql
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP INDEX IF EXISTS idx_users_email_with_tenant;
DROP INDEX IF EXISTS idx_users_email_null_tenant;
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS user_role;
