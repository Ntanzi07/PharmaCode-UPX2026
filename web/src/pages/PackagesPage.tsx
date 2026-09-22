import { useMemo, useState, type FormEvent } from 'react'
import { api, errorMessage } from '../api'
import type { Drug, Package } from '../types'
import Modal from '../components/Modal'
import Pager from '../components/Pager'
import Field from '../components/Field'
import { usePaged, PAGE_SIZE } from '../components/usePaged'
import { useToast } from '../components/Toast'
import { fmtDate } from '../components/format'
import { useDrugs } from '../components/drugs'
import DrugPicker from '../components/DrugPicker'
import EanListInput from '../components/EanListInput'

export default function PackagesPage() {
  const { rows, loading, error, offset, setOffset, reload } = usePaged(api.packages.list)
  const [editing, setEditing] = useState<Package | 'new' | null>(null)
  const drugs = useDrugs()
  const byId = useMemo(() => new Map(drugs.map((d) => [d.id, d])), [drugs])
  const notify = useToast()

  const remove = async (p: Package) => {
    if (!confirm(`Remover a embalagem "${p.description}" e seus ${p.eans.length} EAN(s)?`)) return
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
            <tr><th>ID</th><th>EANs</th><th>Descrição</th><th>Reg. apresentação</th><th>Remédio</th><th>Atualizado</th><th /></tr>
          </thead>
          <tbody>
            {rows.map((p) => {
              const d = byId.get(p.drug_id)
              return (
                <tr key={p.id}>
                  <td className="muted">{p.id}</td>
                  <td>
                    <div className="chips">
                      {p.eans.map((e) => <span key={e} className="chip mono">{e}</span>)}
                      {p.eans.length === 0 && <span className="muted">—</span>}
                    </div>
                  </td>
                  <td>{p.description}</td>
                  <td className="mono">{p.presentation_registration || <span className="muted">—</span>}</td>
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
              <tr><td colSpan={7} className="empty">Nenhuma embalagem cadastrada.</td></tr>
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
  const [eans, setEans] = useState<string[]>(pkg?.eans.length ? pkg.eans : [''])
  const [description, setDescription] = useState(pkg?.description ?? '')
  const [presentation, setPresentation] = useState(pkg?.presentation_registration ?? '')
  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const notify = useToast()

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (drugId === '') return setErr('Selecione o remédio')
    const cleanEans = [...new Set(eans.map((x) => x.trim()).filter(Boolean))]
    if (cleanEans.length === 0) return setErr('Informe pelo menos um EAN')
    const bad = cleanEans.find((x) => x.length < 8 || x.length > 13)
    if (bad) return setErr(`EAN ${bad} inválido: precisa ter de 8 a 13 dígitos`)
    if (presentation && presentation.length !== 13) return setErr('O registro da apresentação tem 13 dígitos')
    const fields = { eans: cleanEans, description, presentation_registration: presentation }
    setSaving(true)
    setErr(null)
    try {
      if (pkg) {
        await api.packages.update(pkg.id, { ...fields, drug_id: drugId })
      } else {
        // POST /packages takes the registration number, not the drug ID
        const drug = drugs.find((d) => d.id === drugId)!
        await api.packages.create({ ...fields, registration_number: drug.registration_number })
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
          <DrugPicker drugs={drugs} value={drugId} onChange={setDrugId} />
        </Field>
        <Field label="Descrição da embalagem" required>
          <input value={description} onChange={(e) => setDescription(e.target.value)} required placeholder="Ex.: 500 mg, caixa com 20 comprimidos" />
        </Field>
        <Field
          label="Códigos EAN (código de barras)"
          required
          hint="A mesma caixa pode circular com mais de um código (ex.: código antigo e novo). Cadastre todos."
        >
          <EanListInput value={eans} onChange={setEans} />
        </Field>
        <Field label="Registro da apresentação (Anvisa)" hint="13 dígitos. É o código que liga a embalagem à tabela de preços da CMED.">
          <input
            value={presentation}
            onChange={(e) => setPresentation(e.target.value.replace(/\D/g, ''))}
            inputMode="numeric"
            maxLength={13}
            placeholder="1234567890014"
          />
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
