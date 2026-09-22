import { useEffect, useRef, useState } from 'react'
import DrugsPage from './pages/DrugsPage'
import PackagesPage from './pages/PackagesPage'
import SummariesPage from './pages/SummariesPage'
import LookupPage from './pages/LookupPage'
import UsersPage from './pages/UsersPage'
import LoginPage from './pages/LoginPage'
import { ToastProvider } from './components/Toast'
import { AuthProvider, useAuth } from './components/auth'
import ChangePasswordModal from './components/ChangePasswordModal'
import { ROLE_LABEL, type Role } from './types'

const TABS: { key: string; label: string; min: Role; el: React.ReactNode }[] = [
  { key: 'drugs', label: 'Remédios', min: 'editor', el: <DrugsPage /> },
  { key: 'packages', label: 'Embalagens', min: 'editor', el: <PackagesPage /> },
  { key: 'summaries', label: 'Bulas', min: 'editor', el: <SummariesPage /> },
  { key: 'lookup', label: 'Consultar EAN', min: 'editor', el: <LookupPage /> },
  { key: 'users', label: 'Usuários', min: 'admin', el: <UsersPage /> },
]

const hashTab = () => window.location.hash.slice(1)

export default function App() {
  return (
    <ToastProvider>
      <AuthProvider>
        <Shell />
      </AuthProvider>
    </ToastProvider>
  )
}

function Shell() {
  const { user, checking, can } = useAuth()
  const [tab, setTab] = useState(hashTab)

  // Browser back/forward switches tabs
  useEffect(() => {
    const onHash = () => setTab(hashTab())
    window.addEventListener('hashchange', onHash)
    return () => window.removeEventListener('hashchange', onHash)
  }, [])

  if (checking) return <div className="loading full">Carregando…</div>
  if (!user) return <LoginPage />

  const tabs = TABS.filter((t) => can(t.min))
  const current = tabs.find((t) => t.key === tab) ?? tabs[0]

  const select = (k: string) => {
    setTab(k)
    window.location.hash = k
  }

  return (
    <>
      <header className="topbar">
        <div className="brand">
          <span className="logo">P</span> PharmaCode
        </div>
        <nav className="tabs">
          {tabs.map((t) => (
            <button key={t.key} className={t.key === current.key ? 'tab active' : 'tab'} onClick={() => select(t.key)}>
              {t.label}
            </button>
          ))}
        </nav>
        <UserMenu />
      </header>
      {/* key: switching users remounts the page and reloads the data */}
      <main className="container" key={user.id}>{current.el}</main>
    </>
  )
}

function UserMenu() {
  const { user, logout } = useAuth()
  const [open, setOpen] = useState(false)
  const [changingPassword, setChangingPassword] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const onDown = (e: MouseEvent) => {
      if (!ref.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDown)
    return () => document.removeEventListener('mousedown', onDown)
  }, [])

  if (!user) return null
  const initials = user.name.split(/\s+/).map((p) => p[0]).slice(0, 2).join('').toUpperCase()

  return (
    <div className="user-menu" ref={ref}>
      <button className="user-btn" onClick={() => setOpen((o) => !o)} aria-expanded={open}>
        <span className="avatar">{initials}</span>
        <span className="user-name">{user.name}</span>
      </button>
      {open && (
        <div className="menu">
          <div className="menu-head">
            <strong>{user.name}</strong>
            <div className="muted small">{user.email}</div>
            <span className={`badge role-${user.role}`}>{ROLE_LABEL[user.role]}</span>
          </div>
          <button onClick={() => { setOpen(false); setChangingPassword(true) }}>Trocar minha senha</button>
          <button className="danger" onClick={() => logout()}>Sair</button>
        </div>
      )}
      {changingPassword && <ChangePasswordModal onClose={() => setChangingPassword(false)} />}
    </div>
  )
}
