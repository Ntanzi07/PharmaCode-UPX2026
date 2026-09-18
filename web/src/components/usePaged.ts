import { useCallback, useEffect, useState } from 'react'
import { errorMessage } from '../api'
import type { Page } from '../types'

export const PAGE_SIZE = 20

/** Carrega uma lista paginada e expõe reload() para depois de criar/editar/remover. */
export function usePaged<T>(fetcher: (limit: number, offset: number) => Promise<Page<T>>) {
  const [offset, setOffset] = useState(0)
  const [rows, setRows] = useState<T[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const reload = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetcher(PAGE_SIZE, offset)
      setRows(res.data)
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setLoading(false)
    }
  }, [fetcher, offset])

  useEffect(() => {
    reload()
  }, [reload])

  return { rows, loading, error, offset, setOffset, reload }
}
