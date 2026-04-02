-- Add bank_account_id to bookings
ALTER TABLE bookings ADD COLUMN bank_account_id UUID REFERENCES club_bank_accounts(id) ON DELETE SET NULL;

-- Relax constraint on booking_account_id and rename it
ALTER TABLE bookings ALTER COLUMN booking_account_id DROP NOT NULL;
ALTER TABLE bookings RENAME COLUMN booking_account_id TO assigned_booking_account_id;
