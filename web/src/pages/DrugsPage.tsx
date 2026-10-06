import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { api, errorMessage } from '../api'
import type { Drug, DrugInput } from '../types'
import Modal from '../components/Modal'
import Pager from '../components/Pager'
import Field from '../components/Field'
import ListInput from '../components/ListInput'
import { usePaged, PAGE_SIZE } from '../components/usePaged'
import { useToast } from '../components/Toast'
import { fmtDate } from '../components/format'

const EMPTY: DrugInput = { registration_number: '', brand_name: '', active_ingredients: [''], manufacturer: '' }

export default function DrugsPage() {
  // The search is typed here and runs in the database: with the Anvisa base
  // loaded there are tens of thousands of drugs, and paging 20 at a time to
  // find one is not a way to work.
  const [search, setSearch] = useState('')
  const [query, setQuery] = useState('')
  const list = useCallback(
    (limit: number, offset: number) => api.drugs.list(limit, offset, query),
    [query],
  )
  const { rows, loading, error, offset, setOffset, reload } = usePaged(list)
  const [editing, setEditing] = useState<Drug | 'new' | null>(null)
  const notify = useToast()

  // Wait for a pause in the typing before asking the API.
  useEffect(() => {
    const timer = setTimeout(() => {
      setQuery(search.trim())
      setOffset(0)
    }, 250)
    return () => clearTimeout(timer)
  }, [search, setOffset])

  const remove = async (d: Drug) => {
    if (!confirm(`Remover "${d.brand_name || d.active_ingredients.join(" + ")}"?`)) return
    try {
      await api.drugs.remove(d.id)
      notify('ok', 'Remédio removido')
      reload()
    } catch (e) {
      notify('error', errorMessage(e))
    }
  }

  return (
    <section>
      <div className="section-head">
        <h1>Remédios</h1>
        <button className="primary" onClick={() => setEditing('new')}>+ Novo remédio</button>
      </div>

      <input
        className="search"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        placeholder="Buscar por nome, princípio ativo, fabricante ou registro…"
        aria-label="Buscar remédio"
      />

      {error && <div className="alert">{error}</div>}

      <div className="card">
        <table>
          <thead>
            <tr>
              <th>ID</th><th>Registro Anvisa</th><th>Nome comercial</th><th>Princípio ativo</th>
              <th>Fabricante</th><th>Atualizado</th><th />
            </tr>
          </thead>
          <tbody>
            {rows.map((d) => (
              <tr key={d.id}>
                <td className="muted">{d.id}</td>
                <td className="mono">{d.registration_number}</td>
                <td>{d.brand_name || <span className="muted">—</span>}</td>
                <td>
                  <div className="chips">
                    {d.active_ingredients.map((i) => <span key={i} className="chip">{i}</span>)}
                    {d.active_ingredients.length === 0 && <span className="muted">—</span>}
                  </div>
                </td>
                <td>{d.manufacturer}</td>
                <td className="muted">{fmtDate(d.updated_at)}</td>
                <td className="actions">
                  <button onClick={() => setEditing(d)}>Editar</button>
                  <button className="danger" onClick={() => remove(d)}>Remover</button>
                </td>
              </tr>
            ))}
            {!loading && rows.length === 0 && (
              <tr>
                <td colSpan={7} className="empty">
                  {query ? `Nenhum remédio encontrado para "${query}".` : 'Nenhum remédio cadastrado.'}
                </td>
              </tr>
            )}
          </tbody>
        </table>
        {loading && <div className="loading">Carregando…</div>}
      </div>
      <Pager offset={offset} limit={PAGE_SIZE} count={rows.length} onChange={setOffset} />

      {editing && (
        <DrugForm
          drug={editing === 'new' ? null : editing}
          onClose={() => setEditing(null)}
          onEdit={setEditing}
          onSaved={() => { setEditing(null); reload() }}
        />
      )}
    </section>
  )
}

type FormProps = {
  drug: Drug | null
  onClose: () => void
  /** opens an existing drug for editing, when the one being typed is already there */
  onEdit: (drug: Drug) => void
  onSaved: () => void
}

