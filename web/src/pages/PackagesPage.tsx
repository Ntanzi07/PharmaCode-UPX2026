import { useMemo, useState, type FormEvent } from 'react'
import { api, errorMessage } from '../api'
import type { Drug, Package } from '../types'
import Modal from '../components/Modal'
import Pager from '../components/Pager'
import Field from '../components/Field'
import { usePaged, PAGE_SIZE } from '../components/usePaged'
import { useToast } from '../components/Toast'
import { fmtDate } from '../components/format'
import { drugLabel, useDrugs } from '../components/drugs'

export default function PackagesPage() {
  const { rows, loading, error, offset, setOffset, reload } = usePaged(api.packages.list)
  const [editing, setEditing] = useState<Package | 'new' | null>(null)
  const drugs = useDrugs()
  const byId = useMemo(() => new Map(drugs.map((d) => [d.id, d])), [drugs])
  const notify = useToast()

  const remove = async (p: Package) => {
    if (!confirm(`Remover a embalagem EAN ${p.ean}?`)) return
    try {
      await api.packages.remove(p.id)
      notify('ok', 'Embalagem removida')
      reload()
    } catch (e) {
      notify('error', errorMessage(e))
    }
  }

  return (
    <section>
      <div className="section-head">
        <h1>Embalagens</h1>
        <button className="primary" onClick={() => setEditing('new')}>+ Nova embalagem</button>
      </div>

      {error && <div className="alert">{error}</div>}

      <div className="card">
        <table>
          <thead>
            <tr><th>ID</th><th>EAN</th><th>Descrição</th><th>Remédio</th><th>Atualizado</th><th /></tr>
          </thead>
          <tbody>
            {rows.map((p) => {
              const d = byId.get(p.drug_id)
              return (
                <tr key={p.id}>
                  <td className="muted">{p.id}</td>
                  <td className="mono">{p.ean}</td>
                  <td>{p.description}</td>
                  <td>{d ? d.brand_name || d.active_ingredient : <span className="muted">#{p.drug_id}</span>}</td>
                  <td className="muted">{fmtDate(p.updated_at)}</td>
                  <td className="actions">
                    <button onClick={() => setEditing(p)}>Editar</button>
                    <button className="danger" onClick={() => remove(p)}>Remover</button>
                  </td>
                </tr>
              )
            })}
            {!loading && rows.length === 0 && (
              <tr><td colSpan={6} className="empty">Nenhuma embalagem cadastrada.</td></tr>
            )}
          </tbody>
        </table>
        {loading && <div className="loading">Carregando…</div>}
      </div>
      <Pager offset={offset} limit={PAGE_SIZE} count={rows.length} onChange={setOffset} />

      {editing && (
        <PackageForm
          pkg={editing === 'new' ? null : editing}
          drugs={drugs}
          onClose={() => setEditing(null)}
          onSaved={() => { setEditing(null); reload() }}
        />
      )}
    </section>
  )
}

type FormProps = { pkg: Package | null; drugs: Drug[]; onClose: () => void; onSaved: () => void }

function PackageForm({ pkg, drugs, onClose, onSaved }: FormProps) {
  const [drugId, setDrugId] = useState<number | ''>(pkg?.drug_id ?? '')
  const [ean, setEan] = useState(pkg?.ean ?? '')
  const [description, setDescription] = useState(pkg?.description ?? '')
  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const notify = useToast()

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (drugId === '') return setErr('Selecione o remédio')
    setSaving(true)
    setErr(null)
    try {
      if (pkg) {
        await api.packages.update(pkg.id, { drug_id: drugId, ean, description })
      } else {
        // O POST /packages recebe o número de registro, não o ID do remédio
        const drug = drugs.find((d) => d.id === drugId)!
        await api.packages.create({ registration_number: drug.registration_number, ean, description })
      }
      notify('ok', pkg ? 'Embalagem atualizada' : 'Embalagem cadastrada')
      onSaved()
    } catch (e) {
      setErr(errorMessage(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title={pkg ? 'Editar embalagem' : 'Nova embalagem'} onClose={onClose}>
      <form onSubmit={submit} className="form">
        <Field label="Remédio" required>
          <select value={drugId} onChange={(e) => setDrugId(e.target.value ? Number(e.target.value) : '')} required>
            <option value="">Selecione…</option>
            {drugs.map((d) => <option key={d.id} value={d.id}>{drugLabel(d)}</option>)}
          </select>
        </Field>
        <Field label="Código EAN (código de barras)" required>
          <input value={ean} onChange={(e) => setEan(e.target.value.trim())} required inputMode="numeric" maxLength={13} placeholder="7891234567890" />
        </Field>
        <Field label="Descrição da embalagem" required>
          <input value={description} onChange={(e) => setDescription(e.target.value)} required placeholder="Ex.: 500 mg, caixa com 20 comprimidos" />
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
