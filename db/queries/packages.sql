

-- name: CreatePackage :one
INSERT INTO packages (drug_id, ean, description)
SELECT d.id, $2, $3
FROM drugs d
WHERE d.registration_number = $1
    RETURNING id;