-- name: CreateClub :one
INSERT INTO clubs (
  registered_association, name, type, category, number, 
  street_house_number, postal_code, city, name_extension, address_extension,
  tax_office_name, tax_office_tax_number, tax_office_assessment_period,
  tax_office_purpose, tax_office_decision_date, tax_office_decision_type
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
  $11, $12, $13, $14, $15, $16
)
RETURNING *;

-- name: GetClubByID :one
SELECT * FROM clubs
WHERE id = $1 LIMIT 1;

-- name: ListClubs :many
SELECT * FROM clubs
ORDER BY name;

-- name: UpdateClub :one
UPDATE clubs
SET
  registered_association = $2,
  name = $3,
  type = $4,
  category = $5,
  number = $6,
  street_house_number = $7,
  postal_code = $8,
  city = $9,
  name_extension = $10,
  address_extension = $11,
  tax_office_name = $12,
  tax_office_tax_number = $13,
  tax_office_assessment_period = $14,
  tax_office_purpose = $15,
  tax_office_decision_date = $16,
  tax_office_decision_type = $17,
  updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetFirstClub :one
SELECT * FROM clubs
ORDER BY created_at ASC
LIMIT 1;

-- name: DeleteClub :exec
DELETE FROM clubs
WHERE id = $1;

