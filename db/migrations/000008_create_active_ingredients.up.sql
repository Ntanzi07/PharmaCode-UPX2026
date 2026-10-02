CREATE EXTENSION IF NOT EXISTS unaccent;

CREATE FUNCTION normalize_ingredient_name(value TEXT) RETURNS TEXT
    LANGUAGE sql
    IMMUTABLE
    STRICT
    PARALLEL SAFE
AS
$$
SELECT btrim(regexp_replace(lower(unaccent('unaccent', value)), '\s+', ' ', 'g'))
$$;

CREATE TABLE active_ingredients
(
    id              BIGSERIAL PRIMARY KEY,
    name            TEXT                      NOT NULL,
    normalized_name TEXT GENERATED ALWAYS AS (normalize_ingredient_name(name)) STORED UNIQUE,
    created_at      TIMESTAMPTZ DEFAULT NOW() NOT NULL,

    CONSTRAINT chk_active_ingredient_name CHECK (btrim(name) <> '')
);

CREATE TABLE drug_ingredients
(
    drug_id       BIGINT                    NOT NULL,
    ingredient_id BIGINT                    NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT NOW() NOT NULL,

    CONSTRAINT pk_drug_ingredients PRIMARY KEY (drug_id, ingredient_id),

    CONSTRAINT fk_drug_id
        FOREIGN KEY (drug_id)
            REFERENCES drugs (id) ON DELETE CASCADE,

    CONSTRAINT fk_ingredient_id
        FOREIGN KEY (ingredient_id)
            REFERENCES active_ingredients (id) ON DELETE RESTRICT
);

CREATE INDEX idx_drug_ingredients_ingredient_id ON drug_ingredients (ingredient_id);

-- ---------------------------------------------------------------------------
-- Data migration: split the old text column.
-- Separators: "+" and ";" (how combinations come from the Bulário and CMED).
-- Commas are NOT split: they show up inside a single ingredient name.
-- ---------------------------------------------------------------------------
INSERT INTO active_ingredients (name)
SELECT DISTINCT ON (normalize_ingredient_name(part)) regexp_replace(btrim(part), '\s+', ' ', 'g')
FROM drugs,
     LATERAL unnest(regexp_split_to_array(drugs.active_ingredient, '\s*[+;]\s*')) AS part
WHERE btrim(part) <> ''
ORDER BY normalize_ingredient_name(part), btrim(part);

INSERT INTO drug_ingredients (drug_id, ingredient_id)
SELECT DISTINCT d.id, ai.id
FROM drugs AS d,
     LATERAL unnest(regexp_split_to_array(d.active_ingredient, '\s*[+;]\s*')) AS part
         JOIN active_ingredients AS ai
              ON ai.normalized_name = normalize_ingredient_name(part)
WHERE btrim(part) <> ''
ON CONFLICT DO NOTHING;
