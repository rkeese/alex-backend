-- name: CreateUser :one
INSERT INTO users (email, password_hash, is_sys_admin, must_change_password, is_blocked)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: HasPermission :one
SELECT EXISTS (
  SELECT 1
  FROM user_roles ur
  JOIN roles r ON ur.role_id = r.id
  JOIN role_permissions rp ON r.id = rp.role_id
  JOIN permissions p ON rp.permission_id = p.id
  WHERE ur.user_id = $1
    AND ur.club_id = $2
    AND p.name = $3
);

-- name: GetRoleByName :one
SELECT * FROM roles
WHERE name = $1 LIMIT 1;

-- name: AddUserRole :exec
INSERT INTO user_roles (user_id, role_id, club_id)
VALUES ($1, $2, $3);

-- name: ListRoles :many
SELECT * FROM roles
ORDER BY name;

-- name: RemoveUserRole :exec
DELETE FROM user_roles
WHERE user_id = $1 AND role_id = $2 AND club_id = $3;

-- name: ListUsers :many
SELECT id, email, is_blocked, failed_login_attempts, locked_until, created_at, updated_at FROM users
ORDER BY email;

-- name: IncrementFailedLoginAttempts :exec
UPDATE users
SET failed_login_attempts = failed_login_attempts + 1,
    locked_until = CASE
        WHEN failed_login_attempts + 1 >= 5 THEN NOW() + INTERVAL '15 minutes'
        ELSE locked_until
    END,
    updated_at = NOW()
WHERE id = $1;

-- name: ResetFailedLoginAttempts :exec
UPDATE users
SET failed_login_attempts = 0,
    locked_until = NULL,
    updated_at = NOW()
WHERE id = $1;

-- name: GetUserRoles :many
SELECT r.name as role_name, ur.club_id
FROM user_roles ur
JOIN roles r ON ur.role_id = r.id
WHERE ur.user_id = $1;

-- name: UpdateUser :exec
UPDATE users
SET password_hash = $2,
    must_change_password = $3,
    is_blocked = $4,
    updated_at = NOW()
WHERE id = $1;
