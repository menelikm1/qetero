CREATE TABLE listing_blocks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id  UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    owner_id    UUID NOT NULL REFERENCES users(id),
    start_date  DATE NOT NULL,
    end_date    DATE NOT NULL,
    note        VARCHAR(100),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_listing_blocks_listing_id ON listing_blocks(listing_id);
