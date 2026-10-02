-- Known interactions between two active ingredients. This is the knowledge the
-- database did not have: which combination is a problem, how serious it is and
-- what the person should do.
--
-- Each pair is stored once, in a canonical order (a < b). Without that rule the
-- same pair could be registered twice, in opposite order, with contradicting
-- text.
CREATE TABLE ingredient_interactions
(
    ingredient_a_id BIGINT                    NOT NULL,
    ingredient_b_id BIGINT                    NOT NULL,
    severity        VARCHAR(12)               NOT NULL,
    -- description: what happens, in plain language
    description     TEXT                      NOT NULL,
    -- recommendation: what to do (avoid, space the doses, see a doctor)
    recommendation  TEXT,
    -- source_url: where the information came from
    source_url      TEXT                      NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at      TIMESTAMPTZ,

    CONSTRAINT pk_ingredient_interactions PRIMARY KEY (ingredient_a_id, ingredient_b_id),

    CONSTRAINT chk_pair_order CHECK (ingredient_a_id < ingredient_b_id),
    CONSTRAINT chk_severity CHECK (severity IN ('grave', 'moderada', 'leve')),

    CONSTRAINT fk_ingredient_a_id
        FOREIGN KEY (ingredient_a_id)
            REFERENCES active_ingredients (id) ON DELETE RESTRICT,

    CONSTRAINT fk_ingredient_b_id
        FOREIGN KEY (ingredient_b_id)
            REFERENCES active_ingredients (id) ON DELETE RESTRICT
);

-- The primary key covers lookups starting at ingredient_a_id; listing every
-- rule of one ingredient also needs the other side.
CREATE INDEX idx_ingredient_interactions_b ON ingredient_interactions (ingredient_b_id);
