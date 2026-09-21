import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { api, errorMessage } from '../api'
import { SUMMARY_TEXT_FIELDS, type SummaryInput, type SummaryListItem } from '../types'
import Modal from '../components/Modal'
import Pager from '../components/Pager'
import Field from '../components/Field'
import { usePaged, PAGE_SIZE } from '../components/usePaged'
import { useToast } from '../components/Toast'
import { fmtDate, fmtDay } from '../components/format'
import { drugLabel, useDrugs } from '../components/drugs'
import DrugPicker from '../components/DrugPicker'

const EMPTY: SummaryInput = {
  what_is_it_for: '', posology: '', missed_dose: '', warnings: '', adverse_effects: '', drug_interactions: '',
  contraindications: '', side_effects: '', when_to_seek_help: '', mechanism_of_action: '', storage: '',
  source_url: '', leaflet_expedient: '', leaflet_published_at: '',
}

type Editing = { mode: 'new' } | { mode: 'edit'; id: number } | { mode: 'review'; item: SummaryListItem }

export default function SummariesPage() {
  const { rows, loading, error, offset, setOffset, reload } = usePaged(api.summaries.list)
  const [editing, setEditing] = useState<Editing | null>(null)
  const notify = useToast()

  const remove = async (s: SummaryListItem) => {
    if (!confirm(`Remover a bula de "${s.brand_name || s.active_ingredient}"?`)) return
    try {
      await api.summaries.remove(s.id)
      notify('ok', 'Bula removida')
      reload()
    } catch (e) {
      notify('error', errorMessage(e))
    }
  }

  const done = () => { setEditing(null); reload() }

  return (
    <section>
      <div className="section-head">
        <h1>Bulas simplificadas</h1>
        <button className="primary" onClick={() => setEditing({ mode: 'new' })}>+ Nova bula</button>
      </div>

      {error && <div className="alert">{error}</div>}

      <div className="card">
        <table>
          <thead>
            <tr><th>ID</th><th>Remédio</th><th>Para que serve</th><th>Versão da bula</th><th>Revisão</th><th>Atualizado</th><th /></tr>
          </thead>
          <tbody>
            {rows.map((s) => (
              <tr key={s.id}>
                <td className="muted">{s.id}</td>
                <td>
                  <strong>{s.brand_name || s.active_ingredient}</strong>
                  <div className="muted small">{s.active_ingredient}</div>
                </td>
                <td className="purpose"><div className="clamp" title={s.what_is_it_for}>{s.what_is_it_for}</div></td>
                <td>
                  {s.leaflet_expedient || s.leaflet_published_at ? (
                    <>
                      <div className="mono small">{s.leaflet_expedient || '—'}</div>
                      <div className="muted small">{fmtDay(s.leaflet_published_at)}</div>
                    </>
                  ) : (
                    <span className="badge warn" title="Sem expediente nem data: não dá para saber se o resumo está atualizado">
                      Sem versão
                    </span>
                  )}
                </td>
                <td>
                  {s.reviewed_by
                    ? <span className="badge ok" title={fmtDate(s.reviewed_at)}>✓ {s.reviewed_by}</span>
                    : <span className="badge warn">Pendente</span>}
                </td>
                <td className="muted">{fmtDate(s.updated_at)}</td>
                <td className="actions">
                  <button onClick={() => setEditing({ mode: 'edit', id: s.id })}>Editar</button>
                  <button onClick={() => setEditing({ mode: 'review', item: s })}>Revisar</button>
                  <button className="danger" onClick={() => remove(s)}>Remover</button>
                </td>
              </tr>
            ))}
            {!loading && rows.length === 0 && (
              <tr><td colSpan={7} className="empty">Nenhuma bula cadastrada.</td></tr>
            )}
          </tbody>
        </table>
        {loading && <div className="loading">Carregando…</div>}
      </div>
      <Pager offset={offset} limit={PAGE_SIZE} count={rows.length} onChange={setOffset} />

      {editing?.mode === 'new' && <SummaryForm onClose={() => setEditing(null)} onSaved={done} />}
      {editing?.mode === 'edit' && <SummaryForm id={editing.id} onClose={() => setEditing(null)} onSaved={done} />}
      {editing?.mode === 'review' && <ReviewForm item={editing.item} onClose={() => setEditing(null)} onSaved={done} />}
    </section>
  )
}

