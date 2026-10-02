-- Example data: Advil 12h (ibuprofen 600 mg, extended release).
--
-- Sources:
--   drug + leaflet  -> Anvisa Bulário Eletrônico, registration 192900007
--   packages + EANs -> CMED price list (REGISTRO / EAN 1..3 columns)
--
-- The leaflet text below is a SHORT EXAMPLE written for demos. Before showing
-- this data as real, check it against the official leaflet PDF and let a
-- pharmacist review it in the panel.
--
-- Safe to run more than once: existing rows are updated, not duplicated.

BEGIN;

-- 1. Drug (Bulário: "Detalhe do Produto")
INSERT INTO drugs (registration_number, brand_name, manufacturer)
VALUES ('192900007',
        'Advil 12h',
        'PF Consumer Healthcare Brazil Importadora e Distribuidora de Medicamentos Ltda')
ON CONFLICT (registration_number)
    DO UPDATE SET brand_name   = EXCLUDED.brand_name,
                  manufacturer = EXCLUDED.manufacturer,
                  updated_at   = NOW();

-- 1b. Active ingredients (one row per ingredient; Advil 12h has only one).
--     Matching is by normalized_name, so running this twice reuses the row.
INSERT INTO active_ingredients (name)
VALUES ('ibuprofeno')
ON CONFLICT (normalized_name) DO UPDATE SET name = active_ingredients.name;

INSERT INTO drug_ingredients (drug_id, ingredient_id)
SELECT d.id, ai.id
FROM drugs AS d,
     active_ingredients AS ai
WHERE d.registration_number = '192900007'
  AND ai.normalized_name = normalize_ingredient_name('ibuprofeno')
ON CONFLICT DO NOTHING;

-- 2. Packages (CMED: one row per presentation; REGISTRO has 13 digits and
--    starts with the drug's 9-digit registration number)
INSERT INTO packages (drug_id, description, presentation_registration)
SELECT d.id, v.description, v.presentation_registration
FROM drugs AS d,
     (VALUES ('1929000070034', '600 mg comprimido revestido de liberação prolongada, caixa com 10'),
             ('1929000070069', '600 mg comprimido revestido de liberação prolongada, caixa com 20'),
             ('1929000070107', '600 mg comprimido revestido de liberação prolongada, caixa com 30'))
         AS v(presentation_registration, description)
WHERE d.registration_number = '192900007'
ON CONFLICT (presentation_registration)
    DO UPDATE SET description = EXCLUDED.description,
                  updated_at  = NOW();

-- 3. EANs (CMED columns EAN 1, EAN 2, EAN 3 - one row per filled code)
INSERT INTO package_eans (package_id, ean)
SELECT p.id, v.ean
FROM packages AS p
         JOIN (VALUES ('1929000070034', '7896015592752'),
                      ('1929000070069', '7896015592769'),
                      ('1929000070107', '7896015592745'))
    AS v(presentation_registration, ean)
              ON p.presentation_registration = v.presentation_registration
ON CONFLICT (ean) DO NOTHING;

-- 4. Simplified leaflet (written from the Bulário PDF)
INSERT INTO summaries (drug_id,
                       what_is_it_for,
                       posology,
                       missed_dose,
                       warnings,
                       adverse_effects,
                       drug_interactions,
                       contraindications,
                       side_effects,
                       when_to_seek_help,
                       mechanism_of_action,
                       storage,
                       source_url,
                       leaflet_expedient,
                       leaflet_published_at)
SELECT d.id,
       'Alivia dores leves a moderadas por até 12 horas e baixa a febre: dor de cabeça, dor muscular, dor nas costas, dor de dente e cólica menstrual.',
       'Adultos e maiores de 12 anos: 1 comprimido a cada 12 horas, com água e de preferência após comer. Não passe de 2 comprimidos por dia. Engula inteiro, sem partir ou mastigar.',
       'Tome assim que lembrar. Se já estiver perto do horário da próxima dose, pule a esquecida. Nunca tome duas doses juntas.',
       'Fale com o médico antes de usar se você tem pressão alta, problemas no coração, nos rins, no fígado ou no estômago, ou se está grávida ou amamentando. Não use no último trimestre da gravidez. Evite bebida alcoólica durante o tratamento.',
       'Podem aparecer azia, dor no estômago, enjoo, diarreia, tontura ou dor de cabeça.',
       'Evite usar junto com outros anti-inflamatórios (inclusive AAS), anticoagulantes como a varfarina, corticoides, lítio, metotrexato e remédios para pressão alta. Avise o médico sobre tudo o que você toma.',
       'Não use se você tem alergia ao ibuprofeno ou a outros anti-inflamatórios, se já teve asma ou urticária ao usar AAS, se tem úlcera ou sangramento no estômago, ou se está no último trimestre da gravidez.',
       'Os mais comuns afetam o estômago: azia, má digestão e dor. Reações na pele, inchaço e falta de ar são raros, mas exigem atenção.',
       'Procure um serviço de saúde se aparecer vômito com sangue, fezes escuras, dor forte no estômago, inchaço no rosto, falta de ar, manchas na pele, ou se a dor ou a febre continuarem depois de 3 dias de uso.',
       'O ibuprofeno reduz as substâncias que o corpo produz durante a inflamação (prostaglandinas), e com isso diminui a dor e a febre. A liberação prolongada faz o efeito durar cerca de 12 horas.',
       'Guarde em temperatura ambiente (15 °C a 30 °C), longe da umidade e do sol. Mantenha fora do alcance de crianças.',
       'https://consultas.anvisa.gov.br/#/bulario/q/?numeroRegistro=192900007',
       NULL, -- leaflet_expedient: fill in with the filing number shown in the Bulário
       NULL  -- leaflet_published_at: fill in with the leaflet publication date
FROM drugs AS d
WHERE d.registration_number = '192900007'
ON CONFLICT (drug_id)
    DO UPDATE SET what_is_it_for      = EXCLUDED.what_is_it_for,
                  posology            = EXCLUDED.posology,
                  missed_dose         = EXCLUDED.missed_dose,
                  warnings            = EXCLUDED.warnings,
                  adverse_effects     = EXCLUDED.adverse_effects,
                  drug_interactions   = EXCLUDED.drug_interactions,
                  contraindications   = EXCLUDED.contraindications,
                  side_effects        = EXCLUDED.side_effects,
                  when_to_seek_help   = EXCLUDED.when_to_seek_help,
                  mechanism_of_action = EXCLUDED.mechanism_of_action,
                  storage             = EXCLUDED.storage,
                  source_url          = EXCLUDED.source_url,
                  updated_at          = NOW();

COMMIT;

-- The leaflet is NOT marked as reviewed on purpose: open the panel as a
-- reviewer (pharmacist) and press "Revisar". Only then does it show up in
-- GET /drugs/ean/{ean}, which is what the app calls.
