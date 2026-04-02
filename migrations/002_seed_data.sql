-- Insert Roles
INSERT INTO roles (name) VALUES ('admin') ON CONFLICT DO NOTHING;
INSERT INTO roles (name) VALUES ('member') ON CONFLICT DO NOTHING;

-- Insert Permissions
INSERT INTO permissions (name) VALUES ('Mitglieder') ON CONFLICT DO NOTHING;
INSERT INTO permissions (name) VALUES ('Abteilungen') ON CONFLICT DO NOTHING;
INSERT INTO permissions (name) VALUES ('Finanzen') ON CONFLICT DO NOTHING;
INSERT INTO permissions (name) VALUES ('Kalender') ON CONFLICT DO NOTHING;
INSERT INTO permissions (name) VALUES ('Dokumente') ON CONFLICT DO NOTHING;

-- Assign Permissions to Admin Role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'admin' AND p.name IN ('Mitglieder', 'Abteilungen', 'Finanzen', 'Kalender', 'Dokumente')
ON CONFLICT DO NOTHING;

-- Insert Default Club
INSERT INTO clubs (name, registered_association, type, number, street_house_number, postal_code, city, tax_office_name, tax_office_tax_number, tax_office_assessment_period, tax_office_purpose, tax_office_decision_date, tax_office_decision_type)
VALUES ('Default Club', true, 'sport_club', '12345', 'Musterstr. 1', '12345', 'Musterstadt', 'Finanzamt', '123/456/7890', '2023', 'Gemeinnützigkeit', '2023-01-01', 'Freistellungsbescheid')
ON CONFLICT DO NOTHING;

-- Insert Default Admin User
INSERT INTO users (email, password_hash)
VALUES ('raimund.keese@web.de', '$2a$14$E1oBNCbMt1X6ea0v3a4.n.GDSDdMnIuSg3l0.EloigjRot83ybrWi') -- Hash for "admin"
ON CONFLICT DO NOTHING;

-- Assign Admin Role to Default User for Default Club (optional if Sys-Admin implies access, but good for consistency)
INSERT INTO user_roles (user_id, role_id, club_id)
SELECT u.id, r.id, c.id
FROM users u, roles r, clubs c
WHERE u.email = 'raimund.keese@web.de' AND r.name = 'admin' AND c.name = 'Default Club'
ON CONFLICT DO NOTHING;

