import type {
  Drug, DrugInput, EanSummary, Package, PackageCreate, PackageUpdate,
  Page, Summary, SummaryInput, SummaryListItem,
} from './types'

const BASE = '/api'

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message)
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(BASE + path, {
    method,
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  if (!res.ok) {
    // A API responde erros em texto puro (http.Error)
    const text = (await res.text()).trim()
    throw new ApiError(res.status, text || `Erro ${res.status}`)
  }
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

const page = (limit: number, offset: number) => `?limit=${limit}&offset=${offset}`

export const api = {
  drugs: {
    list: (limit = 20, offset = 0) => request<Page<Drug>>('GET', '/drugs' + page(limit, offset)),
    create: (d: DrugInput) => request<{ id: number }>('POST', '/drugs', d),
    update: (id: number, d: DrugInput) => request<void>('PUT', `/drugs/${id}`, d),
    remove: (id: number) => request<void>('DELETE', `/drugs/${id}`),
    byEan: (ean: string) => request<EanSummary>('GET', `/drugs/ean/${encodeURIComponent(ean)}`),
  },
  packages: {
    list: (limit = 20, offset = 0) => request<Page<Package>>('GET', '/packages' + page(limit, offset)),
    create: (p: PackageCreate) => request<{ id: number }>('POST', '/packages', p),
    update: (id: number, p: PackageUpdate) => request<void>('PUT', `/packages/${id}`, p),
    remove: (id: number) => request<void>('DELETE', `/packages/${id}`),
  },
  summaries: {
    list: (limit = 20, offset = 0) =>
      request<Page<SummaryListItem>>('GET', '/summaries' + page(limit, offset)),
    get: (id: number) => request<Summary>('GET', `/summaries/${id}`),
    create: (s: SummaryInput & { drug_id: number }) => request<{ id: number }>('POST', '/summaries', s),
    update: (id: number, s: SummaryInput) => request<void>('PUT', `/summaries/${id}`, s),
    review: (id: number, reviewed_by: string) =>
      request<void>('PATCH', `/summaries/${id}/review`, { reviewed_by }),
    remove: (id: number) => request<void>('DELETE', `/summaries/${id}`),
  },
}

/** Busca todos os remédios (a API limita 100 por página) para preencher selects. */
export async function fetchAllDrugs(): Promise<Drug[]> {
  const all: Drug[] = []
  for (let offset = 0; ; offset += 100) {
    const { data } = await api.drugs.list(100, offset)
    all.push(...data)
    if (data.length < 100) return all
  }
}

export function errorMessage(e: unknown): string {
  if (e instanceof ApiError) return e.message
  if (e instanceof Error) return e.message
  return String(e)
}
