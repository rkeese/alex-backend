-- Migration: 017_expand_receipts_for_invoices.sql
-- Description: Expands receipts table to store invoice details including seller/buyer info and items.

ALTER TABLE receipts
ADD COLUMN IF NOT EXISTS seller_name TEXT,
ADD COLUMN IF NOT EXISTS seller_address TEXT,
ADD COLUMN IF NOT EXISTS buyer_name TEXT,
ADD COLUMN IF NOT EXISTS buyer_address TEXT,
ADD COLUMN IF NOT EXISTS seller_tax_id TEXT,
ADD COLUMN IF NOT EXISTS seller_vat_id TEXT,
ADD COLUMN IF NOT EXISTS delivery_date DATE,
ADD COLUMN IF NOT EXISTS total_vat_amount NUMERIC(12, 2) DEFAULT 0,
ADD COLUMN IF NOT EXISTS invoice_items JSONB NOT NULL DEFAULT '[]'::jsonb;
