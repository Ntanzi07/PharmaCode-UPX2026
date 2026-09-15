-- name: CreateDrug :one
INSERT INTO drugs (registration_number, brand_name, active_ingredient, manufacturer)
VALUES ($1, $2, $3, $4) RETURNING id;

-- name: UpdateDrug :exec
UPDATE drugs
SET registration_number = $2,
    brand_name          = $3,
    active_ingredient   = $4,
    manufacturer        = $5,
    updated_at          = NOW()
WHERE id = $1;

-- name: DeleteDrug :execrows
DELETE
FROM drugs
WHERE id = $1;

-- name: ListDrugs :many
SELECT id,
       registration_number,
       brand_name,
       active_ingredient,
       manufacturer,
       updated_at
FROM drugs
ORDER BY brand_name NULLS FIRST LIMIT $1
OFFSET $2;

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

-- name: GetDrugByEAN :one
SELECT p.ean,
       d.id,
       d.registration_number,
       d.brand_name,
       d.active_ingredient,
       d.manufacturer,
       d.updated_at,
       d.created_at
FROM drugs AS d
         INNER JOIN packages AS p
                    ON d.id = p.drug_id
WHERE p.ean = $1;