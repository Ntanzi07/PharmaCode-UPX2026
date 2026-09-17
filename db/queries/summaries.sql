-- name: CreateSummary :one
INSERT INTO summaries (drug_id,
                       what_is_it_for,
                       posology,
                       adverse_effects,
                       drug_interactions,
                       contraindications,
                       side_effects,
                       when_to_seek_help,
                       mechanism_of_action,
                       storage,
                       source_url)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id;

-- name: UpdateSummary :execrows
UPDATE summaries
SET what_is_it_for      = $2,
    posology            = $3,
    adverse_effects     = $4,
    drug_interactions   = $5,
    contraindications   = $6,
    side_effects        = $7,
    when_to_seek_help   = $8,
    mechanism_of_action = $9,
    storage             = $10,
    source_url          = $11,
    updated_at          = NOW()
WHERE id = $1;

-- name: ReviewSummary :execrows
UPDATE summaries
SET reviewed_by = $2,
    reviewed_at = NOW(),
    updated_at  = NOW()
WHERE id = $1;

-- name: DeleteSummary :execrows
DELETE
FROM summaries
WHERE id = $1;

-- name: GetSummaryByID :one
SELECT id,
       drug_id,
       what_is_it_for,
       posology,
       adverse_effects,
       drug_interactions,
       contraindications,
       side_effects,
       when_to_seek_help,
       mechanism_of_action,
       storage,
       source_url,
       reviewed_by,
       reviewed_at,
       updated_at
FROM summaries
WHERE id = $1;

-- name: GetSummaryByDrugID :one
SELECT id,
       drug_id,
       what_is_it_for,
       posology,
       adverse_effects,
       drug_interactions,
       contraindications,
       side_effects,
       when_to_seek_help,
       mechanism_of_action,
       storage,
       source_url,
       reviewed_by,
       reviewed_at,
       updated_at
FROM summaries
WHERE drug_id = $1;

-- name: ListSummaries :many
SELECT s.id,
       s.drug_id,
       d.brand_name,
       d.active_ingredient,
       s.what_is_it_for,
       s.source_url,
       s.reviewed_by,
       s.reviewed_at,
       s.updated_at
FROM summaries AS s
         INNER JOIN drugs AS d ON d.id = s.drug_id
ORDER BY d.brand_name NULLS FIRST LIMIT $1
OFFSET $2;
