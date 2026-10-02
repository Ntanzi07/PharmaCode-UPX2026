-- Last step of the normalization: the text column is not read by anything
-- anymore (queries, panel and import all use drug_ingredients since migration
-- 000008 and the code change that followed it).
ALTER TABLE drugs
    DROP COLUMN active_ingredient;
