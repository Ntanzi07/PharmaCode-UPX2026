import { useEffect, useMemo, useRef, useState } from 'react'
import type { Drug } from '../types'

type Props = {
  drugs: Drug[]
  value: number | ''
  onChange: (id: number | '') => void
  autoFocus?: boolean
}

/** Tira acento e caixa para a busca: "IBUPROFÉNO" encontra "ibuprofeno". */
const norm = (s: string) => s.normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase()

const MAX_RESULTS = 50

/**
 * Campo de busca de remédio: digite parte do nome comercial, do princípio ativo
 * ou do número de registro e escolha na lista (mouse ou ↑ ↓ Enter).
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

  // Fecha ao clicar fora
  useEffect(() => {
    const onDown = (e: MouseEvent) => {
      if (!boxRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDown)
    return () => document.removeEventListener('mousedown', onDown)
  }, [])

  // Mantém o item ativo visível ao navegar pelo teclado
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
      // não deixa o Esc fechar o modal inteiro, só a lista
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
                e.preventDefault() // não tira o foco antes do clique contar
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
