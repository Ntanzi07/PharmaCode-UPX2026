ALTER TABLE summaries
    DROP COLUMN IF EXISTS missed_dose,
    DROP COLUMN IF EXISTS warnings,
    DROP COLUMN IF EXISTS leaflet_expedient,
    DROP COLUMN IF EXISTS leaflet_published_at;
