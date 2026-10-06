import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { api, errorMessage } from '../api'
import {
  SEVERITIES,
  type InteractionReport, type InteractionRule, type InteractionRuleInput, type Severity,
} from '../types'
import ListInput from '../components/ListInput'
import Modal from '../components/Modal'
import Field from '../components/Field'
import IngredientInput from '../components/IngredientInput'
import { useAuth } from '../components/auth'
import { useToast } from '../components/Toast'

const SEVERITY_LABEL: Record<Severity, string> = {
  grave: 'Grave',
  moderada: 'Moderada',
  leve: 'Leve',
}

/** Panel for the interaction check and for the rules behind it. */
export default function InteractionsPage() {
  const [tab, setTab] = useState<'check' | 'rules'>('check')

  return (
    <section>
      <div className="section-head">
        <h1>Interações</h1>
      </div>

      <div className="subtabs">
        <button className={tab === 'check' ? 'subtab active' : 'subtab'} onClick={() => setTab('check')}>
          Checar caixas
        </button>
        <button className={tab === 'rules' ? 'subtab active' : 'subtab'} onClick={() => setTab('rules')}>
          Regras cadastradas
        </button>
      </div>

      {tab === 'check' ? <CheckPanel /> : <RulesPanel />}
    </section>
  )
}

