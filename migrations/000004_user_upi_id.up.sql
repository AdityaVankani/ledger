ALTER TABLE users
    ADD COLUMN upi_id TEXT NOT NULL DEFAULT ''
    CHECK (length(upi_id) <= 320);