-- Create bookings import table
CREATE TABLE IF NOT EXISTS bank_bookings_import (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  club_id UUID NOT NULL REFERENCES clubs(id) ON DELETE CASCADE,
  club_bank_account_id UUID REFERENCES club_bank_accounts(id) ON DELETE SET NULL,
  booking_date DATE NOT NULL,
  valuta_date DATE NOT NULL,
  amount NUMERIC(12, 2) NOT NULL,
  currency VARCHAR(3) NOT NULL DEFAULT 'EUR',
  purpose TEXT,
  payment_participant_name TEXT,
  payment_participant_iban TEXT,
  payment_participant_bic TEXT,
  status TEXT NOT NULL DEFAULT 'pending',
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Rename columns in bookings (idempotent)
DO $$ BEGIN
  ALTER TABLE bookings RENAME COLUMN bank_account_id TO club_bank_account_id;
EXCEPTION WHEN undefined_column THEN NULL;
END $$;
DO $$ BEGIN
  ALTER TABLE bookings RENAME COLUMN client_iban TO payment_participant_iban;
EXCEPTION WHEN undefined_column THEN NULL;
END $$;
DO $$ BEGIN
  ALTER TABLE bookings RENAME COLUMN client_bic TO payment_participant_bic;
EXCEPTION WHEN undefined_column THEN NULL;
END $$;

-- Drop columns from club_bank_accounts
ALTER TABLE club_bank_accounts DROP COLUMN IF EXISTS initial_balance;
ALTER TABLE club_bank_accounts DROP COLUMN IF EXISTS initial_balance_date;
