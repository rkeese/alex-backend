-- Allow the position_assignment to be empty/null by dropping the check constraint
ALTER TABLE receipts DROP CONSTRAINT IF EXISTS receipts_position_assignment_check;

-- Ensure the column allows NULL values
ALTER TABLE receipts ALTER COLUMN position_assignment DROP NOT NULL;
