import { useRef } from 'react'

type Props = { value: string[]; onChange: (eans: string[]) => void }

/**
 * Editor de lista de EANs: um campo por código, com botão para adicionar e remover.
 * Enter num campo cria o próximo (útil com leitor de código de barras, que manda Enter no final).
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
