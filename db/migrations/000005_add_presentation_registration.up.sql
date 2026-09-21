-- Registro da apresentação na Anvisa (13 dígitos): é a chave que liga a
-- embalagem com a tabela de preços da CMED. Opcional porque os packages que
-- já existem não têm esse dado.
ALTER TABLE packages
    ADD COLUMN presentation_registration VARCHAR(13) UNIQUE
        CONSTRAINT chk_presentation_registration
            CHECK (presentation_registration ~ '^[0-9]{13}$');
