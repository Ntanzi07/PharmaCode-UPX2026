export function fmtDate(s: string | null | undefined): string {
  if (!s) return '—'
  const d = new Date(s)
  return isNaN(d.getTime()) ? '—' : d.toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' })
}

/** "2024-05-31" -> "31/05/2024" (sem passar por Date, para não sofrer com fuso) */
export function fmtDay(s: string | null | undefined): string {
  if (!s) return '—'
  const [y, m, d] = s.split('-')
  return d && m && y ? `${d}/${m}/${y}` : s
}
