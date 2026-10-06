import { useEffect, useState } from 'react'
import { api } from '../api'
import type { Drug } from '../types'

export const drugLabel = (d: Drug) =>
  `${d.brand_name || d.active_ingredients.join(' + ')} — ${d.active_ingredients.join(' + ')} (${d.registration_number})`

/** One drug by id, for a form that opens already pointing at it. */
export function useDrug(id: number | '') {
  const [drug, setDrug] = useState<Drug | null>(null)
  useEffect(() => {
    if (id === '') {
      setDrug(null)
      return
    }
    let alive = true
    api.drugs
      .get(id)
      .then((d) => alive && setDrug(d))
      .catch(() => alive && setDrug(null))
    return () => {
      alive = false
    }
  }, [id])
  return drug
}
