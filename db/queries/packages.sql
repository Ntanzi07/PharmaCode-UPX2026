-- name: CreatePackage :one
INSERT INTO packages (drug_id, ean, description)
SELECT d.id, $2, $3
FROM drugs d
WHERE d.registration_number = $1 RETURNING id;

-- name: UpdatePackage :execrows
UPDATE packages
SET drug_id     = $2,
    ean         = $3,
    description = $4,
    updated_at  = NOW()
WHERE id = $1;

-- name: DeletePackage :execrows
DELETE
FROM packages
WHERE id = $1;

-- name: ListPackages :many
SELECT id,
       drug_id,
       ean,
       description,
       updated_at
FROM packages
ORDER BY ean LIMIT $1
OFFSET $2;

-- name: GetPackageById :one
SELECT id,
       drug_id,
       ean,
       description,
       updated_at
FROM packages
WHERE id = $1;

-- name: ListPackagesByDrugID :many
SELECT id,
       drug_id,
       ean,
       description,
       updated_at
FROM packages
WHERE drug_id = $1
ORDER BY ean LIMIT $2
OFFSET $3;
