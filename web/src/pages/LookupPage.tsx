import { useState, type FormEvent } from 'react'
import { api, ApiError, errorMessage } from '../api'
import { SUMMARY_TEXT_FIELDS, type EanSummary } from '../types'
import { fmtDay } from '../components/format'

/** Shows what the end-user app will receive when it scans the barcode. */
export default function LookupPage() {
  const [ean, setEan] = useState('')
  const [result, setResult] = useState<EanSummary | null>(null)
  const [err, setErr] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const search = async (e: FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setErr(null)
    setResult(null)
    try {
      setResult(await api.drugs.byEan(ean.trim()))
    } catch (e) {
      setErr(e instanceof ApiError && e.status === 404 ? 'Nenhum remédio com bula encontrado para esse EAN.' : errorMessage(e))
    } finally {
      setLoading(false)
    }
  }

  return (
    <section>
      <div className="section-head">
        <h1>Consultar por EAN</h1>
      </div>
      <form className="search" onSubmit={search}>
        <input
          value={ean}
          onChange={(e) => setEan(e.target.value)}
          placeholder="Digite ou escaneie o código de barras"
          inputMode="numeric"
          autoFocus
          required
        />
        <button className="primary" disabled={loading}>{loading ? 'Buscando…' : 'Buscar'}</button>
      </form>

      {err && <div className="alert">{err}</div>}

      {result && (
        <div className="card leaflet">
          <div className="leaflet-head">
            <h2>{result.brand_name || result.active_ingredient}</h2>
            <p className="muted">
              {result.active_ingredient} · {result.manufacturer} · Reg. {result.registration_number}
            </p>
            <p className="muted small">
              EAN {result.ean} — {result.description}
              {result.presentation_registration && <> · Apresentação {result.presentation_registration}</>}
            </p>
          </div>
          {SUMMARY_TEXT_FIELDS.map((f) =>
            result[f.key] ? (
              <div key={f.key} className="leaflet-block">
                <h3>{f.label}</h3>
                <p>{result[f.key]}</p>
              </div>
            ) : null,
          )}
          {!result.what_is_it_for && (
            <p className="muted">Este remédio ainda não tem bula simplificada revisada.</p>
          )}
          {(result.leaflet_expedient || result.leaflet_published_at || result.source_url) && (
            <p className="leaflet-foot muted small">
              Baseado na bula
              {result.leaflet_expedient && <> expediente {result.leaflet_expedient}</>}
              {result.leaflet_published_at && <> publicada em {fmtDay(result.leaflet_published_at)}</>}
              {result.source_url && (
                <> · <a href={result.source_url} target="_blank" rel="noreferrer">fonte</a></>
              )}
            </p>
          )}
        </div>
      )}
    </section>
  )
}
