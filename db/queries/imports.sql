-- Queries used by the spreadsheet import. They upsert by the natural keys
-- (registration number, presentation registration, EAN) so importing the same
-- file twice updates the rows instead of duplicating them.
-- "(xmax = 0) AS created" is the Postgres trick to tell an INSERT from an UPDATE.

-- name: UpsertDrug :one
INSERT INTO drugs (registration_number, brand_name, active_ingredient, manufacturer)
VALUES ($1, $2, $3, $4)
ON CONFLICT (registration_number)
    DO UPDATE SET brand_name        = EXCLUDED.brand_name,
                  active_ingredient = EXCLUDED.active_ingredient,
                  manufacturer      = EXCLUDED.manufacturer,
                  updated_at        = NOW()
RETURNING id, (xmax = 0) AS created;

-- name: GetDrugIDByRegistration :one
SELECT id
FROM drugs
WHERE registration_number = $1;

-- name: UpsertPackage :one
INSERT INTO packages (drug_id, description, presentation_registration)
VALUES ($1, $2, $3)
ON CONFLICT (presentation_registration)
    DO UPDATE SET drug_id     = EXCLUDED.drug_id,
                  description = EXCLUDED.description,
                  updated_at  = NOW()
RETURNING id, (xmax = 0) AS created;

-- name: UpsertPackageEAN :one
-- The DO UPDATE keeps the current owner on purpose: the caller compares the
-- returned package_id with the expected one and reports an error when the EAN
-- already belongs to another package.
INSERT INTO package_eans (package_id, ean)
VALUES ($1, $2)
ON CONFLICT (ean)
    DO UPDATE SET package_id = package_eans.package_id
RETURNING package_id, (xmax = 0) AS created;

-- name: UpsertSummary :one
-- Importing leaflet text clears the review, exactly like editing it in the panel.
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
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
ON CONFLICT (drug_id)
    DO UPDATE SET what_is_it_for       = EXCLUDED.what_is_it_for,
                  posology             = EXCLUDED.posology,
                  missed_dose          = EXCLUDED.missed_dose,
                  warnings             = EXCLUDED.warnings,
                  adverse_effects      = EXCLUDED.adverse_effects,
                  drug_interactions    = EXCLUDED.drug_interactions,
                  contraindications    = EXCLUDED.contraindications,
                  side_effects         = EXCLUDED.side_effects,
                  when_to_seek_help    = EXCLUDED.when_to_seek_help,
                  mechanism_of_action  = EXCLUDED.mechanism_of_action,
                  storage              = EXCLUDED.storage,
                  source_url           = EXCLUDED.source_url,
                  leaflet_expedient    = EXCLUDED.leaflet_expedient,
                  leaflet_published_at = EXCLUDED.leaflet_published_at,
                  reviewed_by          = NULL,
                  reviewed_by_user_id  = NULL,
                  reviewed_at          = NULL,
                  updated_at           = NOW()
RETURNING id, (xmax = 0) AS created;
