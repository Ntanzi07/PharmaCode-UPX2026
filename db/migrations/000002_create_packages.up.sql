CREATE TABLE packages
(
    id          BIGSERIAL PRIMARY KEY,
    drug_id     BIGINT                    NOT NULL,
    ean         VARCHAR(13) UNIQUE        NOT NULL,
    description TEXT                      NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW() NOT NULL,

    CONSTRAINT fk_drug_id
        FOREIGN KEY (drug_id)
            REFERENCES drugs (id) ON DELETE CASCADE
);

CREATE INDEX idx_packages_drug_id ON packages (drug_id);