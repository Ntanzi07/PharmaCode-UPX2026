import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { api, errorMessage } from '../api'
import { ROLE_HINT, ROLE_LABEL, type Role, type UserRow } from '../types'
import Modal from '../components/Modal'
import Field from '../components/Field'
import { useToast } from '../components/Toast'
import { useAuth } from '../components/auth'
import { fmtDate } from '../components/format'

const ROLES: Role[] = ['editor', 'reviewer', 'admin']

type Editing = { mode: 'new' } | { mode: 'edit'; user: UserRow } | { mode: 'password'; user: UserRow }

export default function UsersPage() {
  const [rows, setRows] = useState<UserRow[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [editing, setEditing] = useState<Editing | null>(null)
  const { user: me } = useAuth()

  const reload = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setRows(await api.users.list())
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    reload()
  }, [reload])

  const done = () => { setEditing(null); reload() }

  return (
    <section>
      <div className="section-head">
        <h1>Usuários</h1>
        <button className="primary" onClick={() => setEditing({ mode: 'new' })}>+ Novo usuário</button>
      </div>

      {error && <div className="alert">{error}</div>}

      <div className="card">
        <table>
          <thead>
            <tr><th>Nome</th><th>Email</th><th>Papel</th><th>Status</th><th>Criado</th><th /></tr>
          </thead>
          <tbody>
            {rows.map((u) => (
              <tr key={u.id} className={u.active ? undefined : 'inactive'}>
                <td>
                  <strong>{u.name}</strong>
                  {u.id === me?.id && <span className="muted small"> (você)</span>}
                </td>
                <td>{u.email}</td>
                <td><span className={`badge role-${u.role}`}>{ROLE_LABEL[u.role]}</span></td>
                <td>{u.active ? <span className="badge ok">Ativo</span> : <span className="badge">Desativado</span>}</td>
                <td className="muted">{fmtDate(u.created_at)}</td>
                <td className="actions">
                  <button onClick={() => setEditing({ mode: 'edit', user: u })}>Editar</button>
                  <button onClick={() => setEditing({ mode: 'password', user: u })}>Definir senha</button>
                </td>
              </tr>
            ))}
            {!loading && rows.length === 0 && (
              <tr><td colSpan={6} className="empty">Nenhum usuário.</td></tr>
            )}
          </tbody>
        </table>
        {loading && <div className="loading">Carregando…</div>}
      </div>
      <p className="muted small">
        Usuários não são apagados, só desativados: assim o histórico de quem revisou cada bula continua válido.
      </p>

      {editing?.mode === 'new' && <UserForm onClose={() => setEditing(null)} onSaved={done} />}
      {editing?.mode === 'edit' && <UserForm user={editing.user} isMe={editing.user.id === me?.id} onClose={() => setEditing(null)} onSaved={done} />}
      {editing?.mode === 'password' && <PasswordForm user={editing.user} onClose={() => setEditing(null)} onSaved={done} />}
    </section>
  )
}

type FormProps = { user?: UserRow; isMe?: boolean; onClose: () => void; onSaved: () => void }

function UserForm({ user, isMe, onClose, onSaved }: FormProps) {
  const [name, setName] = useState(user?.name ?? '')
  const [email, setEmail] = useState(user?.email ?? '')
  const [role, setRole] = useState<Role>(user?.role ?? 'editor')
  const [active, setActive] = useState(user?.active ?? true)
  const [password, setPassword] = useState('')
  const [err, setErr] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const notify = useToast()

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setErr(null)
    try {
      if (user) {
        await api.users.update(user.id, { name, email, role, active })
        notify('ok', 'Usuário atualizado')
      } else {
        await api.users.create({ name, email, password, role })
        notify('ok', 'Usuário criado')
      }
      onSaved()
    } catch (e) {
      setErr(errorMessage(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title={user ? 'Editar usuário' : 'Novo usuário'} onClose={onClose}>
      <form className="form" onSubmit={submit}>
        <Field label="Nome" required hint="Aparece como quem revisou a bula.">
          <input value={name} onChange={(e) => setName(e.target.value)} required autoFocus />
        </Field>
        <Field label="Email" required>
          <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
        </Field>
        {!user && (
          <Field label="Senha inicial" required hint="De 8 a 72 caracteres. Passe para a pessoa trocar no primeiro acesso.">
            <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={8} maxLength={72} autoComplete="new-password" />
          </Field>
        )}
        <Field label="Papel" required hint={ROLE_HINT[role]}>
          <select value={role} onChange={(e) => setRole(e.target.value as Role)}>
            {ROLES.map((r) => <option key={r} value={r}>{ROLE_LABEL[r]}</option>)}
          </select>
        </Field>
        {user && (
          <label className="check">
            <input type="checkbox" checked={active} onChange={(e) => setActive(e.target.checked)} disabled={isMe} />
            <span>Conta ativa {isMe && <span className="muted small">(você não pode desativar a si mesmo)</span>}</span>
          </label>
        )}
        {user && (role !== user.role || !active) && (
          <div className="alert info">Salvar vai encerrar as sessões abertas deste usuário.</div>
        )}
        {err && <div className="alert">{err}</div>}
        <div className="form-actions">
          <button type="button" onClick={onClose}>Cancelar</button>
          <button className="primary" disabled={saving}>{saving ? 'Salvando…' : 'Salvar'}</button>
        </div>
      </form>
    </Modal>
  )
}

function PasswordForm({ user, onClose, onSaved }: { user: UserRow; onClose: () => void; onSaved: () => void }) {
  const [password, setPassword] = useState('')
  const [err, setErr] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const notify = useToast()

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setErr(null)
    try {
      await api.users.setPassword(user.id, password)
      notify('ok', `Senha de ${user.name} definida`)
      onSaved()
    } catch (e) {
      setErr(errorMessage(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title={`Definir senha de ${user.name}`} onClose={onClose}>
      <form className="form" onSubmit={submit}>
        <Field label="Senha nova" required hint="De 8 a 72 caracteres. As sessões abertas dessa pessoa serão encerradas.">
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={8} maxLength={72} autoComplete="new-password" autoFocus />
        </Field>
        {err && <div className="alert">{err}</div>}
        <div className="form-actions">
          <button type="button" onClick={onClose}>Cancelar</button>
          <button className="primary" disabled={saving}>{saving ? 'Salvando…' : 'Definir senha'}</button>
        </div>
      </form>
    </Modal>
  )
}
