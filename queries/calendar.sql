-- name: CreateEvent :one
INSERT INTO events (
  club_id, date, time, description
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: ListEvents :many
SELECT * FROM events
WHERE club_id = $1
ORDER BY date, time;

-- name: ListEventsByDateRange :many
SELECT * FROM events
WHERE club_id = $1
  AND date >= sqlc.arg(from_date)
  AND date <= sqlc.arg(to_date)
ORDER BY date, time;

-- name: UpdateEvent :one
UPDATE events
SET
  date = $3,
  time = $4,
  description = $5,
  updated_at = NOW()
WHERE id = $1 AND club_id = $2
RETURNING *;

-- name: DeleteEvent :exec
DELETE FROM events
WHERE id = $1 AND club_id = $2;
