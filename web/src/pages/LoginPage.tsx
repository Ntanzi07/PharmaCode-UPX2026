import { useState, type FormEvent } from 'react'
import { ApiError, errorMessage } from '../api'
import { useAuth } from '../components/auth'

export default function LoginPage() {
  const { login, expired } = useAuth()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [err, setErr] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setErr(null)
    try {
      await login(email, password)
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) setErr('Email ou senha incorretos.')
      else if (e instanceof ApiError && e.status === 429) setErr('Muitas tentativas. Espere alguns minutos e tente de novo.')
      else setErr(errorMessage(e))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-wrap">
      <form className="login-card" onSubmit={submit}>
        <div className="brand login-brand">
          <span className="logo">P</span> PharmaCode
        </div>
        <p className="muted">Entre com a sua conta para cadastrar e revisar bulas.</p>
        {expired && !err && <div className="alert info">Sua sessão terminou. Entre de novo.</div>}
        <label className="field">
          <span className="label">Email</span>
          <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="username" required autoFocus />
        </label>
        <label className="field">
          <span className="label">Senha</span>
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="current-password" required />
        </label>
        {err && <div className="alert">{err}</div>}
        <button className="primary block" disabled={loading}>{loading ? 'Entrando…' : 'Entrar'}</button>
      </form>
    </div>
  )
}
