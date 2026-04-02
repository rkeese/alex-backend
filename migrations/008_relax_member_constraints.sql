-- Migration 008: Relax member constraints for imported data editing
ALTER TABLE members ALTER COLUMN street_house_number DROP NOT NULL;
ALTER TABLE members ALTER COLUMN postal_code DROP NOT NULL;
ALTER TABLE members ALTER COLUMN city DROP NOT NULL;
ALTER TABLE members ALTER COLUMN joined_at DROP NOT NULL;
ALTER TABLE members ALTER COLUMN birth_date DROP NOT NULL;
