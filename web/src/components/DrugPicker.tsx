import { useEffect, useRef, useState } from 'react'
import { api, errorMessage } from '../api'
import type { Drug } from '../types'

type Props = {
  value: number | ''
  onChange: (id: number | '') => void
  autoFocus?: boolean
}

/** How long to wait after the last keystroke before asking the API. */
const DEBOUNCE_MS = 250

const MAX_RESULTS = 30

/**
 * Drug search field: type part of the brand name, active ingredient, company or
 * registration number and pick from the list (mouse or ↑ ↓ Enter).
 *
 * The search runs in the database, not here. It used to download every drug and
 * filter in the browser, which was fine for the few dozen typed by hand and
 * became 293 requests and 6 MB once cmd/anvisa-import loaded the Anvisa base.
 */
export default function DrugPicker({ value, onChange, autoFocus }: Props) {
  const [selected, setSelected] = useState<Drug | null>(null)
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<Drug[]>([])
  const [loading, setLoading] = useState(false)
  const [open, setOpen] = useState(false)
  const [active, setActive] = useState(0)
  const boxRef = useRef<HTMLDivElement>(null)
  const listRef = useRef<HTMLUListElement>(null)

  // The form may open already pointing at a drug (editing a package): only the
  // id is known, so the name is fetched on its own.
  useEffect(() => {
    if (value === '') {
      setSelected(null)
      return
    }
    let alive = true
    api.drugs
      .get(value)
      .then((d) => alive && setSelected(d))
      .catch(() => alive && setSelected(null))
    return () => {
      alive = false
    }
  }, [value])

  // Search as you type, waiting for a pause so one word is one request.
  useEffect(() => {
    if (!open) return
    let alive = true
    setLoading(true)
    const timer = setTimeout(() => {
      api.drugs
        .list(MAX_RESULTS, 0, query.trim())
        .then(({ data }) => alive && setResults(data))
        .catch((e) => {
          if (!alive) return
          setResults([])
          console.error(errorMessage(e))
        })
        .finally(() => alive && setLoading(false))
    }, DEBOUNCE_MS)
    return () => {
      alive = false
      clearTimeout(timer)
    }
  }, [query, open])

  // Close when clicking outside
  useEffect(() => {
    const onDown = (e: MouseEvent) => {
      if (!boxRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDown)
    return () => document.removeEventListener('mousedown', onDown)
  }, [])

  // Keep the active item visible while navigating with the keyboard
  useEffect(() => {
    listRef.current?.children[active]?.scrollIntoView({ block: 'nearest' })
  }, [active])

  const pick = (d: Drug) => {
    setSelected(d)
    onChange(d.id)
    setQuery('')
    setOpen(false)
  }

  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setOpen(true)
      setActive((a) => Math.min(a + 1, results.length - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setActive((a) => Math.max(a - 1, 0))
    } else if (e.key === 'Enter') {
      if (open && results[active]) {
        e.preventDefault()
        pick(results[active])
      }
    } else if (e.key === 'Escape' && open) {
      // don't let Esc close the whole modal, only the list
      e.stopPropagation()
      e.nativeEvent.stopImmediatePropagation()
      setOpen(false)
    }
  }

  return (
    <div className="picker" ref={boxRef}>
      {selected && !open ? (
        <button type="button" className="picker-selected" onClick={() => setOpen(true)}>
          <span>
            <strong>{selected.brand_name || selected.active_ingredients.join(' + ')}</strong>
            <span className="muted"> — {selected.active_ingredients.join(' + ')} · {selected.registration_number}</span>
          </span>
          <span className="muted small">trocar</span>
        </button>
      ) : (
        <input
          value={query}
          autoFocus={autoFocus || open}
          onChange={(e) => {
            setQuery(e.target.value)
            setActive(0)
            setOpen(true)
          }}
          onFocus={() => setOpen(true)}
          onKeyDown={onKeyDown}
          placeholder="Digite o nome, princípio ativo ou registro…"
          role="combobox"
          aria-expanded={open}
          aria-autocomplete="list"
        />
      )}

      {open && (
        <ul className="picker-list" ref={listRef} role="listbox">
          {results.map((d, i) => (
            <li
              key={d.id}
              role="option"
              aria-selected={d.id === value}
              className={i === active ? 'active' : undefined}
              onMouseEnter={() => setActive(i)}
              onMouseDown={(e) => {
                e.preventDefault() // keep focus until the click is registered
                pick(d)
              }}
            >
              <strong>{d.brand_name || d.active_ingredients.join(' + ')}</strong>
              <span className="muted"> — {d.active_ingredients.join(' + ')}</span>
              <div className="muted small">{d.manufacturer} · Reg. {d.registration_number}</div>
            </li>
          ))}
          {!loading && results.length === 0 && (
            <li className="picker-empty">
              {query.trim() ? 'Nenhum remédio encontrado.' : 'Digite para buscar.'}
            </li>
          )}
          {loading && results.length === 0 && <li className="picker-empty">Buscando…</li>}
        </ul>
      )}
    </div>
  )
}
