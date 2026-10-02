-- Example data: a few interaction rules between active ingredients.
--
-- These rules are what POST /interactions compares the scanned boxes against:
-- without them the check always comes back empty (which means "no rule
-- registered", NOT "safe to take together").
--
-- The texts below are SHORT EXAMPLES written for demos, in the plain language
-- the app shows. Before using this data as real, check every pair against the
-- official leaflet and have a pharmacist review it.
--
-- The pair has no order: a,b and b,a are the same rule, and the database keeps
-- the smaller id first (chk_pair_order).
--
-- Safe to run more than once: existing rules are updated, not duplicated.

BEGIN;

-- 1. The active ingredients the rules below talk about. Matching is by
--    normalized_name, so "Ácido Acetilsalicílico" reuses an existing row.
INSERT INTO active_ingredients (name)
VALUES ('ibuprofeno'),
       ('varfarina'),
       ('ácido acetilsalicílico'),
       ('enalapril'),
       ('metotrexato')
ON CONFLICT (normalized_name) DO UPDATE SET name = active_ingredients.name;

-- 2. The rules. LEAST/GREATEST puts the pair in the order the table requires,
--    so the order written here doesn't matter.
INSERT INTO ingredient_interactions (ingredient_a_id, ingredient_b_id, severity,
                                     description, recommendation, source_url)
SELECT LEAST(a.id, b.id),
       GREATEST(a.id, b.id),
       v.severity,
       v.description,
       v.recommendation,
       v.source_url
FROM (VALUES ('ibuprofeno', 'varfarina', 'grave',
              'Os dois juntos aumentam muito o risco de sangramento, principalmente no estômago e no intestino.',
              'Não use os dois juntos. Procure o médico para trocar o analgésico.',
              'https://consultas.anvisa.gov.br/#/bulario/q/?numeroRegistro=192900007'),

             ('ibuprofeno', 'ácido acetilsalicílico', 'moderada',
              'São dois anti-inflamatórios da mesma família: juntos irritam mais o estômago e um pode cortar o efeito protetor do outro no coração.',
              'Evite tomar os dois no mesmo dia. Se usa AAS por indicação do médico, pergunte qual analgésico pode usar.',
              'https://consultas.anvisa.gov.br/#/bulario/q/?numeroRegistro=192900007'),

             ('ibuprofeno', 'enalapril', 'moderada',
              'O ibuprofeno pode diminuir o efeito do remédio de pressão e, em uso prolongado, sobrecarregar os rins.',
              'Use o ibuprofeno pelo menor tempo possível e avise o médico que você trata pressão alta.',
              'https://consultas.anvisa.gov.br/#/bulario/q/?numeroRegistro=192900007'),

             ('ibuprofeno', 'metotrexato', 'grave',
              'O ibuprofeno faz o corpo demorar mais para eliminar o metotrexato, o que aumenta o risco de intoxicação.',
              'Não use por conta própria. Fale com o médico que acompanha o tratamento.',
              'https://consultas.anvisa.gov.br/#/bulario/q/?numeroRegistro=192900007'))
         AS v(name_a, name_b, severity, description, recommendation, source_url)
         JOIN active_ingredients AS a ON a.normalized_name = normalize_ingredient_name(v.name_a)
         JOIN active_ingredients AS b ON b.normalized_name = normalize_ingredient_name(v.name_b)
ON CONFLICT (ingredient_a_id, ingredient_b_id)
    DO UPDATE SET severity       = EXCLUDED.severity,
                  description    = EXCLUDED.description,
                  recommendation = EXCLUDED.recommendation,
                  source_url     = EXCLUDED.source_url,
                  updated_at     = NOW();

COMMIT;

-- To see a rule fire end to end, register a drug whose active ingredient is one
-- of the names above (varfarina, for example) with its own EAN, and then POST
-- /interactions with that EAN plus one of the Advil 12h EANs.
