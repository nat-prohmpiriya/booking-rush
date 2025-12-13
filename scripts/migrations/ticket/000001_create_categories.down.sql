-- 000001_create_categories.down.sql
DROP TRIGGER IF EXISTS update_categories_updated_at ON categories;
DROP TABLE IF EXISTS categories;
DROP FUNCTION IF EXISTS update_updated_at_column();
