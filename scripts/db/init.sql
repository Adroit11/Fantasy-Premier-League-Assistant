-- ==============================================================================
-- FPL Assistant – MySQL Initialization Script
-- Runs automatically on first container start via docker-entrypoint-initdb.d
--
-- The database itself is already created by the MYSQL_DATABASE env var.
-- This script ensures the correct charset/collation and grants privileges.
-- Go migrations (internal/db/mysql.go) will create all tables on first boot.
-- ==============================================================================

-- Enforce utf8mb4 on the target database
ALTER DATABASE fpl_assistant
    CHARACTER SET = utf8mb4
    COLLATE = utf8mb4_unicode_ci;

-- Grant all privileges to the application user
GRANT ALL PRIVILEGES ON fpl_assistant.* TO 'fpl_user'@'%';
FLUSH PRIVILEGES;
