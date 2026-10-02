import { useRef } from 'react'

type Props = {
  value: string[]
  onChange: (values: string[]) => void
  placeholder?: string
  addLabel?: string
  /** keeps only digits, for barcodes */
  digitsOnly?: boolean
  maxLength?: number
  label?: string
}

/**
 * Edits a list of values: one field per item, with buttons to add and remove.
 * Enter in a field creates the next one (handy with barcode scanners, which
 * send Enter at the end).
 */
export default function ListInput({
  value, onChange, placeholder, addLabel = '+ adicionar outro', digitsOnly, maxLength, label = 'item',
}: Props) {
  const refs = useRef<(HTMLInputElement | null)[]>([])
  const items = value.length ? value : ['']

  const set = (i: number, v: string) =>
    onChange(items.map((e, j) => (j === i ? (digitsOnly ? v.replace(/\D/g, '') : v) : e)))
  const remove = (i: number) => onChange(items.filter((_, j) => j !== i))
  const add = () => {
    onChange([...items, ''])
    setTimeout(() => refs.current[items.length]?.focus())
  }

  return (
    <div className="ean-list">
      {items.map((item, i) => (
        <div key={i} className="ean-row">
          <input
            ref={(el) => { refs.current[i] = el }}
            value={item}
            onChange={(e) => set(i, e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault()
                if (item) add()
              }
            }}
            inputMode={digitsOnly ? 'numeric' : undefined}
            maxLength={maxLength}
            placeholder={placeholder}
            aria-label={`${label} ${i + 1}`}
          />
          <button type="button" className="icon" onClick={() => remove(i)} disabled={items.length === 1} aria-label={`Remover ${label}`}>
            ×
          </button>
        </div>
      ))}
      <button type="button" className="link" onClick={add}>{addLabel}</button>
    </div>
  )
}
