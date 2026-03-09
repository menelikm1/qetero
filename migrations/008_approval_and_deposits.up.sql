-- Listing status: pending_review (default) → active | rejected
ALTER TABLE listings
    ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'pending_review'
        CHECK (status IN ('pending_review', 'active', 'rejected'));

-- Existing listings were created before review was required — mark them active
UPDATE listings SET status = 'active';

-- Deposit tracking on bookings
ALTER TABLE bookings
    ADD COLUMN deposit_ref    VARCHAR(100),
    ADD COLUMN deposit_status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (deposit_status IN ('pending', 'verified', 'rejected'));
