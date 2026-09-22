import { useRef } from 'react'

type Props = { value: string[]; onChange: (eans: string[]) => void }

/**
 * EAN list editor: one field per code, with buttons to add and remove.
 * Enter in a field creates the next one (handy with barcode scanners, which send Enter at the end).
 */
export default function EanListInput({ value, onChange }: Props) {
  const refs = useRef<(HTMLInputElement | null)[]>([])
  const eans = value.length ? value : ['']

  const set = (i: number, v: string) => onChange(eans.map((e, j) => (j === i ? v.replace(/\D/g, '') : e)))
  const remove = (i: number) => onChange(eans.filter((_, j) => j !== i))
  const add = () => {
    onChange([...eans, ''])
    setTimeout(() => refs.current[eans.length]?.focus())
  }

  return (
    <div className="ean-list">
      {eans.map((ean, i) => (
        <div key={i} className="ean-row">
          <input
            ref={(el) => { refs.current[i] = el }}
            value={ean}
            onChange={(e) => set(i, e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault()
                if (ean) add()
              }
            }}
            inputMode="numeric"
            maxLength={13}
            placeholder="7891234567890"
            aria-label={`EAN ${i + 1}`}
          />
          <button type="button" className="icon" onClick={() => remove(i)} disabled={eans.length === 1} aria-label="Remover EAN">
            ×
          </button>
        </div>
      ))}
      <button type="button" className="link" onClick={add}>+ adicionar outro EAN</button>
    </div>
  )
}
