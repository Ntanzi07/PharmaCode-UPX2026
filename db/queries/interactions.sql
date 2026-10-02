-- Rules about what happens when two active ingredients are taken together.
-- Pairs are always stored with the smaller id first (chk_pair_order), so every
-- query here orders the ids with LEAST/GREATEST before touching the table.

-- name: UpsertIngredientInteraction :one
INSERT INTO ingredient_interactions (ingredient_a_id, ingredient_b_id, severity, description, recommendation, source_url)
VALUES (LEAST(@ingredient_a_id::BIGINT, @ingredient_b_id::BIGINT),
        GREATEST(@ingredient_a_id::BIGINT, @ingredient_b_id::BIGINT),
        @severity, @description, @recommendation, @source_url)
ON CONFLICT (ingredient_a_id, ingredient_b_id)
    DO UPDATE SET severity       = EXCLUDED.severity,
                  description    = EXCLUDED.description,
                  recommendation = EXCLUDED.recommendation,
                  source_url     = EXCLUDED.source_url,
                  updated_at     = NOW()
RETURNING ingredient_a_id, ingredient_b_id, (xmax = 0) AS created;

-- name: DeleteIngredientInteraction :execrows
DELETE
FROM ingredient_interactions
WHERE ingredient_a_id = LEAST(@ingredient_a_id::BIGINT, @ingredient_b_id::BIGINT)
  AND ingredient_b_id = GREATEST(@ingredient_a_id::BIGINT, @ingredient_b_id::BIGINT);

-- name: ListIngredientInteractions :many
SELECT ii.ingredient_a_id,
       a.name AS ingredient_a,
       ii.ingredient_b_id,
       b.name AS ingredient_b,
       ii.severity,
       ii.description,
       ii.recommendation,
       ii.source_url,
       ii.updated_at
FROM ingredient_interactions AS ii
         JOIN active_ingredients AS a ON a.id = ii.ingredient_a_id
         JOIN active_ingredients AS b ON b.id = ii.ingredient_b_id
ORDER BY a.name, b.name LIMIT $1
OFFSET $2;

-- name: CountIngredientInteractions :one
SELECT COUNT(*)
FROM ingredient_interactions;

-- name: FindInteractionsBetweenEANs :many
-- The check the app calls: every pair of active ingredients coming from
-- DIFFERENT boxes is compared against the rules.
--   scanned -> one row per (drug, ingredient) behind the scanned barcodes
--   pairs   -> every combination of ingredients from two different drugs
-- The combination inside a single drug is left out on purpose: that mix is
-- already approved by Anvisa.
WITH scanned AS (SELECT DISTINCT d.id AS drug_id, di.ingredient_id
                 FROM package_eans AS pe
                          JOIN packages AS p ON p.id = pe.package_id
                          JOIN drugs AS d ON d.id = p.drug_id
                          JOIN drug_ingredients AS di ON di.drug_id = d.id
                 WHERE pe.ean = ANY (@eans::TEXT[])),
     pairs AS (SELECT LEAST(x.ingredient_id, y.ingredient_id)    AS ingredient_a_id,
                      GREATEST(x.ingredient_id, y.ingredient_id) AS ingredient_b_id,
                      x.drug_id                                  AS drug_a_id,
                      y.drug_id                                  AS drug_b_id
               FROM scanned AS x
                        JOIN scanned AS y ON x.drug_id < y.drug_id)
SELECT DISTINCT a.name   AS ingredient_a,
                b.name   AS ingredient_b,
                ii.severity,
                ii.description,
                ii.recommendation,
                ii.source_url,
                pairs.drug_a_id,
                pairs.drug_b_id
FROM pairs
         JOIN ingredient_interactions AS ii
              ON ii.ingredient_a_id = pairs.ingredient_a_id
                  AND ii.ingredient_b_id = pairs.ingredient_b_id
         JOIN active_ingredients AS a ON a.id = ii.ingredient_a_id
         JOIN active_ingredients AS b ON b.id = ii.ingredient_b_id
ORDER BY ii.severity, a.name, b.name;
