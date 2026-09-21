import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { api, ApiError, SESSION_EXPIRED } from '../api'
import type { Role, User } from '../types'

const LEVEL: Record<Role, number> = { editor: 1, reviewer: 2, admin: 3 }

type AuthState = {
  user: User | null
  /** true enquanto confere se já existe sessão (primeira carga) */
  checking: boolean
  /** true quando a sessão caiu no meio do uso (mostra aviso no login) */
  expired: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  /** o usuário tem pelo menos esse papel? (mesma regra da API) */
  can: (min: Role) => boolean
}

const Ctx = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [checking, setChecking] = useState(true)
  const [expired, setExpired] = useState(false)

  // Ao abrir o painel: se o cookie de sessão ainda vale, já entra logado
  useEffect(() => {
    api.auth
      .me()
      .then(setUser)
      .catch((e) => {
        if (!(e instanceof ApiError && e.status === 401)) console.error(e)
      })
      .finally(() => setChecking(false))
  }, [])

  // Qualquer 401 no meio do uso derruba para a tela de login
  useEffect(() => {
    // (só chamadas de fora de /auth disparam o evento, e elas só acontecem logado)
    const onExpired = () => {
      setExpired(true)
      setUser(null)
    }
    window.addEventListener(SESSION_EXPIRED, onExpired)
    return () => window.removeEventListener(SESSION_EXPIRED, onExpired)
  }, [])

  const login = useCallback(async (email: string, password: string) => {
    const u = await api.auth.login(email, password)
    setExpired(false)
    setUser(u)
  }, [])

  const logout = useCallback(async () => {
    try {
      await api.auth.logout()
    } finally {
      setUser(null)
    }
  }, [])

  const can = useCallback((min: Role) => !!user && LEVEL[user.role] >= LEVEL[min], [user])

  return <Ctx.Provider value={{ user, checking, expired, login, logout, can }}>{children}</Ctx.Provider>
}

export function useAuth(): AuthState {
  const ctx = useContext(Ctx)
  if (!ctx) throw new Error('useAuth fora do AuthProvider')
  return ctx
}
