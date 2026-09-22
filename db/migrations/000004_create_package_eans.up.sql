-- The same presentation (package) can have more than one EAN on the
-- shelf at the same time (e.g. Advil 400 with an old and a new code). The EAN
-- moves out of packages into its own table, 1 package -> N EANs.
CREATE TABLE package_eans
(
    id         BIGSERIAL PRIMARY KEY,
    package_id BIGINT                    NOT NULL,
    ean        VARCHAR(13) UNIQUE        NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,

    CONSTRAINT fk_package_id
        FOREIGN KEY (package_id)
            REFERENCES packages (id) ON DELETE CASCADE
);

CREATE INDEX idx_package_eans_package_id ON package_eans (package_id);

-- Copy the existing EANs into the new table before dropping the column.
INSERT INTO package_eans (package_id, ean)
SELECT id, ean
FROM packages;

ALTER TABLE packages
    DROP COLUMN ean;
