import { createContext, useCallback, useContext, useState, type ReactNode } from 'react'

type Toast = { id: number; kind: 'ok' | 'error'; text: string }
type Notify = (kind: Toast['kind'], text: string) => void

const Ctx = createContext<Notify>(() => {})

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const notify = useCallback<Notify>((kind, text) => {
    const id = Date.now() + Math.random()
    setToasts((t) => [...t.slice(-2), { id, kind, text }])
    setTimeout(() => setToasts((t) => t.filter((x) => x.id !== id)), 4000)
  }, [])

  return (
    <Ctx.Provider value={notify}>
      {children}
      <div className="toasts">
        {toasts.map((t) => (
          <div key={t.id} className={`toast ${t.kind}`}>{t.text}</div>
        ))}
      </div>
    </Ctx.Provider>
  )
}

export const useToast = () => useContext(Ctx)
