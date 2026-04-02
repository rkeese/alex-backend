-- name: CreateFinanceStatement :one
INSERT INTO finance_statements (
  club_id, year, start_date, end_date, data
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetFinanceStatement :one
SELECT * FROM finance_statements
WHERE id = $1 AND club_id = $2
LIMIT 1;

-- name: ListFinanceStatements :many
SELECT id, club_id, year, start_date, end_date, created_at FROM finance_statements
WHERE club_id = $1
ORDER BY year DESC;

-- name: DeleteFinanceStatement :exec
DELETE FROM finance_statements
WHERE id = $1 AND club_id = $2;

-- name: GetFinanceStatementByYear :one
SELECT * FROM finance_statements
WHERE club_id = $1 AND year = $2
LIMIT 1;

-- name: GetBookingSumInRange :one
SELECT COALESCE(SUM(amount), 0)::NUMERIC
FROM bookings
WHERE club_id = $1
  AND club_bank_account_id = $2
  AND valuta_date >= $3::DATE
  AND valuta_date < $4::DATE;

-- name: ListBookingsInRange :many
SELECT b.*, ba.majority_list, ba.minority_list 
FROM bookings b
LEFT JOIN booking_accounts ba ON b.assigned_booking_account_id = ba.id
WHERE b.club_id = $1
  AND b.valuta_date >= $2::DATE
  AND b.valuta_date <= $3::DATE
ORDER BY b.valuta_date ASC;

-- name: ListUnbookedReceiptsInRange :many
SELECT * FROM receipts
WHERE club_id = $1
  AND is_booked = FALSE
  AND date >= $2::DATE
  AND date <= $3::DATE
ORDER BY date ASC;

-- name: DebugListAllBookings :many
SELECT b.*, ba.majority_list, ba.minority_list
FROM bookings b
LEFT JOIN booking_accounts ba ON b.assigned_booking_account_id = ba.id
WHERE b.club_id = $1
ORDER BY b.valuta_date ASC;
