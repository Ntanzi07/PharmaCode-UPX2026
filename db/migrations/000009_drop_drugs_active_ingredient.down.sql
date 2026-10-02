-- Brings the column back and refills it from the link table, so rolling back
-- to the old code keeps working.
ALTER TABLE drugs
    ADD COLUMN active_ingredient TEXT;

UPDATE drugs AS d
SET active_ingredient = COALESCE(
        (SELECT string_agg(ai.name, ' + ' ORDER BY ai.name)
         FROM drug_ingredients AS di
                  JOIN active_ingredients AS ai ON ai.id = di.ingredient_id
         WHERE di.drug_id = d.id),
        '');

ALTER TABLE drugs
    ALTER COLUMN active_ingredient SET NOT NULL;
