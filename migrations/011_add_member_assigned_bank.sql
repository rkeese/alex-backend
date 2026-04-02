-- Add assigned_club_bank_id to members
ALTER TABLE members ADD COLUMN assigned_club_bank_id UUID REFERENCES club_bank_accounts(id) ON DELETE SET NULL;
