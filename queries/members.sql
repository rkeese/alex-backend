-- name: CreateMember :one
INSERT INTO members (
  club_id, member_number, first_name, last_name, birth_date, gender,
  street_house_number, postal_code, city, honorary, status,
  salutation, letter_salutation, phone1, phone1_note, phone2, phone2_note,
  mobile, mobile_note, email, email_note, nation, joined_at, member_until,
  note, marital_status, title, assigned_club_bank_id
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17,
  $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28
)
RETURNING *;

-- name: GetMemberByID :one
SELECT * FROM members
WHERE id = $1 AND club_id = $2 LIMIT 1;

-- name: UpdateMemberUser :exec
UPDATE members
SET user_id = $2
WHERE id = $1;

-- name: GetMemberByUserID :one
SELECT * FROM members
WHERE user_id = $1 LIMIT 1;

-- name: GetMemberDetails :one
SELECT 
    m.id, m.club_id, m.member_number, m.first_name, m.last_name, m.birth_date, m.gender, m.street_house_number, m.postal_code, m.city, m.honorary, m.status, m.salutation, m.letter_salutation, m.phone1, m.phone1_note, m.phone2, m.phone2_note, m.mobile, m.mobile_note, m.email, m.email_note, m.nation, m.joined_at, m.member_until, m.note, m.marital_status, m.title, m.created_at, m.updated_at, m.archived_at, m.user_id, m.assigned_club_bank_id,
    COALESCE(b.iban, '') AS iban,
    COALESCE(b.account_holder, '') AS account_holder,
    COALESCE(b.sepa_mandate_available, false) AS sepa_mandate_granted,
    COALESCE(b.mandate_reference, '') AS mandate_reference,
    b.mandate_issued_at AS mandate_granted_at,
    COALESCE(f.payment_method, '') AS payment_method,
    COALESCE(f.fee_label, '') AS fee_label,
    COALESCE(f.amount, 0)::float8 AS fee_amount,
    COALESCE(f.period, '') AS fee_period,
    f.starts_at AS fee_starts_at,
    f.maturity_date AS fee_maturity,
    b.id AS bank_account_id,
    f.creditor_account_id
FROM members m
LEFT JOIN (
    SELECT DISTINCT ON (member_id) member_id, id, iban, account_holder, sepa_mandate_available, mandate_reference, mandate_issued_at
    FROM member_bank_accounts
    ORDER BY member_id, created_at DESC
) b ON m.id = b.member_id
LEFT JOIN (
    SELECT DISTINCT ON (member_id) member_id, payment_method, fee_label, amount, period, starts_at, maturity_date, creditor_account_id
    FROM membership_fees
    ORDER BY member_id, starts_at DESC
) f ON m.id = f.member_id
WHERE m.id = $1 AND m.club_id = $2;

-- name: ListMemberDetails :many
SELECT 
    m.id, m.club_id, m.member_number, m.first_name, m.last_name, m.birth_date, m.gender, m.street_house_number, m.postal_code, m.city, m.honorary, m.status, m.salutation, m.letter_salutation, m.phone1, m.phone1_note, m.phone2, m.phone2_note, m.mobile, m.mobile_note, m.email, m.email_note, m.nation, m.joined_at, m.member_until, m.note, m.marital_status, m.title, m.created_at, m.updated_at, m.archived_at, m.user_id, m.assigned_club_bank_id,
    COALESCE(b.iban, '') AS iban,
    COALESCE(b.account_holder, '') AS account_holder,
    COALESCE(b.sepa_mandate_available, false) AS sepa_mandate_granted,
    COALESCE(b.mandate_reference, '') AS mandate_reference,
    b.mandate_issued_at AS mandate_granted_at,
    COALESCE(f.payment_method, '') AS payment_method,
    COALESCE(f.fee_label, '') AS fee_label,
    COALESCE(f.amount, 0)::float8 AS fee_amount,
    COALESCE(f.period, '') AS fee_period,
    f.starts_at AS fee_starts_at,
    f.maturity_date AS fee_maturity,
    b.id AS bank_account_id,
    f.creditor_account_id
FROM members m
LEFT JOIN (
    SELECT DISTINCT ON (member_id) member_id, id, iban, account_holder, sepa_mandate_available, mandate_reference, mandate_issued_at
    FROM member_bank_accounts
    ORDER BY member_id, created_at DESC
) b ON m.id = b.member_id
LEFT JOIN (
    SELECT DISTINCT ON (member_id) member_id, payment_method, fee_label, amount, period, starts_at, maturity_date, creditor_account_id
    FROM membership_fees
    ORDER BY member_id, starts_at DESC
) f ON m.id = f.member_id
WHERE m.club_id = $1
ORDER BY m.last_name, m.first_name;

-- name: ListMembers :many
SELECT * FROM members
WHERE club_id = $1
ORDER BY last_name, first_name;

-- name: UpdateMember :one
UPDATE members
SET
  member_number = $3,
  first_name = $4,
  last_name = $5,
  birth_date = $6,
  gender = $7,
  street_house_number = $8,
  postal_code = $9,
  city = $10,
  honorary = $11,
  status = $12,
  salutation = $13,
  letter_salutation = $14,
  phone1 = $15,
  phone1_note = $16,
  phone2 = $17,
  phone2_note = $18,
  mobile = $19,
  mobile_note = $20,
  email = $21,
  email_note = $22,
  nation = $23,
  joined_at = $24,
  member_until = $25,
  note = $26,
  marital_status = $27,
  title = $28,
  assigned_club_bank_id = $29,
  updated_at = NOW()
WHERE id = $1 AND club_id = $2
RETURNING *;

-- name: DeleteMember :exec
DELETE FROM members
WHERE id = $1 AND club_id = $2;

-- name: CreateMemberBankAccount :one
INSERT INTO member_bank_accounts (
  member_id, account_holder, iban, bic, sepa_mandate_available,
  mandate_reference, mandate_type, mandate_issued_at,
  mandate_kind, next_direct_debit_type, last_used_at, mandate_valid_until
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: GetMemberStatistics :many
SELECT
    CAST(EXTRACT(YEAR FROM birth_date) AS INTEGER) AS birth_year,
    COUNT(*) FILTER (WHERE gender = 'm') AS count_m,
    COUNT(*) FILTER (WHERE gender = 'f') AS count_f,
    COUNT(*) FILTER (WHERE gender = 'd') AS count_d,
    COUNT(*) AS count_total
FROM members
WHERE
    club_id = $1
    AND joined_at <= make_date($2::int, 12, 31)
    AND (member_until IS NULL OR member_until >= make_date($2::int, 1, 1))
GROUP BY birth_year
ORDER BY birth_year DESC;

-- name: UpdateMemberBankAccount :one
UPDATE member_bank_accounts
SET
  account_holder = $3,
  iban = $4,
  bic = $5,
  sepa_mandate_available = $6,
  mandate_reference = $7,
  mandate_type = $8,
  mandate_issued_at = $9,
  mandate_kind = $10,
  next_direct_debit_type = $11,
  last_used_at = $12,
  mandate_valid_until = $13,
  updated_at = NOW()
WHERE id = $1 AND member_id = $2
RETURNING *;

-- name: GetMemberBankAccountByIBAN :one
SELECT * FROM member_bank_accounts
WHERE member_id = $1 AND iban = $2
LIMIT 1;

-- name: GetLatestMembershipFee :one
SELECT * FROM membership_fees
WHERE member_id = $1
ORDER BY starts_at DESC
LIMIT 1;

