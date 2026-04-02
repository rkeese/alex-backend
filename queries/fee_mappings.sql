-- name: CreateFeeAccountMapping :one
INSERT INTO fee_account_mappings (
  club_id, fee_type, club_bank_account_id
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: GetFeeAccountMapping :one
SELECT * FROM fee_account_mappings
WHERE club_id = $1 AND fee_type = $2;

-- name: ListFeeAccountMappings :many
SELECT fam.*, cba.name as bank_account_name, cba.iban
FROM fee_account_mappings fam
JOIN club_bank_accounts cba ON fam.club_bank_account_id = cba.id
WHERE fam.club_id = $1;

-- name: UpdateFeeAccountMapping :one
UPDATE fee_account_mappings
SET club_bank_account_id = $3, updated_at = NOW()
WHERE club_id = $1 AND fee_type = $2
RETURNING *;

-- name: DeleteFeeAccountMapping :exec
DELETE FROM fee_account_mappings
WHERE club_id = $1 AND fee_type = $2;
