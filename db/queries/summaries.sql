-- name: CreateSummary :one
INSERT INTO summaries (drug_id,
                       what_is_it_for,
                       posology,
                       missed_dose,
                       warnings,
                       adverse_effects,
                       drug_interactions,
                       contraindications,
                       side_effects,
                       when_to_seek_help,
                       mechanism_of_action,
                       storage,
                       source_url,
                       leaflet_expedient,
                       leaflet_published_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15) RETURNING id;

-- name: UpdateSummary :execrows
-- The text changed, so the previous review no longer counts: the leaflet leaves the app until
-- a pharmacist reviews it again.
UPDATE summaries
SET what_is_it_for       = $2,
    posology             = $3,
    missed_dose          = $4,
    warnings             = $5,
    adverse_effects      = $6,
    drug_interactions    = $7,
    contraindications    = $8,
    side_effects         = $9,
    when_to_seek_help    = $10,
    mechanism_of_action  = $11,
    storage              = $12,
    source_url           = $13,
    leaflet_expedient    = $14,
    leaflet_published_at = $15,
    reviewed_by          = NULL,
    reviewed_by_user_id  = NULL,
    reviewed_at          = NULL,
    updated_at           = NOW()
WHERE id = $1;

-- name: ReviewSummary :execrows
UPDATE summaries
SET reviewed_by         = @reviewed_by,
    reviewed_by_user_id = @reviewed_by_user_id,
    reviewed_at         = NOW(),
    updated_at          = NOW()
WHERE id = @id;

-- name: DeleteSummary :execrows
DELETE
FROM summaries
WHERE id = $1;

-- name: GetSummaryByID :one
SELECT id,
       drug_id,
       what_is_it_for,
       posology,
       missed_dose,
       warnings,
       adverse_effects,
       drug_interactions,
       contraindications,
       side_effects,
       when_to_seek_help,
       mechanism_of_action,
       storage,
       source_url,
       leaflet_expedient,
       leaflet_published_at,
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
       missed_dose,
       warnings,
       adverse_effects,
       drug_interactions,
       contraindications,
       side_effects,
       when_to_seek_help,
       mechanism_of_action,
       storage,
       source_url,
       leaflet_expedient,
       leaflet_published_at,
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
       s.leaflet_expedient,
       s.leaflet_published_at,
       s.reviewed_by,
       s.reviewed_at,
       s.updated_at
FROM summaries AS s
         INNER JOIN drugs AS d ON d.id = s.drug_id
ORDER BY d.brand_name NULLS FIRST LIMIT $1
OFFSET $2;
