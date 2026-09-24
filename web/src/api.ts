import type {
  Drug, DrugInput, EanSummary, Package, PackageCreate, PackageUpdate,
  ImportResult, Page, Summary, SummaryInput, SummaryListItem, User, UserCreate, UserRow, UserUpdate,
} from './types'

const BASE = '/api'

/** Event fired when the API answers 401 while the panel is in use */
export const SESSION_EXPIRED = 'pharmacode:session-expired'

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
    // The API returns errors as plain text (http.Error)
    const text = (await res.text()).trim()
    // Session expired or was ended (e.g. an admin deactivated the user): tell the app
    // to go back to the login screen. Login and /auth/me handle their own 401.
    if (res.status === 401 && !path.startsWith('/auth/')) {
      window.dispatchEvent(new Event(SESSION_EXPIRED))
    }
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
    // the reviewer is the logged-in user; the API takes it from the session
    review: (id: number) => request<void>('PATCH', `/summaries/${id}/review`),
    remove: (id: number) => request<void>('DELETE', `/summaries/${id}`),
  },
  auth: {
    login: (email: string, password: string) => request<User>('POST', '/auth/login', { email, password }),
    logout: () => request<void>('POST', '/auth/logout'),
    me: () => request<User>('GET', '/auth/me'),
    changePassword: (current_password: string, new_password: string) =>
      request<void>('PUT', '/auth/password', { current_password, new_password }),
  },
  imports: {
    /** direct link: the browser downloads it with the session cookie */
    templateUrl: BASE + '/imports/template',
    preview: (file: File) => uploadSpreadsheet('/imports/preview', file),
    apply: (file: File) => uploadSpreadsheet('/imports/apply', file),
  },
  users: {
    list: () => request<UserRow[]>('GET', '/users'),
    create: (u: UserCreate) => request<{ id: number }>('POST', '/users', u),
    update: (id: number, u: UserUpdate) => request<void>('PUT', `/users/${id}`, u),
    setPassword: (id: number, password: string) => request<void>('PUT', `/users/${id}/password`, { password }),
  },
}

/**
 * Sends the spreadsheet as multipart/form-data. A 422 is not an exception here:
 * it is the API saying the file has errors, and the body is the same result
 * object, with the list of what to fix.
 */
async function uploadSpreadsheet(path: string, file: File): Promise<ImportResult> {
  const body = new FormData()
  body.append('file', file)
  const res = await fetch(BASE + path, { method: 'POST', body })
  if (res.ok || res.status === 422) return (await res.json()) as ImportResult
  const text = (await res.text()).trim()
  if (res.status === 401) window.dispatchEvent(new Event(SESSION_EXPIRED))
  throw new ApiError(res.status, text || `Erro ${res.status}`)
}

/** Fetches every drug (the API caps pages at 100) to fill pickers. */
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
