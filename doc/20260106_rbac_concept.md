# RBAC Concept for Access Levels

To meet the requirement of defining specific access levels (Locked, Read Only, Write, Write & Delete) for different areas (Members, Finance, etc.), we will use a **Granular Permission** and **Composite Role** approach within the existing database structure.

## 1. Permissions (Concept)

Permissions will be renamed to follow specific `area:action` format. This allows the code to check specifically for `read`, `write`, or `delete` capability.

**Format:** `<area>:<action>`

**Actions:**
- `read`: Allows viewing data.
- `write`: Allows creating and editing data.
- `delete`: Allows deleting data.

**Areas:**
- `members` (Mitglieder)
- `departments` (Abteilungen)
- `finance` (Finanzen)
- `calendar` (Kalender)
- `documents` (Dokumente)

**Generated Permissions:**
- `members:read`, `members:write`, `members:delete`
- `departments:read`, `departments:write`, `departments:delete`
- ...and so on.

## 2. Roles (Representation)

We will define **Access Roles** that correspond to the "Allowed" states mentioned in the requirements.

| Level | Description | Permissions Included | Role Naming Convention |
| :--- | :--- | :--- | :--- |
| **Locked** | No access | None | (No role assigned for this area) |
| **Read Only** | Can only view | `area:read` | `<area>_viewer` |
| **Write** | Can view and edit | `area:read`, `area:write` | `<area>_writer` |
| **Write & Delete** | Full control | `area:read`, `area:write`, `area:delete` | `<area>_manager` |

## 3. Minimal Rights (New User)

We will define a **Base Role** (e.g., `standard_user` or `guest`) that is assigned to all new users upon creation.

- **Role Name:** `new_user` (or `guest`)
- **Permissions:** (Example) Only `calendar:read` or none, depending on "minimal".

## 4. Admin Privileges

A `superuser` or `admin` role will exist that contains **all permissions** across all areas.

## 5. Implementation Strategy

### Database Changes (Migration)
We will populate the `permissions` and `roles` tables with these new definitions.

### User Assignment
To configure a user who has "Read Only" on Members and "Write" on Finance:
1. Assign role `members_viewer`
2. Assign role `finance_writer`

This allows full flexibility to mix and match levels across different areas.

### SQL Schema Usage
No schema structural changes are strictly necessary (the existing `users`, `roles`, `permissions`, `user_roles`, `role_permissions` tables are sufficient). We only need to update the **data** within them.

## 6. Example SQL for Permissions

```sql
INSERT INTO permissions (name) VALUES 
('members:read'), ('members:write'), ('members:delete'),
('finance:read'), ('finance:write'), ('finance:delete');
-- etc...
```

## 7. Example SQL for Roles

```sql
-- Create "Members Viewer" Role
INSERT INTO roles (name) VALUES ('members_viewer');
-- Assign 'members:read' to 'members_viewer'

-- Create "Members Writer" Role
INSERT INTO roles (name) VALUES ('members_writer');
-- Assign 'members:read' and 'members:write' to 'members_writer'
```
