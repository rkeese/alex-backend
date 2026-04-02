-- Relax check constraint on membership_fees.amount to allow 0.00
ALTER TABLE membership_fees DROP CONSTRAINT membership_fees_amount_check;
ALTER TABLE membership_fees ADD CONSTRAINT membership_fees_amount_check CHECK (amount >= 0);
