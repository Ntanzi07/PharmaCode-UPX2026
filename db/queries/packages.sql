-- name: CreatePackage :one
-- Cria o package e os EANs dele num único comando (atômico): se um EAN já
-- existir, nada é gravado. Sem remédio com esse registration_number, não
-- retorna linha (pgx.ErrNoRows).
WITH new_package AS (
    INSERT INTO packages (drug_id, description, presentation_registration)
        SELECT d.id, @description::text, sqlc.narg(presentation_registration)::varchar
        FROM drugs AS d
        WHERE d.registration_number = @registration_number::varchar
        RETURNING id),
     new_eans AS (
         INSERT INTO package_eans (package_id, ean)
             SELECT np.id, e.ean
             FROM new_package AS np,
                  unnest(@eans::text[]) AS e(ean))
SELECT id
FROM new_package;

-- name: UpdatePackage :execrows
-- Atualiza o package e sincroniza a lista de EANs: remove os que saíram da
-- lista e insere os novos, tudo num único comando.
WITH updated AS (
    UPDATE packages AS p
        SET drug_id = @drug_id,
            description = @description,
            presentation_registration = sqlc.narg(presentation_registration),
            updated_at = NOW()
        WHERE p.id = @id
        RETURNING p.id),
     removed_eans AS (
         DELETE FROM package_eans AS pe
             USING updated AS u
             WHERE pe.package_id = u.id
                 AND NOT (pe.ean = ANY (@eans::text[]))),
     added_eans AS (
         INSERT INTO package_eans (package_id, ean)
             SELECT u.id, e.ean
             FROM updated AS u,
                  unnest(@eans::text[]) AS e(ean)
             WHERE NOT EXISTS (SELECT 1
                               FROM package_eans AS x
                               WHERE x.package_id = u.id
                                 AND x.ean = e.ean))
SELECT id
FROM updated;

-- name: DeletePackage :execrows
DELETE
FROM packages
WHERE id = $1;

-- name: ListPackages :many
SELECT p.id,
       p.drug_id,
       p.description,
       p.presentation_registration,
       COALESCE(array_agg(pe.ean ORDER BY pe.ean) FILTER (WHERE pe.ean IS NOT NULL), '{}')::text[] AS eans,
       p.updated_at
FROM packages AS p
         LEFT JOIN package_eans AS pe ON pe.package_id = p.id
GROUP BY p.id
ORDER BY p.id LIMIT $1
OFFSET $2;

-- name: GetPackageById :one
SELECT p.id,
       p.drug_id,
       p.description,
       p.presentation_registration,
       COALESCE(array_agg(pe.ean ORDER BY pe.ean) FILTER (WHERE pe.ean IS NOT NULL), '{}')::text[] AS eans,
       p.updated_at
FROM packages AS p
         LEFT JOIN package_eans AS pe ON pe.package_id = p.id
WHERE p.id = $1
GROUP BY p.id;

-- name: ListPackagesByDrugID :many
SELECT p.id,
       p.drug_id,
       p.description,
       p.presentation_registration,
       COALESCE(array_agg(pe.ean ORDER BY pe.ean) FILTER (WHERE pe.ean IS NOT NULL), '{}')::text[] AS eans,
       p.updated_at
FROM packages AS p
         LEFT JOIN package_eans AS pe ON pe.package_id = p.id
WHERE p.drug_id = $1
GROUP BY p.id
ORDER BY p.id LIMIT $2
OFFSET $3;
