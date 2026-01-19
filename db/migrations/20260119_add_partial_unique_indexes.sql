-- Migration: Add partial unique indexes for soft delete compatibility
-- Date: 2026-01-19
-- Description: Replace standard unique indexes with partial unique indexes
--              to allow reusing values after soft delete

-- ============================================================
-- STORES TABLE
-- ============================================================

-- Drop existing unique constraints if they exist
DROP INDEX IF EXISTS stores_business_number_key;
DROP INDEX IF EXISTS idx_stores_business_number;
DROP INDEX IF EXISTS stores_slug_key;
DROP INDEX IF EXISTS idx_stores_slug;

-- Create partial unique indexes (only for non-deleted rows)
CREATE UNIQUE INDEX idx_stores_business_number
ON stores(business_number)
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX idx_stores_slug
ON stores(slug)
WHERE deleted_at IS NULL;

-- Optional: Create indexes for deleted rows (if you need to query them efficiently)
-- CREATE INDEX idx_stores_deleted_business_number
-- ON stores(business_number)
-- WHERE deleted_at IS NOT NULL;

-- ============================================================
-- USERS TABLE
-- ============================================================

-- Drop existing unique constraints if they exist
DROP INDEX IF EXISTS users_email_key;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS users_nickname_key;
DROP INDEX IF EXISTS idx_users_nickname;

-- Create partial unique indexes (only for non-deleted rows)
CREATE UNIQUE INDEX idx_users_email
ON users(email)
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX idx_users_nickname
ON users(nickname)
WHERE deleted_at IS NULL;

-- ============================================================
-- BUSINESS_REGISTRATIONS TABLE
-- ============================================================

-- Drop existing unique constraints if they exist
DROP INDEX IF EXISTS business_registrations_store_id_key;
DROP INDEX IF EXISTS idx_business_registrations_store_id;

-- Create partial unique index (only for non-deleted rows)
CREATE UNIQUE INDEX idx_business_registrations_store_id
ON business_registrations(store_id)
WHERE deleted_at IS NULL;

-- ============================================================
-- VERIFICATION QUERIES (Run these to verify the migration)
-- ============================================================

-- Verify indexes were created successfully
-- SELECT
--     schemaname,
--     tablename,
--     indexname,
--     indexdef
-- FROM pg_indexes
-- WHERE tablename IN ('stores', 'users', 'business_registrations')
--     AND indexname LIKE 'idx_%'
-- ORDER BY tablename, indexname;

-- Check for any duplicate active records that might prevent index creation
-- SELECT business_number, COUNT(*)
-- FROM stores
-- WHERE deleted_at IS NULL
-- GROUP BY business_number
-- HAVING COUNT(*) > 1;

-- SELECT email, COUNT(*)
-- FROM users
-- WHERE deleted_at IS NULL
-- GROUP BY email
-- HAVING COUNT(*) > 1;

-- SELECT store_id, COUNT(*)
-- FROM business_registrations
-- WHERE deleted_at IS NULL
-- GROUP BY store_id
-- HAVING COUNT(*) > 1;
