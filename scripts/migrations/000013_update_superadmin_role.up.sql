-- 000013_update_superadmin_role.up.sql
-- Temporary migration to update test user to super_admin role

UPDATE users
SET role = 'super_admin'
WHERE email = 'superadmin@bookingrush.com';
