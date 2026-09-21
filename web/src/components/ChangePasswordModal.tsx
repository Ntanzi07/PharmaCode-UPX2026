import { useState, type FormEvent } from 'react'
import { api, ApiError, errorMessage } from '../api'
import Modal from './Modal'
import Field from './Field'
import { useToast } from './Toast'

export default function ChangePasswordModal({ onClose }: { onClose: () => void }) {
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [err, setErr] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const notify = useToast()

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (next.length < 8) return setErr('A senha nova precisa ter pelo menos 8 caracteres.')
    if (next !== confirm) return setErr('A confirmação não bate com a senha nova.')
    setSaving(true)
    setErr(null)
    try {
      await api.auth.changePassword(current, next)
      notify('ok', 'Senha alterada. Suas outras sessões foram encerradas.')
      onClose()
    } catch (e) {
      setErr(e instanceof ApiError && e.status === 403 ? 'Senha atual incorreta.' : errorMessage(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title="Trocar minha senha" onClose={onClose}>
      <form className="form" onSubmit={submit}>
        <Field label="Senha atual" required>
          <input type="password" value={current} onChange={(e) => setCurrent(e.target.value)} autoComplete="current-password" required autoFocus />
        </Field>
        <Field label="Senha nova" required hint="De 8 a 72 caracteres.">
          <input type="password" value={next} onChange={(e) => setNext(e.target.value)} autoComplete="new-password" required minLength={8} maxLength={72} />
        </Field>
        <Field label="Confirme a senha nova" required>
          <input type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} autoComplete="new-password" required />
        </Field>
        {err && <div className="alert">{err}</div>}
        <div className="form-actions">
          <button type="button" onClick={onClose}>Cancelar</button>
          <button className="primary" disabled={saving}>{saving ? 'Salvando…' : 'Trocar senha'}</button>
        </div>
      </form>
    </Modal>
  )
}
