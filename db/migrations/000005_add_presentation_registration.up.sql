-- Anvisa presentation registration (13 digits): the key that links the
-- package to the CMED price table. Optional because existing packages
-- don't have this data.
ALTER TABLE packages
    ADD COLUMN presentation_registration VARCHAR(13) UNIQUE
        CONSTRAINT chk_presentation_registration
            CHECK (presentation_registration ~ '^[0-9]{13}$');