function DrugForm({ drug, onClose, onEdit, onSaved }: FormProps) {
  const [form, setForm] = useState<DrugInput>(
    drug
      ? {
          registration_number: drug.registration_number,
          brand_name: drug.brand_name ?? '',
          active_ingredients: drug.active_ingredients.length ? drug.active_ingredients : [''],
          manufacturer: drug.manufacturer,
        }
      : EMPTY,
  )
  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const notify = useToast()
  // Drugs already registered that look like the one being typed. Without this,
  // typing a name that exists just created a second row, and a repeated
  // registration number only blew up at the database.
  const [matches, setMatches] = useState<Drug[]>([])

  const term = (form.registration_number.trim() || form.brand_name.trim()).toLowerCase()
  useEffect(() => {
    // Only when creating: editing a drug is supposed to match itself.
    if (drug || term.length < 3) {
      setMatches([])
      return
    }
    let alive = true
    const timer = setTimeout(() => {
      api.drugs
        .list(5, 0, term)
        .then(({ data }) => alive && setMatches(data))
        .catch(() => alive && setMatches([]))
    }, 300)
    return () => {
      alive = false
      clearTimeout(timer)
    }
  }, [term, drug])

  // The registration number is unique in the database: a repeat is a conflict,
  // not a warning. The same name is not — seven different warfarin drugs are
  // called Marevan by different companies.
  const sameRegistration = matches.find(
    (m) => m.registration_number === form.registration_number.trim(),
  )

  const set = (k: keyof DrugInput) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm({ ...form, [k]: e.target.value })

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    const active_ingredients = [...new Set(form.active_ingredients.map((i) => i.trim()).filter(Boolean))]
    if (active_ingredients.length === 0) return setErr('Informe pelo menos um princípio ativo')
    if (sameRegistration) {
      return setErr(`O registro ${sameRegistration.registration_number} já está cadastrado. Abra o remédio existente para editar.`)
    }
    const payload = { ...form, active_ingredients }
    setSaving(true)
    setErr(null)
    try {
      if (drug) await api.drugs.update(drug.id, payload)
      else await api.drugs.create(payload)
      notify('ok', drug ? 'Remédio atualizado' : 'Remédio cadastrado')
      onSaved()
    } catch (e) {
      setErr(errorMessage(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title={drug ? 'Editar remédio' : 'Novo remédio'} onClose={onClose}>
      <form onSubmit={submit} className="form">
        <Field label="Número de registro (Anvisa)" required>
          <input value={form.registration_number} onChange={set('registration_number')} required autoFocus />
        </Field>
        <Field label="Nome comercial">
          <input value={form.brand_name} onChange={set('brand_name')} placeholder="Ex.: Tylenol" />
        </Field>

        {matches.length > 0 && (
          <div className={sameRegistration ? 'alert' : 'alert info'}>
            <strong>
              {sameRegistration
                ? 'Esse registro já está cadastrado:'
                : 'Já existe remédio parecido cadastrado:'}
            </strong>
            <ul className="matches">
              {matches.map((m) => (
                <li key={m.id}>
                  <span>
                    {m.brand_name || m.active_ingredients.join(' + ')}
                    <span className="muted"> — {m.manufacturer} · Reg. {m.registration_number}</span>
                  </span>
                  <button type="button" onClick={() => onEdit(m)}>Editar esse</button>
                </li>
              ))}
            </ul>
            {!sameRegistration && (
              <small>
                Nomes iguais de fabricantes diferentes são remédios diferentes: se for esse o caso, pode cadastrar.
              </small>
            )}
          </div>
        )}
        <Field
          label="Princípios ativos"
          required
          hint="Um por campo. Associações como a Neosaldina têm vários, e é por eles que a API checa as interações entre as caixas."
        >
          <ListInput
            value={form.active_ingredients}
            onChange={(active_ingredients) => setForm({ ...form, active_ingredients })}
            placeholder="Ex.: dipirona sódica"
            addLabel="+ adicionar outro princípio ativo"
            label="princípio ativo"
          />
        </Field>
        <Field label="Fabricante" required>
          <input value={form.manufacturer} onChange={set('manufacturer')} required />
        </Field>
        {err && <div className="alert">{err}</div>}
        <div className="form-actions">
          <button type="button" onClick={onClose}>Cancelar</button>
          <button className="primary" disabled={saving}>{saving ? 'Salvando…' : 'Salvar'}</button>
        </div>
      </form>
    </Modal>
  )
}
