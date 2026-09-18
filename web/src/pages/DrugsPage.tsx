import { useState, type FormEvent } from 'react'
import { api, errorMessage } from '../api'
import type { Drug, DrugInput } from '../types'
import Modal from '../components/Modal'
import Pager from '../components/Pager'
import Field from '../components/Field'
import { usePaged, PAGE_SIZE } from '../components/usePaged'
import { useToast } from '../components/Toast'
import { fmtDate } from '../components/format'

const EMPTY: DrugInput = { registration_number: '', brand_name: '', active_ingredient: '', manufacturer: '' }

export default function DrugsPage() {
  const { rows, loading, error, offset, setOffset, reload } = usePaged(api.drugs.list)
  const [editing, setEditing] = useState<Drug | 'new' | null>(null)
  const notify = useToast()

  const remove = async (d: Drug) => {
    if (!confirm(`Remover "${d.brand_name || d.active_ingredient}"?`)) return
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
                <td>{d.active_ingredient}</td>
                <td>{d.manufacturer}</td>
                <td className="muted">{fmtDate(d.updated_at)}</td>
                <td className="actions">
                  <button onClick={() => setEditing(d)}>Editar</button>
                  <button className="danger" onClick={() => remove(d)}>Remover</button>
                </td>
              </tr>
            ))}
            {!loading && rows.length === 0 && (
              <tr><td colSpan={7} className="empty">Nenhum remédio cadastrado.</td></tr>
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
          onSaved={() => { setEditing(null); reload() }}
        />
      )}
    </section>
  )
}

function DrugForm({ drug, onClose, onSaved }: { drug: Drug | null; onClose: () => void; onSaved: () => void }) {
  const [form, setForm] = useState<DrugInput>(
    drug
      ? {
          registration_number: drug.registration_number,
          brand_name: drug.brand_name ?? '',
          active_ingredient: drug.active_ingredient,
          manufacturer: drug.manufacturer,
        }
      : EMPTY,
  )
  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const notify = useToast()

  const set = (k: keyof DrugInput) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm({ ...form, [k]: e.target.value })

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setErr(null)
    try {
      if (drug) await api.drugs.update(drug.id, form)
      else await api.drugs.create(form)
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
        <Field label="Princípio ativo" required>
          <input value={form.active_ingredient} onChange={set('active_ingredient')} required placeholder="Ex.: Paracetamol" />
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
