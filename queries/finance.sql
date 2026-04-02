-- name: CreateBookingAccount :one
INSERT INTO booking_accounts (
  club_id, majority_list, minority_list
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: ListBookingAccounts :many
SELECT * FROM booking_accounts
WHERE club_id = $1
ORDER BY majority_list, minority_list;

-- name: CreateReceipt :one
INSERT INTO receipts (
  club_id, type, recipient, number, date, position_assignment,
  amount, is_booked, note, position_tax_account, position_percentage, donor_id,
  seller_name, seller_address, buyer_name, buyer_address, seller_tax_id, seller_vat_id, delivery_date, total_vat_amount, invoice_items
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
  $13, $14, $15, $16, $17, $18, $19, $20, $21
)
RETURNING *;

-- name: UpdateReceipt :one
UPDATE receipts
SET
  type = $3,
  recipient = $4,
  number = $5,
  date = $6,
  position_assignment = $7,
  amount = $8,
  is_booked = $9,
  note = $10,
  position_tax_account = $11,
  position_percentage = $12,
  donor_id = $13,
  seller_name = $14,
  seller_address = $15,
  buyer_name = $16,
  buyer_address = $17,
  seller_tax_id = $18,
  seller_vat_id = $19,
  delivery_date = $20,
  total_vat_amount = $21,
  invoice_items = $22,
  updated_at = NOW()
WHERE id = $1 AND club_id = $2
RETURNING *;

-- name: DeleteReceipt :exec
DELETE FROM receipts
WHERE id = $1 AND club_id = $2;

-- name: ListReceipts :many
SELECT * FROM receipts
WHERE club_id = $1
ORDER BY date DESC;

-- name: GetReceipt :one
SELECT * FROM receipts
WHERE id = $1 AND club_id = $2
LIMIT 1;

-- name: SetReceiptBooked :exec
UPDATE receipts
SET
  is_booked = $3,
  updated_at = NOW()
WHERE id = $1 AND club_id = $2;

-- name: CreateBooking :one
INSERT INTO bookings (
  club_id, booking_date, valuta_date, client_recipient, booking_text,
  purpose, amount, currency, receipt_id, assigned_booking_account_id, club_bank_account_id,
  payment_participant_iban, payment_participant_bic
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
RETURNING *;

-- name: ListBookings :many
SELECT * FROM bookings
WHERE club_id = $1
  AND (sqlc.narg('club_bank_account_id')::UUID IS NULL OR club_bank_account_id = sqlc.narg('club_bank_account_id'))
  AND (sqlc.narg('start_date')::DATE IS NULL OR booking_date >= sqlc.narg('start_date'))
  AND (sqlc.narg('end_date')::DATE IS NULL OR booking_date <= sqlc.narg('end_date'))
ORDER BY booking_date DESC;

-- name: GetBookingStartBalance :one
-- Calculates balance: sum(bookings before_date)
SELECT 
  COALESCE(SUM(b.amount), 0)::NUMERIC
FROM bookings b
WHERE b.club_bank_account_id = sqlc.arg('club_bank_account_id') 
  AND b.club_id = sqlc.arg('club_id')
  AND b.valuta_date < sqlc.arg('before_date')::DATE;

-- name: UpdateBooking :one
UPDATE bookings
SET
  assigned_booking_account_id = $3,
  updated_at = NOW()
WHERE id = $1 AND club_id = $2
RETURNING *;

-- name: CreateClubBankAccount :one
INSERT INTO club_bank_accounts (
  club_id, name, account_holder, creditor_id, iban, bic, is_default
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;
-- name: ListClubBankAccounts :many
SELECT * FROM club_bank_accounts
WHERE club_id = $1
ORDER BY name;

-- name: GetDefaultClubBankAccount :one
SELECT * FROM club_bank_accounts
WHERE club_id = $1 AND is_default = TRUE
LIMIT 1;

-- name: GetReceiptWithDonor :one
SELECT r.*, d.first_name, d.last_name, d.street_house_number AS donor_street, d.postal_code AS donor_zip, d.city AS donor_city
FROM receipts r
LEFT JOIN donors d ON r.donor_id = d.id
WHERE r.id = $1 AND r.club_id = $2;



-- name: GetClubBankAccountByID :one
SELECT * FROM club_bank_accounts
WHERE id = $1 AND club_id = $2
LIMIT 1;

-- name: UpdateClubBankAccount :one
UPDATE club_bank_accounts
SET
  name = $3,
  account_holder = $4,
  creditor_id = $5,
  iban = $6,
  bic = $7,
  is_default = $8,
  updated_at = NOW()
WHERE id = $1 AND club_id = $2
RETURNING *;

-- name: DeleteClubBankAccount :exec
DELETE FROM club_bank_accounts
WHERE id = $1 AND club_id = $2;


-- name: CreateBankBookingImport :one
INSERT INTO bank_bookings_import (
  club_id, club_bank_account_id, booking_date, valuta_date, amount, currency,
  purpose, payment_participant_name, payment_participant_iban, payment_participant_bic,
  status
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING *;

-- name: ListBankBookingImports :many
SELECT * FROM bank_bookings_import
WHERE club_id = $1 AND status = 'pending'
ORDER BY booking_date DESC;

-- name: UpdateBankBookingImport :one
UPDATE bank_bookings_import
SET
  club_bank_account_id = $3,
  booking_date = $4,
  valuta_date = $5,
  amount = $6,
  purpose = $7,
  payment_participant_name = $8,
  payment_participant_iban = $9,
  payment_participant_bic = $10,
  status = $11
WHERE id = $1 AND club_id = $2
RETURNING *;

-- name: DeleteBankBookingImport :exec
DELETE FROM bank_bookings_import
WHERE id = $1 AND club_id = $2;
