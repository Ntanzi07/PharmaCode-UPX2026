CREATE TABLE drugs
(
    id                  BIGSERIAL PRIMARY KEY,
    registration_number VARCHAR(20) UNIQUE NOT NULL,
    brand_name          TEXT,
    active_ingredient   TEXT               NOT NULL,
    manufacturer        TEXT               NOT NULL,
    created_at          TIMESTAMPTZ        NOT NULL DEFAULT NOW()
);
