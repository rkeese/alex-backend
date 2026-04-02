-- name: CreateMembershipFee :one
INSERT INTO membership_fees (
  member_id, fee_label, fee_type, assignment, amount, period,
  maturity_date, payment_method, starts_at, ends_at,
  creditor_account_id
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
ON CONFLICT (member_id, starts_at) DO UPDATE SET
  fee_label = EXCLUDED.fee_label,
  fee_type = EXCLUDED.fee_type,
  assignment = EXCLUDED.assignment,
  amount = EXCLUDED.amount,
  period = EXCLUDED.period,
  maturity_date = EXCLUDED.maturity_date,
  payment_method = EXCLUDED.payment_method,
  ends_at = EXCLUDED.ends_at,
  creditor_account_id = EXCLUDED.creditor_account_id,
  updated_at = NOW()
RETURNING *;

-- name: ListMembershipFees :many
SELECT * FROM membership_fees
WHERE member_id = $1
ORDER BY starts_at DESC;

-- name: GetDueMembershipFees :many
SELECT DISTINCT ON (mf.member_id)
  mf.*, 
  m.first_name, 
  m.last_name, 
  mba.iban AS member_iban, 
  mba.bic AS member_bic, 
  mba.mandate_reference, 
  mba.mandate_issued_at,
  mba.next_direct_debit_type,
  cba.name AS target_bank_name,
  cba.iban::text AS target_iban,
  cba.bic::text AS target_bic,
  cba.creditor_id::text AS target_creditor_id,
  cba.account_holder::text AS target_account_holder
FROM membership_fees mf
JOIN members m ON mf.member_id = m.id
JOIN member_bank_accounts mba ON m.id = mba.member_id
JOIN club_bank_accounts cba ON cba.id = COALESCE(mf.creditor_account_id, m.assigned_club_bank_id, (SELECT id FROM club_bank_accounts WHERE club_id = m.club_id AND is_default = TRUE LIMIT 1))
WHERE m.club_id = $1
  AND mf.payment_method = 'sepa'
  AND mf.maturity_date <= $2
  AND (mf.ends_at IS NULL OR mf.ends_at >= $2)
  AND mba.sepa_mandate_available = TRUE
ORDER BY mf.member_id, mf.maturity_date ASC, mba.created_at DESC;
