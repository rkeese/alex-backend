-- name: CreateBoardMember :one
INSERT INTO board_members (club_id, member_id, user_id, position)
VALUES ($1, $2, $3, $4)
RETURNING id, club_id, member_id, user_id, position, created_at, updated_at;

-- name: GetBoardMembers :many
SELECT bm.id, bm.club_id, bm.member_id, bm.user_id, bm.position, bm.created_at, bm.updated_at,
       m.first_name, m.last_name, m.member_number,
       u.email
FROM board_members bm
JOIN members m ON bm.member_id = m.id
JOIN users u ON bm.user_id = u.id
WHERE bm.club_id = $1
ORDER BY m.last_name, m.first_name;

-- name: GetBoardMember :one
SELECT id, club_id, member_id, user_id, position, created_at, updated_at FROM board_members WHERE id = $1;

-- name: UpdateBoardMember :one
UPDATE board_members
SET position = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, club_id, member_id, user_id, position, created_at, updated_at;

-- name: DeleteBoardMember :exec
DELETE FROM board_members WHERE id = $1;