/** Simulates what the app does when the person scans more than one box. */
function CheckPanel() {
  const [eans, setEans] = useState<string[]>(['', ''])
  const [report, setReport] = useState<InteractionReport | null>(null)
  const [err, setErr] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const check = async (e: FormEvent) => {
    e.preventDefault()
    const clean = [...new Set(eans.map((x) => x.trim()).filter(Boolean))]
    if (clean.length < 2) return setErr('Informe pelo menos 2 códigos de barras diferentes')
    setLoading(true)
    setErr(null)
    setReport(null)
    try {
      setReport(await api.interactions.check(clean))
    } catch (e) {
      setErr(errorMessage(e))
    } finally {
      setLoading(false)
    }
  }

  return (
    <>
      <p className="muted">
        Escaneie ou digite os códigos de barras das caixas que a pessoa pretende tomar. A API compara todos os pares
        de princípios ativos que vêm de caixas diferentes com as regras cadastradas.
      </p>

      <form className="card pad" onSubmit={check}>
        <ListInput
          value={eans}
          onChange={setEans}
          digitsOnly
          maxLength={13}
          placeholder="7891234567890"
          addLabel="+ adicionar outra caixa"
          label="EAN"
        />
        <div className="form-actions">
          <button className="primary" disabled={loading}>{loading ? 'Checando…' : 'Checar'}</button>
        </div>
      </form>

      {err && <div className="alert">{err}</div>}

      {report && (
        <>
          {report.findings.length > 0 ? (
            <div className={`alert sev-${report.worst_severity}`}>
              {report.findings.length === 1
                ? 'Encontramos 1 interação entre as caixas informadas.'
                : `Encontramos ${report.findings.length} interações entre as caixas informadas.`}{' '}
              Pior gravidade: <strong>{SEVERITY_LABEL[report.worst_severity as Severity]}</strong>.
            </div>
          ) : (
            <div className="alert info">
              Nenhuma regra cadastrada para os pares dessas caixas.{' '}
              <strong>Isso não quer dizer que a combinação seja segura</strong> — quer dizer que ainda não
              cadastramos nada sobre ela. Na dúvida, oriente a pessoa a falar com um farmacêutico.
            </div>
          )}

          {report.findings.map((f) => (
            <div className={`card pad finding sev-${f.severity}`} key={`${f.ingredient_a}|${f.ingredient_b}`}>
              <div className="finding-head">
                <h2>{f.ingredient_a} + {f.ingredient_b}</h2>
                <span className={`badge sev-${f.severity}`}>{SEVERITY_LABEL[f.severity]}</span>
              </div>
              <p>{f.description}</p>
              {f.recommendation && <p><strong>O que fazer:</strong> {f.recommendation}</p>}
              <p className="muted small">
                Vem das caixas{' '}
                <strong>{f.drug_a.brand_name || `#${f.drug_a.drug_id}`}</strong>{' '}
                <span className="mono">{f.drug_a.ean}</span> e{' '}
                <strong>{f.drug_b.brand_name || `#${f.drug_b.drug_id}`}</strong>{' '}
                <span className="mono">{f.drug_b.ean}</span>
              </p>
              <a className="small" href={f.source_url} target="_blank" rel="noreferrer">Fonte</a>
            </div>
          ))}

          <div className="card">
            <table>
              <thead>
                <tr><th>EAN</th><th>Remédio</th><th>Princípios ativos</th></tr>
              </thead>
              <tbody>
                {report.drugs.map((d) => (
                  <tr key={d.ean}>
                    <td className="mono">{d.ean}</td>
                    <td>{d.brand_name || <span className="muted">—</span>}</td>
                    <td>
                      <div className="chips">
                        {d.active_ingredients.map((i) => <span key={i} className="chip">{i}</span>)}
                      </div>
                    </td>
                  </tr>
                ))}
                {report.not_found.map((ean) => (
                  <tr key={ean}>
                    <td className="mono">{ean}</td>
                    <td colSpan={2} className="muted">Não cadastrado</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </>
  )
}

/** The knowledge base: one rule per pair of active ingredients. */
function RulesPanel() {
  const { can } = useAuth()
  const mayEdit = can('reviewer')
  const notify = useToast()

  const [rules, setRules] = useState<InteractionRule[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [editing, setEditing] = useState<InteractionRule | 'new' | null>(null)

  const reload = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const { data } = await api.interactions.rules.list()
      setRules(data)
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { reload() }, [reload])

  const remove = async (r: InteractionRule) => {
    if (!confirm(`Remover a regra "${r.ingredient_a} + ${r.ingredient_b}"?`)) return
    try {
      await api.interactions.rules.remove(r.ingredient_a_id, r.ingredient_b_id)
      notify('ok', 'Regra removida')
      reload()
    } catch (e) {
      notify('error', errorMessage(e))
    }
  }

  return (
    <>
      <div className="section-head">
        <p className="muted">
          Cada regra vale para um par de princípios ativos, em qualquer ordem. Também é possível cadastrar em lote
          pela aba <strong>interacoes_ativos</strong> da planilha modelo.
        </p>
        {mayEdit && <button className="primary" onClick={() => setEditing('new')}>+ Nova regra</button>}
      </div>

      {error && <div className="alert">{error}</div>}

      <div className="card">
        <table>
          <thead>
            <tr>
              <th>Princípio A</th><th>Princípio B</th><th>Gravidade</th><th>O que acontece</th>
              <th>Fonte</th>{mayEdit && <th />}
            </tr>
          </thead>
          <tbody>
            {rules.map((r) => (
              <tr key={`${r.ingredient_a_id}-${r.ingredient_b_id}`}>
                <td>{r.ingredient_a}</td>
                <td>{r.ingredient_b}</td>
                <td><span className={`badge sev-${r.severity}`}>{SEVERITY_LABEL[r.severity]}</span></td>
                <td className="purpose"><div className="clamp">{r.description}</div></td>
                <td><a className="small" href={r.source_url} target="_blank" rel="noreferrer">abrir</a></td>
                {mayEdit && (
                  <td className="actions">
                    <button onClick={() => setEditing(r)}>Editar</button>
                    <button className="danger" onClick={() => remove(r)}>Remover</button>
                  </td>
                )}
              </tr>
            ))}
            {!loading && rules.length === 0 && (
              <tr>
                <td colSpan={mayEdit ? 6 : 5} className="empty">
                  Nenhuma regra cadastrada ainda. Sem regras, a checagem sempre volta vazia.
                </td>
              </tr>
            )}
          </tbody>
        </table>
        {loading && <div className="loading">Carregando…</div>}
      </div>

      {editing && (
        <RuleForm
          rule={editing === 'new' ? null : editing}
          onClose={() => setEditing(null)}
          onSaved={() => { setEditing(null); reload() }}
        />
      )}
    </>
  )
}

const EMPTY: InteractionRuleInput = {
  ingredient_a: '', ingredient_b: '', severity: 'moderada',
  description: '', recommendation: '', source_url: '',
}

function RuleForm({ rule, onClose, onSaved }: {
  rule: InteractionRule | null
  onClose: () => void
  onSaved: () => void
}) {
  const [form, setForm] = useState<InteractionRuleInput>(
    rule
      ? {
          ingredient_a: rule.ingredient_a,
          ingredient_b: rule.ingredient_b,
          severity: rule.severity,
          description: rule.description,
          recommendation: rule.recommendation ?? '',
          source_url: rule.source_url,
        }
      : EMPTY,
  )
  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const notify = useToast()

  const set = <K extends keyof InteractionRuleInput>(key: K, value: InteractionRuleInput[K]) =>
    setForm((f) => ({ ...f, [key]: value }))

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setErr(null)
    try {
      await api.interactions.rules.save(form)
      notify('ok', rule ? 'Regra atualizada' : 'Regra cadastrada')
      onSaved()
    } catch (e) {
      setErr(errorMessage(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title={rule ? 'Editar regra' : 'Nova regra'} onClose={onClose} wide>
      <form onSubmit={submit} className="form">
        <div className="grid2">
          <Field label="Princípio ativo A" required>
            {/* Editing is editing the rule of that pair: changing a name here
                would create a different rule instead of renaming this one. */}
            <IngredientInput
              value={form.ingredient_a}
              onChange={(v) => set('ingredient_a', v)}
              disabled={!!rule}
              required
            />
          </Field>
          <Field label="Princípio ativo B" required>
            <IngredientInput
              value={form.ingredient_b}
              onChange={(v) => set('ingredient_b', v)}
              disabled={!!rule}
              required
            />
          </Field>
        </div>

        <Field label="Gravidade" required>
          <select value={form.severity} onChange={(e) => set('severity', e.target.value as Severity)}>
            {SEVERITIES.map((s) => <option key={s} value={s}>{SEVERITY_LABEL[s]}</option>)}
          </select>
        </Field>

        <Field label="O que acontece" required hint="Em linguagem simples: é o texto que o app mostra.">
          <textarea value={form.description} onChange={(e) => set('description', e.target.value)} rows={3} required />
        </Field>

        <Field label="Recomendação" hint="O que a pessoa deve fazer.">
          <textarea value={form.recommendation} onChange={(e) => set('recommendation', e.target.value)} rows={2} />
        </Field>

        <Field label="Fonte (URL)" required hint="Bula, Anvisa ou literatura usada para escrever a regra.">
          <input value={form.source_url} onChange={(e) => set('source_url', e.target.value)} required />
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
