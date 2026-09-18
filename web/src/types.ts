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
  ean: string
  description: string
  updated_at: string | null
}

export type PackageCreate = { registration_number: string; ean: string; description: string }
export type PackageUpdate = { drug_id: number; ean: string; description: string }

export type SummaryListItem = {
  id: number
  drug_id: number
  brand_name: string | null
  active_ingredient: string
  what_is_it_for: string
  source_url: string
  reviewed_by: string | null
  reviewed_at: string | null
  updated_at: string | null
}

export const SUMMARY_TEXT_FIELDS = [
  { key: 'what_is_it_for', label: 'Para que serve', required: true },
  { key: 'posology', label: 'Posologia', required: true },
  { key: 'adverse_effects', label: 'Reações adversas', required: false },
  { key: 'drug_interactions', label: 'Interações medicamentosas', required: false },
  { key: 'contraindications', label: 'Contraindicações', required: false },
  { key: 'side_effects', label: 'Efeitos colaterais', required: false },
  { key: 'when_to_seek_help', label: 'Quando procurar ajuda', required: false },
  { key: 'mechanism_of_action', label: 'Mecanismo de ação', required: false },
  { key: 'storage', label: 'Armazenamento', required: false },
] as const

export type SummaryTextKey = (typeof SUMMARY_TEXT_FIELDS)[number]['key']

export type SummaryInput = Record<SummaryTextKey, string> & { source_url: string }

export type Summary = { id: number; drug_id: number; source_url: string } & {
  [K in SummaryTextKey]: string | null
} & { reviewed_by: string | null; reviewed_at: string | null; updated_at: string | null }

export type EanSummary = {
  ean: string
  description: string
  registration_number: string
  brand_name: string | null
  active_ingredient: string
  manufacturer: string
  source_url?: string | null
} & { [K in SummaryTextKey]: string | null }
