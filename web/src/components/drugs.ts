import { useEffect, useState } from 'react'
import { fetchAllDrugs } from '../api'
import type { Drug } from '../types'

export const drugLabel = (d: Drug) =>
  `${d.brand_name || d.active_ingredient} — ${d.active_ingredient} (${d.registration_number})`

/** Full drug list for the form pickers. */
export function useDrugs() {
  const [drugs, setDrugs] = useState<Drug[]>([])
  useEffect(() => {
    fetchAllDrugs().then(setDrugs).catch(() => setDrugs([]))
  }, [])
  return drugs
}
