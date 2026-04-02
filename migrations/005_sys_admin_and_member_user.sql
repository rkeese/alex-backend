-- Migration 005: Add Sys-Admin and link Members to Users
ALTER TABLE users ADD COLUMN is_sys_admin BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN must_change_password BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN is_blocked BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE members ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE SET NULL;
CREATE UNIQUE INDEX idx_members_user_id ON members(user_id) WHERE user_id IS NOT NULL;

-- Set Sys-Admin
UPDATE users SET is_sys_admin = TRUE WHERE email = 'raimund.keese@web.de';

-- Cleanup old admin if exists
DELETE FROM users WHERE email = 'admin@example.com';
