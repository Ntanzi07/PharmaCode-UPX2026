-- The panel searches drugs with a LIKE that has no anchor ("%advil%"), and a
-- btree index cannot help with that. pg_trgm indexes the three-letter pieces of
-- the text, which is what the search can seek on.
--
-- This stopped being a nicety when cmd/anvisa-import started loading the whole
-- Anvisa base: over 29,000 drugs the same search takes about 600 ms per
-- keystroke without these indexes, and under 10 ms with them.

-- pg_trgm is a trusted extension (PostgreSQL 13+), so the application user can
-- create it, like unaccent in 000008.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- The text the search looks at: brand name, company and registration number,
-- normalized by the same function the ingredients use, so "ACHÉ" matches "ache".
CREATE INDEX idx_drugs_search ON drugs USING GIN (
    normalize_ingredient_name(
            COALESCE(brand_name, '') || ' ' || manufacturer || ' ' || registration_number
    ) gin_trgm_ops);

-- Searching by active ingredient goes through the other table, which needs its
-- own index: normalized_name is UNIQUE (btree), useless for "%dipirona%".
CREATE INDEX idx_active_ingredients_search ON active_ingredients USING GIN (normalized_name gin_trgm_ops);
