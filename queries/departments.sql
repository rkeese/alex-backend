-- name: CreateDepartment :one
INSERT INTO departments (
  club_id, name, subdivision, parent_id
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: GetDepartmentByID :one
SELECT * FROM departments
WHERE id = $1 AND club_id = $2 LIMIT 1;

-- name: ListDepartments :many
SELECT * FROM departments
WHERE club_id = $1
ORDER BY name;

-- name: UpdateDepartment :one
UPDATE departments
SET
  name = $3,
  subdivision = $4,
  parent_id = $5,
  updated_at = NOW()
WHERE id = $1 AND club_id = $2
RETURNING *;

-- name: DeleteDepartment :exec
DELETE FROM departments
WHERE id = $1 AND club_id = $2;
