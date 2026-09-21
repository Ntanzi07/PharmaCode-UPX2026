import { useEffect, useState } from 'react'
import DrugsPage from './pages/DrugsPage'
import PackagesPage from './pages/PackagesPage'
import SummariesPage from './pages/SummariesPage'
import LookupPage from './pages/LookupPage'
import { ToastProvider } from './components/Toast'

const TABS = [
  { key: 'drugs', label: 'Remédios', el: <DrugsPage /> },
  { key: 'packages', label: 'Embalagens', el: <PackagesPage /> },
  { key: 'summaries', label: 'Bulas', el: <SummariesPage /> },
  { key: 'lookup', label: 'Consultar EAN', el: <LookupPage /> },
] as const

type TabKey = (typeof TABS)[number]['key']

function initialTab(): TabKey {
  const h = window.location.hash.slice(1)
  return (TABS.find((t) => t.key === h)?.key ?? 'drugs') as TabKey
}

export default function App() {
  const [tab, setTab] = useState<TabKey>(initialTab)

  // Voltar/avançar do navegador troca de aba
  useEffect(() => {
    const onHash = () => setTab(initialTab())
    window.addEventListener('hashchange', onHash)
    return () => window.removeEventListener('hashchange', onHash)
  }, [])

  const select = (k: TabKey) => {
    setTab(k)
    window.location.hash = k
  }

  return (
    <ToastProvider>
      <header className="topbar">
        <div className="brand">
          <span className="logo">℞</span> PharmaCode <small>admin</small>
        </div>
        <nav className="tabs">
          {TABS.map((t) => (
            <button
              key={t.key}
              className={t.key === tab ? 'tab active' : 'tab'}
              onClick={() => select(t.key)}
            >
              {t.label}
            </button>
          ))}
        </nav>
      </header>
      <main className="container">{TABS.find((t) => t.key === tab)!.el}</main>
    </ToastProvider>
  )
}
