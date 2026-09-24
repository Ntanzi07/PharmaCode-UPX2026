import { useState, type ChangeEvent } from 'react'
import { api, errorMessage } from '../api'
import type { ImportCounts, ImportResult } from '../types'
import { useToast } from '../components/Toast'

const STEPS = [
  ['drugs', 'Remédios'],
  ['packages', 'Embalagens'],
  ['eans', 'EANs'],
  ['summaries', 'Bulas'],
] as const

export default function ImportPage() {
  const [file, setFile] = useState<File | null>(null)
  const [result, setResult] = useState<ImportResult | null>(null)
  const [busy, setBusy] = useState<'preview' | 'apply' | null>(null)
  const [err, setErr] = useState<string | null>(null)
  const notify = useToast()

  const pick = (e: ChangeEvent<HTMLInputElement>) => {
    setFile(e.target.files?.[0] ?? null)
    setResult(null)
    setErr(null)
  }

  const run = async (mode: 'preview' | 'apply') => {
    if (!file) return
    setBusy(mode)
    setErr(null)
    try {
      const res = mode === 'preview' ? await api.imports.preview(file) : await api.imports.apply(file)
      setResult(res)
      if (res.applied) notify('ok', 'Planilha importada')
      else if (res.errors.length > 0) notify('error', `A planilha tem ${res.errors.length} erro(s)`)
    } catch (e) {
      setErr(errorMessage(e))
      setResult(null)
    } finally {
      setBusy(null)
    }
  }

  const hasErrors = !!result && result.errors.length > 0
  const canApply = !!result && !result.applied && !hasErrors

  return (
    <section>
      <div className="section-head">
        <h1>Importar planilha</h1>
        <a className="button" href={api.imports.templateUrl} download>⤓ Baixar modelo (.xlsx)</a>
      </div>

      <div className="card pad">
        <ol className="steps">
          <li>Baixe o modelo e preencha as abas <strong>remedios</strong> e <strong>embalagens</strong>. A aba <strong>instrucoes</strong> explica cada coluna.</li>
          <li>Suba o arquivo e clique em <strong>Conferir</strong>: nada é gravado, você só vê o que seria criado e os erros.</li>
          <li>Se estiver tudo certo, clique em <strong>Importar</strong>.</li>
        </ol>
        <p className="muted small">
          Importar de novo o mesmo arquivo atualiza os registros, não duplica. As bulas importadas entram como
          não revisadas e precisam da revisão de um farmacêutico para aparecer no app.
        </p>

        <div className="upload">
          <input type="file" accept=".xlsx" onChange={pick} />
          <button onClick={() => run('preview')} disabled={!file || busy !== null}>
            {busy === 'preview' ? 'Conferindo…' : 'Conferir'}
          </button>
          <button className="primary" onClick={() => run('apply')} disabled={!canApply || busy !== null}>
            {busy === 'apply' ? 'Importando…' : 'Importar'}
          </button>
        </div>
      </div>

      {err && <div className="alert">{err}</div>}

      {result && (
        <>
          <div className={result.applied ? 'alert info' : hasErrors ? 'alert' : 'alert info'}>
            {result.applied
              ? 'Importado. Os dados já estão no banco.'
              : hasErrors
                ? 'Nada foi gravado. Corrija os erros abaixo e confira de novo.'
                : 'Prévia: nada foi gravado ainda. Confira os números e clique em Importar.'}
          </div>

          <div className="card">
            <table>
              <thead>
                <tr><th>O que</th><th>Novos</th><th>Atualizados</th></tr>
              </thead>
              <tbody>
                {STEPS.map(([key, label]) => {
                  const c = result[key] as ImportCounts
                  return (
                    <tr key={key}>
                      <td>{label}</td>
                      <td>{c.created}</td>
                      <td>{c.updated}</td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>

          {hasErrors && (
            <div className="card errors">
              <table>
                <thead>
                  <tr><th>Aba</th><th>Linha</th><th>Coluna</th><th>Problema</th></tr>
                </thead>
                <tbody>
                  {result.errors.map((e, i) => (
                    <tr key={i}>
                      <td>{e.sheet}</td>
                      <td className="mono">{e.line}</td>
                      <td className="mono">{e.column}</td>
                      <td>{e.message}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
    </section>
  )
}
