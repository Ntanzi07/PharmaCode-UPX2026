type Props = { offset: number; limit: number; count: number; onChange: (offset: number) => void }

export default function Pager({ offset, limit, count, onChange }: Props) {
  const page = Math.floor(offset / limit) + 1
  return (
    <div className="pager">
      <button disabled={offset === 0} onClick={() => onChange(Math.max(0, offset - limit))}>
        ← Anterior
      </button>
      <span>Página {page}</span>
      <button disabled={count < limit} onClick={() => onChange(offset + limit)}>
        Próxima →
      </button>
    </div>
  )
}
