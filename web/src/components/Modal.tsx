import { useEffect, type ReactNode } from 'react'

type Props = { title: string; onClose: () => void; children: ReactNode; wide?: boolean }

export default function Modal({ title, onClose, children, wide }: Props) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && onClose()
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  return (
    <div className="backdrop" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div className={wide ? 'modal wide' : 'modal'} role="dialog" aria-label={title}>
        <div className="modal-head">
          <h2>{title}</h2>
          <button className="icon" onClick={onClose} aria-label="Fechar">×</button>
        </div>
        {children}
      </div>
    </div>
  )
}
