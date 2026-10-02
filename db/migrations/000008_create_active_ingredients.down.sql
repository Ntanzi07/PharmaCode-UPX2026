UPDATE drugs AS d
SET active_ingredient = COALESCE(
        (SELECT string_agg(ai.name, ' + ' ORDER BY di.ingredient_id)
         FROM drug_ingredients AS di
                  JOIN active_ingredients AS ai ON ai.id = di.ingredient_id
         WHERE di.drug_id = d.id),
        d.active_ingredient);

DROP TABLE IF EXISTS drug_ingredients;
DROP TABLE IF EXISTS active_ingredients;
DROP FUNCTION IF EXISTS normalize_ingredient_name(TEXT);