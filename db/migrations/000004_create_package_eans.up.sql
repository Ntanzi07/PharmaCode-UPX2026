-- Uma mesma apresentação (package) pode ter mais de um EAN convivendo na
-- prateleira (ex.: Advil 400 com código antigo e novo). O EAN sai de
-- packages e vai para uma tabela própria, 1 package -> N EANs.
CREATE TABLE package_eans
(
    id         BIGSERIAL PRIMARY KEY,
    package_id BIGINT                    NOT NULL,
    ean        VARCHAR(13) UNIQUE        NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,

    CONSTRAINT fk_package_id
        FOREIGN KEY (package_id)
            REFERENCES packages (id) ON DELETE CASCADE
);

CREATE INDEX idx_package_eans_package_id ON package_eans (package_id);

-- Leva os EANs que já existem para a tabela nova antes de apagar a coluna.
INSERT INTO package_eans (package_id, ean)
SELECT id, ean
FROM packages;

ALTER TABLE packages
    DROP COLUMN ean;
