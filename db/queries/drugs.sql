-- name: GetDrugByEAN :one
SELECT p.ean,
       d.registration_number,
       d.brand_name,
       d.active_ingredient,
       d.manufacturer,
       p.description
FROM drugs AS d
         INNER JOIN packages AS p
                    ON d.id = p.drug_id
WHERE p.ean = $1;

-- name: GetSummaryByEAN :one
SELECT p.ean,
       p.description,
       d.registration_number,
       d.brand_name,
       d.active_ingredient,
       d.manufacturer,
       s.what_is_it_for,
       s.posology,
       s.adverse_effects,
       s.drug_interactions,
       s.contraindications,
       s.side_effects,
       s.when_to_seek_help,
       s.mechanism_of_action,
       s.storage,
       s.source_url
FROM drugs AS d
         INNER JOIN packages AS p
                    ON d.id = p.drug_id
         LEFT JOIN summaries AS s
                    ON d.id = s.drug_id AND s.reviewed_at IS NOT NULL
WHERE p.ean = $1;

