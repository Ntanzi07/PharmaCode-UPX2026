DROP INDEX IF EXISTS idx_active_ingredients_search;
DROP INDEX IF EXISTS idx_drugs_search;

-- pg_trgm is left in place on purpose: other objects may use it.
