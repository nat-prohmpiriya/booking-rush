-- 000013_update_superadmin_role.down.sql
-- Rollback: restore user to customer role

UPDATE users
SET role = 'customer'
WHERE email = 'superadmin@bookingrush.com';
