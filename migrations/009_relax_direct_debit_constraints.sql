-- Migration 009: Relax direct debit and fee constraints for incomplete data
ALTER TABLE membership_fees ALTER COLUMN maturity_date DROP NOT NULL;
ALTER TABLE member_bank_accounts ALTER COLUMN mandate_issued_at DROP NOT NULL;
ALTER TABLE member_bank_accounts ALTER COLUMN mandate_valid_until DROP NOT NULL;
ALTER TABLE member_bank_accounts ALTER COLUMN mandate_reference DROP NOT NULL;
