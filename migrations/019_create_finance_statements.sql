-- Migration: Create Finance Statements Table
CREATE TABLE IF NOT EXISTS finance_statements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    club_id UUID NOT NULL REFERENCES clubs(id) ON DELETE CASCADE,
    year INT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    data JSONB NOT NULL,
    UNIQUE(club_id, year)
);

COMMENT ON COLUMN finance_statements.data IS 'Stores the generated JSON report structure (balances, overview, details)';
