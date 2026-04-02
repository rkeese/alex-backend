-- Add initial balance fields to club_bank_accounts
ALTER TABLE club_bank_accounts ADD COLUMN initial_balance NUMERIC(12, 2) NOT NULL DEFAULT 0;
ALTER TABLE club_bank_accounts ADD COLUMN initial_balance_date DATE;

-- Add client IBAN/BIC to bookings to store counterparty details
ALTER TABLE bookings ADD COLUMN client_iban TEXT;
ALTER TABLE bookings ADD COLUMN client_bic TEXT;
