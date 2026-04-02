-- Add creditor_account_id to membership_fees
ALTER TABLE membership_fees ADD COLUMN IF NOT EXISTS creditor_account_id UUID REFERENCES club_bank_accounts(id) ON DELETE SET NULL;
