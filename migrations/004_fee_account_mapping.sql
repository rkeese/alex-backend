-- Fee Account Mappings Table
CREATE TABLE IF NOT EXISTS fee_account_mappings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  club_id UUID NOT NULL REFERENCES clubs(id) ON DELETE CASCADE,
  fee_type TEXT NOT NULL,
  club_bank_account_id UUID NOT NULL REFERENCES club_bank_accounts(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (club_id, fee_type)
);

CREATE INDEX IF NOT EXISTS idx_fee_account_mappings_lookup ON fee_account_mappings (club_id, fee_type);
