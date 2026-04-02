-- Migration 003: Setup Granular RBAC
-- Description: Sets up permissions and roles for area-based access control (Locked, Read, Write, Delete)

-- 1. Clean up old permissions/roles (optional, varies by deployment strategy)
-- For this migration, we will clear existing mappings to ensure a clean state
TRUNCATE TABLE role_permissions CASCADE;
TRUNCATE TABLE permissions CASCADE;
TRUNCATE TABLE roles CASCADE;

-- 2. Insert Granular Permissions
-- Format: area:action
INSERT INTO permissions (name) VALUES
-- Members
('members:read'), ('members:write'), ('members:delete'),
-- Departments
('departments:read'), ('departments:write'), ('departments:delete'),
-- Finance
('finance:read'), ('finance:write'), ('finance:delete'),
-- Calendar
('calendar:read'), ('calendar:write'), ('calendar:delete'),
-- Documents
('documents:read'), ('documents:write'), ('documents:delete');

-- 3. Insert Functional Roles (Level per Area)

-- Members
INSERT INTO roles (name) VALUES ('members_viewer'), ('members_writer'), ('members_manager');

-- Departments
INSERT INTO roles (name) VALUES ('departments_viewer'), ('departments_writer'), ('departments_manager');

-- Finance
INSERT INTO roles (name) VALUES ('finance_viewer'), ('finance_writer'), ('finance_manager');

-- Calendar
INSERT INTO roles (name) VALUES ('calendar_viewer'), ('calendar_writer'), ('calendar_manager');

-- Documents
INSERT INTO roles (name) VALUES ('documents_viewer'), ('documents_writer'), ('documents_manager');

-- Super Admin & Minimal User
INSERT INTO roles (name) VALUES ('admin'), ('new_user');

-- 4. Map Permissions to Roles

-- Helper function or direct inserts. Using Direct Inserts with subqueries for portability.

-- Members Roles
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'members_viewer' AND p.name IN ('members:read');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'members_writer' AND p.name IN ('members:read', 'members:write');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'members_manager' AND p.name IN ('members:read', 'members:write', 'members:delete');

-- Departments Roles
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'departments_viewer' AND p.name IN ('departments:read');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'departments_writer' AND p.name IN ('departments:read', 'departments:write');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'departments_manager' AND p.name IN ('departments:read', 'departments:write', 'departments:delete');

-- Finance Roles
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'finance_viewer' AND p.name IN ('finance:read');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'finance_writer' AND p.name IN ('finance:read', 'finance:write');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'finance_manager' AND p.name IN ('finance:read', 'finance:write', 'finance:delete');

-- Calendar Roles
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'calendar_viewer' AND p.name IN ('calendar:read');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'calendar_writer' AND p.name IN ('calendar:read', 'calendar:write');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'calendar_manager' AND p.name IN ('calendar:read', 'calendar:write', 'calendar:delete');

-- Documents Roles
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'documents_viewer' AND p.name IN ('documents:read');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'documents_writer' AND p.name IN ('documents:read', 'documents:write');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'documents_manager' AND p.name IN ('documents:read', 'documents:write', 'documents:delete');

-- Admin (All Permissions)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'admin';

-- New User (Minimal Rights - e.g., Read Calendar)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'new_user' AND p.name IN ('calendar:read');

-- 5. Restore Admin User Access
-- Ensure Admin User exists (re-inserted if missing, or ignored if present)
INSERT INTO users (email, password_hash)
VALUES ('admin@example.com', '$2a$14$E1oBNCbMt1X6ea0v3a4.n.GDSDdMnIuSg3l0.EloigjRot83ybrWi')
ON CONFLICT DO NOTHING;

-- Ensure Default Club exists (if missing)
-- Using ON CONFLICT DO NOTHING without target to handle ANY unique constraint violation (name OR number)
INSERT INTO clubs (name, registered_association, type, number, street_house_number, postal_code, city, tax_office_name, tax_office_tax_number, tax_office_assessment_period, tax_office_purpose, tax_office_decision_date, tax_office_decision_type)
VALUES ('Default Club', true, 'sport_club', '12345', 'Musterstr. 1', '12345', 'Musterstadt', 'Finanzamt', '123/456/7890', '2023', 'Gemeinnützigkeit', '2023-01-01', 'Freistellungsbescheid')
ON CONFLICT DO NOTHING;

-- Assign 'admin' role to 'admin@example.com' for the club (found by name OR number)
INSERT INTO user_roles (user_id, role_id, club_id)
SELECT u.id, r.id, c.id
FROM users u, roles r, clubs c
WHERE u.email = 'admin@example.com' 
  AND r.name = 'admin' 
  AND (c.name = 'Default Club' OR c.number = '12345')
ON CONFLICT DO NOTHING;
