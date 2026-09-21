-- Volta a ter um EAN por package. Se um package tiver mais de um EAN,
-- fica o menor; packages sem nenhum EAN são apagados (a coluna é NOT NULL).
ALTER TABLE packages
    ADD COLUMN ean VARCHAR(13);

UPDATE packages AS p
SET ean = (SELECT MIN(pe.ean) FROM package_eans AS pe WHERE pe.package_id = p.id);

DELETE
FROM packages
WHERE ean IS NULL;

ALTER TABLE packages
    ALTER COLUMN ean SET NOT NULL,
    ADD CONSTRAINT packages_ean_key UNIQUE (ean);

DROP TABLE IF EXISTS package_eans;
