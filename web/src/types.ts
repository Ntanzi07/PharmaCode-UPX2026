export type Page<T> = { data: T[]; limit: number; offset: number }

export type Drug = {
  id: number
  registration_number: string
  brand_name: string | null
  active_ingredient: string
  manufacturer: string
  updated_at: string | null
}

export type DrugInput = {
  registration_number: string
  brand_name: string
  active_ingredient: string
  manufacturer: string
}

export type Package = {
  id: number
  drug_id: number
  description: string
  /** Registro da apresentação na Anvisa (13 dígitos), ponte com a CMED */
  presentation_registration: string | null
  /** Uma apresentação pode ter mais de um código de barras na prateleira */
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
  active_ingredient: string
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

/** Versão da bula oficial que foi resumida (expediente e data de publicação YYYY-MM-DD) */
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
  active_ingredient: string
  manufacturer: string
  source_url: string | null
  leaflet_expedient: string | null
  leaflet_published_at: string | null
} & { [K in SummaryTextKey]: string | null }

// ---------- usuários e login ----------
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

/** Usuário logado (GET /auth/me) */
export type User = { id: number; email: string; name: string; role: Role }

/** Linha da lista de usuários (GET /users) */
export type UserRow = User & { active: boolean; created_at: string | null; updated_at: string | null }

export type UserCreate = { name: string; email: string; password: string; role: Role }
export type UserUpdate = { name: string; email: string; role: Role; active: boolean }
