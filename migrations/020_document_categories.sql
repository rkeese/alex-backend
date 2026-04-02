-- Migration 020: Document Categories
-- Description: Adds a document_categories table and links documents to categories.
-- Categories are per-club and user-manageable for future flexibility.
-- Default categories are seeded: protocols, contracts, invoices, correspondence, miscellaneous.

-- 1. Create document_categories table
CREATE TABLE IF NOT EXISTS document_categories (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  club_id UUID NOT NULL REFERENCES clubs(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT,
  sort_order INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(club_id, name)
);

-- 2. Add category_id to documents (nullable for backward compatibility)
ALTER TABLE documents ADD COLUMN IF NOT EXISTS category_id UUID REFERENCES document_categories(id) ON DELETE SET NULL;

-- 3. Add a description column to documents for optional metadata
ALTER TABLE documents ADD COLUMN IF NOT EXISTS description TEXT;

-- 4. Seed default categories for every existing club
INSERT INTO document_categories (club_id, name, description, sort_order)
SELECT c.id, cat.name, cat.description, cat.sort_order
FROM clubs c
CROSS JOIN (VALUES
  ('protocols',      'Meeting protocols and minutes',    1),
  ('contracts',      'Contracts and agreements',         2),
  ('invoices',       'Invoices and billing documents',   3),
  ('correspondence', 'Letters and correspondence',       4),
  ('miscellaneous',  'Other documents',                  5)
) AS cat(name, description, sort_order)
ON CONFLICT (club_id, name) DO NOTHING;
