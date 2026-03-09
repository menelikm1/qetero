-- Add Ethiopia-specific equipment categories
ALTER TYPE listing_category ADD VALUE IF NOT EXISTS 'water_truck';
ALTER TYPE listing_category ADD VALUE IF NOT EXISTS 'concrete_mixer';
ALTER TYPE listing_category ADD VALUE IF NOT EXISTS 'dump_truck';
ALTER TYPE listing_category ADD VALUE IF NOT EXISTS 'dozer';
ALTER TYPE listing_category ADD VALUE IF NOT EXISTS 'roller';

-- Add payment method field to bookings (manual payments for MVP)
ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS payment_method VARCHAR(50);

-- Make phone required, email optional
-- Set placeholder for any existing rows that have no phone (dev/migration safety)
UPDATE users SET phone = 'unknown-' || id::text WHERE phone IS NULL OR phone = '';
ALTER TABLE users
    ALTER COLUMN phone SET NOT NULL,
    ALTER COLUMN email DROP NOT NULL;

-- Add unique constraint on phone
ALTER TABLE users ADD CONSTRAINT users_phone_unique UNIQUE (phone);
-- Email unique only when provided
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique ON users(email) WHERE email IS NOT NULL;
