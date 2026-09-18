import type { ReactNode } from 'react'

type Props = { label: string; required?: boolean; hint?: string; children: ReactNode }

export default function Field({ label, required, hint, children }: Props) {
  return (
    <label className="field">
      <span className="label">
        {label} {required && <em>*</em>}
      </span>
      {children}
      {hint && <small className="hint">{hint}</small>}
    </label>
  )
}
