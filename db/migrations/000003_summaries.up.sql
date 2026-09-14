CREATE TABLE summaries
(
    id                  BIGSERIAL PRIMARY KEY,
    drug_id             BIGINT UNIQUE             NOT NULL,
    what_is_it_for      TEXT                      NOT NULL,
    posology            TEXT                      NOT NULL,
    adverse_effects     TEXT,
    drug_interactions   TEXT,
    contraindications   TEXT,
    side_effects        TEXT,
    when_to_seek_help   TEXT,
    mechanism_of_action TEXT,
    storage             TEXT,
    source_url          TEXT                      NOT NULL,
    reviewed_by         TEXT,
    reviewed_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at          TIMESTAMPTZ,
    CONSTRAINT fk_drug_id FOREIGN KEY (drug_id)
        REFERENCES drugs (id) ON DELETE CASCADE

);