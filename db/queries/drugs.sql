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