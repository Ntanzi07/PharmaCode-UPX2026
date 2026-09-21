ALTER TABLE summaries
    -- Seções da bula que ainda não tinham onde entrar
    ADD COLUMN missed_dose          TEXT, -- "O que fazer se esquecer de usar"
    ADD COLUMN warnings             TEXT, -- "Advertências e precauções"
    -- Versão da bula que foi resumida, para saber quando o resumo ficou
    -- desatualizado em relação à bula publicada pela Anvisa
    ADD COLUMN leaflet_expedient    VARCHAR(30), -- número do expediente
    ADD COLUMN leaflet_published_at DATE;        -- data de publicação da bula
