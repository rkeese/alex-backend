-- =============================================
-- VOLLSTÄNDIGES FEHLERFREIES DATENBANK-SCHEMA
-- =============================================
-- Letzte Überprüfung: 2023-10-15 | PostgreSQL 14+
-- =============================================

-- ---------------------------------------------
-- 1. VORBEREITUNG: ALTE STRUKTUR LÖSCHEN
-- ---------------------------------------------
-- Hinweis: Die "NOTICE"-Meldungen sind normal und kein Fehler!
-- ---------------------------------------------
DROP TABLE IF EXISTS member_departments CASCADE;
DROP TABLE IF EXISTS contact_persons CASCADE;
DROP TABLE IF EXISTS member_bank_accounts CASCADE;
DROP TABLE IF EXISTS membership_fees CASCADE;
DROP TABLE IF EXISTS bookings CASCADE;
DROP TABLE IF EXISTS receipts CASCADE;
DROP TABLE IF EXISTS documents CASCADE;
DROP TABLE IF EXISTS events CASCADE;
DROP TABLE IF EXISTS bank_connections CASCADE;
DROP TABLE IF EXISTS club_bank_accounts CASCADE;
DROP TABLE IF EXISTS booking_accounts CASCADE;
DROP TABLE IF EXISTS members CASCADE;
DROP TABLE IF EXISTS departments CASCADE;
DROP TABLE IF EXISTS donors CASCADE;
DROP TABLE IF EXISTS user_roles CASCADE;
DROP TABLE IF EXISTS role_permissions CASCADE;
DROP TABLE IF EXISTS permissions CASCADE;
DROP TABLE IF EXISTS roles CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS clubs CASCADE;

-- ---------------------------------------------
-- 2. BASIS-TABELLEN (MANDANTENSICHERHEIT)
-- ---------------------------------------------
-- Vereine (Mandanten)
CREATE TABLE clubs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  registered_association BOOLEAN NOT NULL,
  name TEXT NOT NULL UNIQUE,
  type TEXT NOT NULL CHECK (type IN (
    'sport_club', 'music_club', 'social_club',
    'environment_club', 'cultural_club', 'hobby_club', 'rescue_service'
  )),
  category TEXT,
  number TEXT NOT NULL UNIQUE,
  street_house_number TEXT NOT NULL,
  postal_code TEXT NOT NULL,
  city TEXT NOT NULL,
  name_extension TEXT,
  address_extension TEXT,
  phone TEXT,
  email TEXT,
  -- Steueramt
  tax_office_name TEXT NOT NULL,
  tax_office_tax_number TEXT NOT NULL,
  tax_office_vat_id TEXT,
  tax_office_assessment_period TEXT NOT NULL,
  tax_office_purpose TEXT NOT NULL,
  tax_office_decision_date DATE NOT NULL,
  tax_office_decision_type TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Abteilungen (mit Functional Index für Eindeutigkeit)
