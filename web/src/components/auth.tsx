import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { api, ApiError, SESSION_EXPIRED } from '../api'
import type { Role, User } from '../types'

const LEVEL: Record<Role, number> = { editor: 1, reviewer: 2, admin: 3 }

type AuthState = {
  user: User | null
  /** true while checking for an existing session (first load) */
  checking: boolean
  /** true when the session dropped while in use (shows a notice on the login screen) */
  expired: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  /** does the user have at least this role? (same rule as the API) */
  can: (min: Role) => boolean
}

const Ctx = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [checking, setChecking] = useState(true)
  const [expired, setExpired] = useState(false)

  // When the panel opens: if the session cookie is still valid, log in right away
  useEffect(() => {
    api.auth
      .me()
      .then(setUser)
      .catch((e) => {
        if (!(e instanceof ApiError && e.status === 401)) console.error(e)
      })
      .finally(() => setChecking(false))
  }, [])

  // Any 401 while in use sends the user back to the login screen
  useEffect(() => {
    // (only calls outside /auth fire the event, and those only happen while logged in)
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
