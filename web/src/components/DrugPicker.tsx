import { useEffect, useMemo, useRef, useState } from 'react'
import type { Drug } from '../types'

type Props = {
  drugs: Drug[]
  value: number | ''
  onChange: (id: number | '') => void
  autoFocus?: boolean
}

/** Strips accents and case for searching: "IBUPROFÉNO" matches "ibuprofeno". */
const norm = (s: string) => s.normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase()

const MAX_RESULTS = 50

/**
 * Drug search field: type part of the brand name, active ingredient
 * or registration number and pick from the list (mouse or ↑ ↓ Enter).
 */
export default function DrugPicker({ drugs, value, onChange, autoFocus }: Props) {
  const selected = drugs.find((d) => d.id === value)
  const [query, setQuery] = useState('')
  const [open, setOpen] = useState(false)
  const [active, setActive] = useState(0)
  const boxRef = useRef<HTMLDivElement>(null)
  const listRef = useRef<HTMLUListElement>(null)

  const results = useMemo(() => {
    const words = norm(query).split(/\s+/).filter(Boolean)
    const hits = drugs.filter((d) => {
      const text = norm(`${d.brand_name ?? ''} ${d.active_ingredient} ${d.manufacturer} ${d.registration_number}`)
      return words.every((w) => text.includes(w))
    })
    return hits.slice(0, MAX_RESULTS)
  }, [drugs, query])

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
            <strong>{selected.brand_name || selected.active_ingredient}</strong>
            <span className="muted"> — {selected.active_ingredient} · {selected.registration_number}</span>
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
              <strong>{d.brand_name || d.active_ingredient}</strong>
              <span className="muted"> — {d.active_ingredient}</span>
              <div className="muted small">{d.manufacturer} · Reg. {d.registration_number}</div>
            </li>
          ))}
          {results.length === 0 && <li className="picker-empty">Nenhum remédio encontrado.</li>}
        </ul>
      )}
    </div>
  )
}
