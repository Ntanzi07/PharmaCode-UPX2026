ALTER TABLE summaries
    -- Leaflet sections that had no column yet
    ADD COLUMN missed_dose          TEXT, -- "What to do if you miss a dose"
    ADD COLUMN warnings             TEXT, -- "Warnings and precautions"
    -- Version of the leaflet that was summarized, to know when the summary is
    -- out of date compared to the leaflet published by Anvisa
    ADD COLUMN leaflet_expedient    VARCHAR(30), -- Anvisa filing (expediente) number
    ADD COLUMN leaflet_published_at DATE;        -- leaflet publication date