function SummaryForm({ id, onClose, onSaved }: { id?: number; onClose: () => void; onSaved: () => void }) {
  const drugs = useDrugs()
  const [drugId, setDrugId] = useState<number | ''>('')
  const [form, setForm] = useState<SummaryInput>(EMPTY)
  const [loading, setLoading] = useState(id !== undefined)
  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const notify = useToast()
  const drug = useMemo(() => drugs.find((d) => d.id === drugId), [drugs, drugId])

  useEffect(() => {
    if (id === undefined) return
    api.summaries
      .get(id)
      .then((s) => {
        setDrugId(s.drug_id)
        const f = {
          ...EMPTY,
          source_url: s.source_url,
          leaflet_expedient: s.leaflet_expedient ?? '',
          leaflet_published_at: s.leaflet_published_at ?? '',
        }
        for (const { key } of SUMMARY_TEXT_FIELDS) f[key] = s[key] ?? ''
        setForm(f)
      })
      .catch((e) => setErr(errorMessage(e)))
      .finally(() => setLoading(false))
  }, [id])

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (drugId === '') return setErr('Selecione o remédio')
    setSaving(true)
    setErr(null)
    try {
      if (id !== undefined) await api.summaries.update(id, form)
      else await api.summaries.create({ ...form, drug_id: drugId })
      notify('ok', id !== undefined ? 'Bula atualizada' : 'Bula cadastrada')
      onSaved()
    } catch (e) {
      setErr(errorMessage(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title={id !== undefined ? 'Editar bula' : 'Nova bula'} onClose={onClose} wide>
      {loading ? (
        <div className="loading">Carregando…</div>
      ) : (
        <form onSubmit={submit} className="form">
          <Field label="Remédio" required hint={id !== undefined ? 'O remédio de uma bula não pode ser trocado.' : undefined}>
            {id !== undefined ? (
              <input value={drug ? drugLabel(drug) : `#${drugId}`} disabled />
            ) : (
              <DrugPicker drugs={drugs} value={drugId} onChange={setDrugId} />
            )}
          </Field>
          <div className="grid2">
            {SUMMARY_TEXT_FIELDS.map((f) => (
              <Field key={f.key} label={f.label} required={f.required}>
                <textarea
                  rows={f.required ? 4 : 3}
                  value={form[f.key]}
                  required={f.required}
                  onChange={(e) => setForm({ ...form, [f.key]: e.target.value })}
                />
              </Field>
            ))}
          </div>
          <Field label="Fonte (URL da bula oficial)" required>
            <input
              type="url"
              value={form.source_url}
              onChange={(e) => setForm({ ...form, source_url: e.target.value })}
              required
              placeholder="https://consultas.anvisa.gov.br/..."
            />
          </Field>
          <fieldset className="group">
            <legend>Versão da bula oficial resumida</legend>
            <p className="hint">Serve para saber quando este resumo ficou desatualizado em relação à bula publicada.</p>
            <div className="grid2">
              <Field label="Número do expediente">
                <input
                  value={form.leaflet_expedient}
                  onChange={(e) => setForm({ ...form, leaflet_expedient: e.target.value })}
                  maxLength={30}
                  placeholder="Ex.: 0123456/24-5"
                />
              </Field>
              <Field label="Data de publicação da bula">
                <input
                  type="date"
                  value={form.leaflet_published_at}
                  onChange={(e) => setForm({ ...form, leaflet_published_at: e.target.value })}
                />
              </Field>
            </div>
          </fieldset>
          {err && <div className="alert">{err}</div>}
          <div className="form-actions">
            <button type="button" onClick={onClose}>Cancelar</button>
            <button className="primary" disabled={saving}>{saving ? 'Salvando…' : 'Salvar'}</button>
          </div>
        </form>
      )}
    </Modal>
  )
}

function ReviewForm({ item, onClose, onSaved }: { item: SummaryListItem; onClose: () => void; onSaved: () => void }) {
  const [by, setBy] = useState(item.reviewed_by ?? '')
  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const notify = useToast()

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setErr(null)
    try {
      await api.summaries.review(item.id, by.trim())
      notify('ok', 'Bula marcada como revisada')
      onSaved()
    } catch (e) {
      setErr(errorMessage(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title={`Revisar bula de ${item.brand_name || item.active_ingredient}`} onClose={onClose}>
      <form onSubmit={submit} className="form">
        <Field label="Revisado por" required hint="Nome do farmacêutico responsável pela revisão.">
          <input value={by} onChange={(e) => setBy(e.target.value)} required autoFocus />
        </Field>
        {err && <div className="alert">{err}</div>}
        <div className="form-actions">
          <button type="button" onClick={onClose}>Cancelar</button>
          <button className="primary" disabled={saving}>{saving ? 'Salvando…' : 'Marcar como revisada'}</button>
        </div>
      </form>
    </Modal>
  )
}
