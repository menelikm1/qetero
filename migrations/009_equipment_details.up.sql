ALTER TABLE listings
    ADD COLUMN year         SMALLINT,
    ADD COLUMN total_hours  INT,
    ADD COLUMN last_serviced VARCHAR(20);
