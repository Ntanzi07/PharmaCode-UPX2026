import { useEffect, useId, useState } from 'react'
import { api } from '../api'
import type { IngredientSuggestion } from '../types'

type Props = {
  value: string
  onChange: (value: string) => void
  disabled?: boolean
  required?: boolean
}

const DEBOUNCE_MS = 250

/**
 * Active ingredient name with suggestions from the database.
 *
 * The count of drugs beside each name is the point of it: warfarin is in the
 * base as "varfarina" (1 drug), "varfarina sódica" (7) and "varfarina sódica
 * cristalina" (1), and a rule only fires for the exact name it was written
 * with. Seeing the counts is what stops a rule being written against the
 * spelling no drug actually uses.
 */
export default function IngredientInput({ value, onChange, disabled, required }: Props) {
  const listId = useId()
  const [options, setOptions] = useState<IngredientSuggestion[]>([])

  useEffect(() => {
    if (disabled) return
    let alive = true
    const timer = setTimeout(() => {
      api.interactions
        .ingredients(value.trim(), 10)
        .then((found) => alive && setOptions(found))
        .catch(() => alive && setOptions([]))
    }, DEBOUNCE_MS)
    return () => {
      alive = false
      clearTimeout(timer)
    }
  }, [value, disabled])

  const exact = options.find((o) => o.name.toLowerCase() === value.trim().toLowerCase())

  return (
    <>
      <input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        list={listId}
        disabled={disabled}
        required={required}
        placeholder="Ex.: ibuprofeno"
      />
      <datalist id={listId}>
        {options.map((o) => (
          <option key={o.id} value={o.name}>
            {o.drugs} remédio(s)
          </option>
        ))}
      </datalist>
      {!disabled && value.trim() !== '' && (
        <small className="hint">
          {exact
            ? `${exact.drugs} remédio(s) com esse princípio ativo`
            : 'Ainda não existe no banco: vai ser criado com esse nome'}
        </small>
      )}
    </>
  )
}