CREATE TABLE departments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  club_id UUID NOT NULL REFERENCES clubs(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  subdivision TEXT,
  parent_id UUID REFERENCES departments(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Functional Index (ersetzt UNIQUE-Constraint mit COALESCE)
CREATE UNIQUE INDEX idx_departments_unique ON departments (
  club_id,
  name,
  COALESCE(subdivision, '')
);

-- ---------------------------------------------
-- 3. MITGLIEDERVERWALTUNG
-- ---------------------------------------------
-- Mitglieder
CREATE TABLE members (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  club_id UUID NOT NULL REFERENCES clubs(id) ON DELETE CASCADE,
  member_number TEXT NOT NULL,
  first_name TEXT NOT NULL,
  last_name TEXT NOT NULL,
  birth_date DATE NOT NULL,
  gender TEXT CHECK (gender IN ('unknown', 'm', 'f', 'd')),
  street_house_number TEXT NOT NULL,
  postal_code TEXT NOT NULL,
  city TEXT NOT NULL,
  honorary BOOLEAN NOT NULL DEFAULT FALSE,
  status TEXT NOT NULL CHECK (status IN ('active', 'passive', 'honorary', 'inactive')),
  salutation TEXT CHECK (salutation IN ('mr', 'ms', 'div', 'company')),
  letter_salutation TEXT,
  phone1 TEXT,
  phone1_note TEXT,
  phone2 TEXT,
  phone2_note TEXT,
  mobile TEXT,
  mobile_note TEXT,
  email TEXT,
  email_note TEXT,
  nation TEXT,
  joined_at DATE NOT NULL,  -- EINTRITTS-DATUM (früher "member_since")
  member_until DATE,        -- AUSTRITTS-DATUM
  note TEXT,
  marital_status TEXT CHECK (marital_status IN ('single', 'married', 'divorced', 'widowed')),
  title TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  archived_at TIMESTAMPTZ,
  UNIQUE (club_id, member_number)
);

-- n:m-Beziehung Mitglieder ↔ Abteilungen
CREATE TABLE member_departments (
  member_id UUID NOT NULL REFERENCES members(id) ON DELETE CASCADE,
  department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
  PRIMARY KEY (member_id, department_id)
);

-- Bankverbindungen pro Mitglied (SEPA)
CREATE TABLE member_bank_accounts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  member_id UUID NOT NULL REFERENCES members(id) ON DELETE CASCADE,
  account_holder TEXT NOT NULL,
  iban TEXT NOT NULL CHECK (iban ~ '^[A-Z]{2}\d{2}[A-Z0-9]{4}\d{7}([A-Z0-9]?){0,16}$'),
  bic TEXT CHECK (bic ~ '^[A-Z]{6}[A-Z2-9][A-NP-Z0-9]([A-Z0-9]{3})?$'),
  sepa_mandate_available BOOLEAN NOT NULL DEFAULT FALSE,
  mandate_reference TEXT NOT NULL,
  mandate_type TEXT NOT NULL DEFAULT 'basic',
  mandate_issued_at DATE NOT NULL,
  mandate_kind TEXT NOT NULL CHECK (mandate_kind IN ('open_ended', 'one_time', 'last')),
  next_direct_debit_type TEXT NOT NULL CHECK (next_direct_debit_type IN ('first', 'followup', 'last')),
  last_used_at DATE,
  mandate_valid_until DATE NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (member_id, iban)
);

-- Ansprechpartner
CREATE TABLE contact_persons (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  member_id UUID NOT NULL REFERENCES members(id) ON DELETE CASCADE,
  contact_member_id UUID REFERENCES members(id) ON DELETE SET NULL,
  salutation TEXT,
  first_name TEXT NOT NULL,
  last_name TEXT NOT NULL,
  birth_date DATE,
  street_house_number TEXT NOT NULL,
  postal_code TEXT NOT NULL,
  city TEXT NOT NULL,
  nation TEXT,
  phone TEXT,
  phone_note TEXT,
  mobile TEXT,
  mobile_note TEXT,
  email TEXT,
  email_note TEXT,
  is_invoice_recipient BOOLEAN NOT NULL DEFAULT FALSE,
  is_correspondence_recipient BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Mitgliedsbeiträge (historisierbar)
CREATE TABLE membership_fees (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  member_id UUID NOT NULL REFERENCES members(id) ON DELETE CASCADE,
  fee_label TEXT NOT NULL,
  fee_type TEXT NOT NULL,
  assignment TEXT NOT NULL CHECK (assignment IN (
    '1_ideel', '2_vermoegen', '3_zweckbetrieb', '4_wirtschaft', '9_sammelposten'
  )),
  amount NUMERIC(10, 2) NOT NULL CHECK (amount > 0),
  period TEXT NOT NULL CHECK (period IN (
    'monthly', 'quarterly', 'half_yearly', 'yearly'
  )),
  maturity_date DATE NOT NULL,
  payment_method TEXT NOT NULL CHECK (payment_method IN (
    'sepa', 'invoice', 'transfer', 'cash'
  )),
  starts_at DATE NOT NULL,
  ends_at DATE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (member_id, starts_at)
);

-- Spender (für Nicht-Mitglieder)
CREATE TABLE donors (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  club_id UUID NOT NULL REFERENCES clubs(id) ON DELETE CASCADE,
  tax_id TEXT NOT NULL,
  first_name TEXT,
  last_name TEXT NOT NULL,
  company_name TEXT,
  street_house_number TEXT NOT NULL,
  postal_code TEXT NOT NULL,
  city TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------
-- 4. FINANZMODUL
-- ---------------------------------------------
-- Buchungskonten
CREATE TABLE booking_accounts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  club_id UUID NOT NULL REFERENCES clubs(id) ON DELETE CASCADE,
  majority_list TEXT NOT NULL CHECK (majority_list IN (
    '1_ideel', '2_vermoegen', '3_zweckbetrieb', '4_wirtschaft', '9_sammelposten'
  )),
  minority_list TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (club_id, majority_list, minority_list)
);

-- Belege
CREATE TABLE receipts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  club_id UUID NOT NULL REFERENCES clubs(id) ON DELETE CASCADE,
  type TEXT NOT NULL CHECK (type IN ('income', 'expense')),
  recipient TEXT NOT NULL,
  number TEXT NOT NULL,
  date DATE NOT NULL,
  position_assignment TEXT NOT NULL CHECK (position_assignment IN (
    '1_ideel', '2_vermoegen', '3_zweckbetrieb', '4_wirtschaft', '9_sammelposten'
  )),
  amount NUMERIC(12, 2) NOT NULL CHECK (amount != 0),
  is_booked BOOLEAN NOT NULL DEFAULT FALSE,
  note TEXT,
  position_tax_account TEXT,
  position_percentage NUMERIC(5, 2),
  donor_id UUID REFERENCES donors(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (club_id, number)
);

-- Buchungen
CREATE TABLE bookings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  club_id UUID NOT NULL REFERENCES clubs(id) ON DELETE CASCADE,
  booking_date DATE NOT NULL,
  valuta_date DATE NOT NULL,
  client_recipient TEXT NOT NULL,
  booking_text TEXT NOT NULL,
  purpose TEXT NOT NULL,
  amount NUMERIC(12, 2) NOT NULL,
  currency TEXT NOT NULL DEFAULT 'EUR',
  receipt_id UUID REFERENCES receipts(id) ON DELETE SET NULL,
  booking_account_id UUID NOT NULL REFERENCES booking_accounts(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Vereins-Bankverbindungen (SEPA)
CREATE TABLE club_bank_accounts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  club_id UUID NOT NULL REFERENCES clubs(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  account_holder TEXT NOT NULL,
  creditor_id TEXT NOT NULL,
  iban TEXT NOT NULL,
  bic TEXT,
  is_default BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (club_id, iban)
);

-- Bankverbindungen (Online-Banking)
CREATE TABLE bank_connections (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  club_id UUID NOT NULL REFERENCES clubs(id) ON DELETE CASCADE,
  bank_name TEXT NOT NULL,
  iban TEXT NOT NULL,
  bic TEXT,
  encrypted_credentials BYTEA NOT NULL,
  last_synced_at TIMESTAMPTZ,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (club_id, iban)
);

-- ---------------------------------------------
-- 5. ZUSATZMODULE
-- ---------------------------------------------
-- Vereinskalender
CREATE TABLE events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  club_id UUID NOT NULL REFERENCES clubs(id) ON DELETE CASCADE,
  date DATE NOT NULL,
  time TIME NOT NULL,
  description TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Dokumentenablage
CREATE TABLE documents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  club_id UUID NOT NULL REFERENCES clubs(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  content BYTEA NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- RBAC (Rechteverwaltung)
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE roles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE permissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE role_permissions (
  role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
  PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_roles (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  club_id UUID NOT NULL REFERENCES clubs(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, role_id, club_id)
);

-- ---------------------------------------------
-- 6. INDEX FÜR PERFORMANCE
-- ---------------------------------------------
-- Mitglieder
CREATE INDEX idx_members_birth_month ON members (club_id, EXTRACT(MONTH FROM birth_date));
CREATE INDEX idx_members_join_date ON members (club_id, joined_at);
CREATE INDEX idx_members_archived ON members (archived_at) WHERE archived_at IS NOT NULL;

-- Beiträge
CREATE INDEX idx_membership_fees_maturity ON membership_fees (maturity_date)
WHERE ends_at IS NULL;

-- Bankverbindungen
CREATE INDEX idx_bank_connections_last_sync ON bank_connections (last_synced_at);

-- Finanzen
CREATE INDEX idx_receipts_year ON receipts (club_id, EXTRACT(YEAR FROM date));
CREATE INDEX idx_bookings_valuta ON bookings (club_id, valuta_date);
