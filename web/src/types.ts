export type Page<T> = { data: T[]; limit: number; offset: number }

export type Drug = {
  id: number
  registration_number: string
  brand_name: string | null
  /** Um por princípio ativo: uma associação como a Neosaldina tem três */
  active_ingredients: string[]
  manufacturer: string
  updated_at: string | null
}

export type DrugInput = {
  registration_number: string
  brand_name: string
  active_ingredients: string[]
  manufacturer: string
}

export type Package = {
  id: number
  drug_id: number
  description: string
  /** Anvisa presentation registration (13 digits), the link to CMED */
  presentation_registration: string | null
  /** A presentation can have more than one barcode on the shelf */
  eans: string[]
  updated_at: string | null
}

type PackageFields = { eans: string[]; description: string; presentation_registration: string }
export type PackageCreate = PackageFields & { registration_number: string }
export type PackageUpdate = PackageFields & { drug_id: number }

export type SummaryListItem = {
  id: number
  drug_id: number
  brand_name: string | null
  active_ingredients: string[]
  what_is_it_for: string
  source_url: string
  leaflet_expedient: string | null
  leaflet_published_at: string | null
  reviewed_by: string | null
  reviewed_at: string | null
  updated_at: string | null
}

export const SUMMARY_TEXT_FIELDS = [
  { key: 'what_is_it_for', label: 'Para que serve', required: true },
  { key: 'posology', label: 'Posologia', required: true },
  { key: 'missed_dose', label: 'O que fazer se esquecer de usar', required: false },
  { key: 'adverse_effects', label: 'Reações adversas', required: false },
  { key: 'drug_interactions', label: 'Interações medicamentosas', required: false },
  { key: 'contraindications', label: 'Contraindicações', required: false },
  { key: 'warnings', label: 'Advertências e precauções', required: false },
  { key: 'side_effects', label: 'Efeitos colaterais', required: false },
  { key: 'when_to_seek_help', label: 'Quando procurar ajuda', required: false },
  { key: 'mechanism_of_action', label: 'Mecanismo de ação', required: false },
  { key: 'storage', label: 'Armazenamento', required: false },
] as const

export type SummaryTextKey = (typeof SUMMARY_TEXT_FIELDS)[number]['key']

/** Version of the official leaflet that was summarized (filing number and publication date YYYY-MM-DD) */
type LeafletVersion = { leaflet_expedient: string; leaflet_published_at: string }

export type SummaryInput = Record<SummaryTextKey, string> & { source_url: string } & LeafletVersion

export type Summary = { id: number; drug_id: number; source_url: string } & {
  [K in SummaryTextKey]: string | null
} & { [K in keyof LeafletVersion]: string | null } & {
  reviewed_by: string | null
  reviewed_at: string | null
  updated_at: string | null
}

export type EanSummary = {
  ean: string
  description: string
  presentation_registration: string | null
  registration_number: string
  brand_name: string | null
  active_ingredients: string[]
  manufacturer: string
  source_url: string | null
  leaflet_expedient: string | null
  leaflet_published_at: string | null
} & { [K in SummaryTextKey]: string | null }

// ---------- users and login ----------
export type Role = 'editor' | 'reviewer' | 'admin'

export const ROLE_LABEL: Record<Role, string> = {
  editor: 'Editor',
  reviewer: 'Farmacêutico (revisor)',
  admin: 'Administrador',
}

export const ROLE_HINT: Record<Role, string> = {
  editor: 'Cadastra e edita remédios, embalagens e bulas.',
  reviewer: 'Tudo do editor + marca bulas como revisadas.',
  admin: 'Tudo do revisor + gerencia usuários.',
}

/** Logged-in user (GET /auth/me) */
export type User = { id: number; email: string; name: string; role: Role }

/** Row of the user list (GET /users) */
export type UserRow = User & { active: boolean; created_at: string | null; updated_at: string | null }

export type UserCreate = { name: string; email: string; password: string; role: Role }
export type UserUpdate = { name: string; email: string; role: Role; active: boolean }

// ---------- spreadsheet import ----------
export type ImportCounts = { created: number; updated: number }

export type ImportRowError = { sheet: string; line: number; column: string; message: string }

export type ImportResult = {
  /** false on a preview and whenever something failed: nothing was saved */
  applied: boolean
  drugs: ImportCounts
  packages: ImportCounts
  eans: ImportCounts
  summaries: ImportCounts
  interactions: ImportCounts
  errors: ImportRowError[]
}

// ---------- interactions between the active ingredients of the scanned boxes ----------
export type Severity = 'grave' | 'moderada' | 'leve'

export const SEVERITIES: Severity[] = ['grave', 'moderada', 'leve']

export type InteractionDrug = {
  ean: string
  drug_id: number
  brand_name: string
  manufacturer: string
  active_ingredients: string[]
}

/** One of the two drugs a finding came from */
export type DrugRef = { drug_id: number; ean: string; brand_name: string }

/** One pair of active ingredients, from two different boxes, that has a rule */
export type InteractionFinding = {
  ingredient_a: string
  ingredient_b: string
  severity: Severity
  description: string
  recommendation: string
  source_url: string
  drug_a: DrugRef
  drug_b: DrugRef
}

export type InteractionReport = {
  drugs: InteractionDrug[]
  /** EANs que não estão cadastrados */
  not_found: string[]
  findings: InteractionFinding[]
  /** severity of the worst finding, or '' when nothing was found */
  worst_severity: Severity | ''
}

/** One registered rule, as the panel lists it */
export type InteractionRule = {
  ingredient_a_id: number
  ingredient_a: string
  ingredient_b_id: number
  ingredient_b: string
  severity: Severity
  description: string
  recommendation: string | null
  source_url: string
  updated_at: string | null
}

export type InteractionRuleInput = {
  ingredient_a: string
  ingredient_b: string
  severity: Severity
  description: string
  recommendation: string
  source_url: string
}
