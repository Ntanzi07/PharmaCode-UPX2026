-- name: CreateDrug :one
-- The active ingredients are not here: they live in drug_ingredients and are
-- written right after this, inside the same transaction.
INSERT INTO drugs (registration_number, brand_name, manufacturer)
VALUES ($1, $2, $3) RETURNING id;

-- name: UpdateDrug :execrows
UPDATE drugs
SET registration_number = $2,
    brand_name          = $3,
    manufacturer        = $4,
    updated_at          = NOW()
WHERE id = $1;

-- name: DeleteDrug :execrows
DELETE
FROM drugs
WHERE id = $1;

-- name: ListDrugs :many
-- @q is the search box of the panel: it matches the brand name, the company,
-- the registration number and the active ingredients, ignoring case and accents
-- (normalize_ingredient_name does that). An empty @q lists everything.
--
-- The search is written as IN (... UNION ...) and not as a plain OR because the
-- OR makes the planner give up on the trigram indexes of 000011 and read the
-- whole table: 600 ms against the Anvisa base instead of under 10.
SELECT d.id,
       d.registration_number,
       d.brand_name,
       COALESCE((SELECT array_agg(ai.name ORDER BY ai.name)
                 FROM drug_ingredients AS di
                          JOIN active_ingredients AS ai ON ai.id = di.ingredient_id
                 WHERE di.drug_id = d.id), '{}')::TEXT[] AS active_ingredients,
       d.manufacturer,
       d.updated_at
FROM drugs AS d
WHERE @q::TEXT = ''
   OR d.id IN (SELECT dd.id
               FROM drugs AS dd
               WHERE normalize_ingredient_name(
                             COALESCE(dd.brand_name, '') || ' ' || dd.manufacturer || ' ' ||
                             dd.registration_number
                     ) LIKE '%' || normalize_ingredient_name(@q) || '%'
               UNION
               SELECT di.drug_id
               FROM drug_ingredients AS di
                        JOIN active_ingredients AS ai ON ai.id = di.ingredient_id
               WHERE ai.normalized_name LIKE '%' || normalize_ingredient_name(@q) || '%')
ORDER BY d.brand_name NULLS FIRST LIMIT $1
OFFSET $2;

-- name: GetDrugByID :one
-- Used when a form opens already pointing at a drug: the picker only has the
-- id, and loading the whole list to find the name is what this change removes.
SELECT d.id,
       d.registration_number,
       d.brand_name,
       COALESCE((SELECT array_agg(ai.name ORDER BY ai.name)
                 FROM drug_ingredients AS di
                          JOIN active_ingredients AS ai ON ai.id = di.ingredient_id
                 WHERE di.drug_id = d.id), '{}')::TEXT[] AS active_ingredients,
       d.manufacturer,
       d.updated_at
FROM drugs AS d
WHERE d.id = $1;

-- name: GetSummaryByEAN :one
SELECT pe.ean,
       p.description,
       p.presentation_registration,
       d.registration_number,
       d.brand_name,
       COALESCE((SELECT array_agg(ai.name ORDER BY ai.name)
                 FROM drug_ingredients AS di
                          JOIN active_ingredients AS ai ON ai.id = di.ingredient_id
                 WHERE di.drug_id = d.id), '{}')::TEXT[] AS active_ingredients,
       d.manufacturer,
       s.what_is_it_for,
       s.posology,
       s.missed_dose,
       s.warnings,
       s.adverse_effects,
       s.drug_interactions,
       s.contraindications,
       s.side_effects,
       s.when_to_seek_help,
       s.mechanism_of_action,
       s.storage,
       s.source_url,
       s.leaflet_expedient,
       s.leaflet_published_at
FROM package_eans AS pe
         INNER JOIN packages AS p
                    ON p.id = pe.package_id
         INNER JOIN drugs AS d
                    ON d.id = p.drug_id
         LEFT JOIN summaries AS s
                   ON d.id = s.drug_id AND s.reviewed_at IS NOT NULL
WHERE pe.ean = $1;

-- name: GetDrugByEAN :one
SELECT pe.ean,
       d.id,
       d.registration_number,
       d.brand_name,
       COALESCE((SELECT array_agg(ai.name ORDER BY ai.name)
                 FROM drug_ingredients AS di
                          JOIN active_ingredients AS ai ON ai.id = di.ingredient_id
                 WHERE di.drug_id = d.id), '{}')::TEXT[] AS active_ingredients,
       d.manufacturer,
       d.updated_at,
       d.created_at
FROM package_eans AS pe
         INNER JOIN packages AS p
                    ON p.id = pe.package_id
         INNER JOIN drugs AS d
                    ON d.id = p.drug_id
WHERE pe.ean = $1;
