import { useState, type FormEvent } from 'react'
import { useAuth } from '../auth/useAuth'
import { Button } from '../components/ui/Button'
import { Card, CardHeader } from '../components/ui/Card'
import { Modal } from '../components/ui/Modal'
import { Notice } from '../components/ui/Notice'
import { TextField } from '../components/ui/TextField'
import { useTitle } from '../hooks/useTitle'
import { ApiError, api, errorMessage } from '../lib/api'
import { formatDate } from '../lib/format'

function ChangePassword() {
  const { adoptSession } = useAuth()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<unknown>(null)
  const [done, setDone] = useState(false)

  const mismatch = confirm.length > 0 && confirm !== next
  const field = (name: string) => (error instanceof ApiError && error.field === name ? error.message : null)

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (next !== confirm) return
    setBusy(true)
    setError(null)
    setDone(false)
    try {
      adoptSession(await api.changePassword({ current_password: current, new_password: next }))
      setCurrent('')
      setNext('')
      setConfirm('')
      setDone(true)
    } catch (err) {
      setError(err)
    } finally {
      setBusy(false)
    }
  }

  const general = error && !(error instanceof ApiError && error.field) ? errorMessage(error) : null

  return (
    <form onSubmit={submit} className="space-y-4 p-5" noValidate>
      {general ? <Notice>{general}</Notice> : null}
      {done ? <Notice tone="success">Password changed. Other devices and browsers have been signed out.</Notice> : null}
      <TextField
        label="Current password"
        type="password"
        autoComplete="current-password"
        value={current}
        onChange={(e) => setCurrent(e.target.value)}
        error={field('current_password')}
      />
      <div className="grid gap-4 sm:grid-cols-2">
        <TextField
          label="New password"
          type="password"
          autoComplete="new-password"
          value={next}
          onChange={(e) => setNext(e.target.value)}
          hint="At least 8 characters."
          error={field('new_password')}
        />
        <TextField
          label="Confirm new password"
          type="password"
          autoComplete="new-password"
          value={confirm}
          onChange={(e) => setConfirm(e.target.value)}
          error={mismatch ? 'Passwords do not match' : null}
        />
      </div>
      <div className="flex justify-end">
        <Button type="submit" loading={busy} disabled={!current || !next || next !== confirm}>
          Change password
        </Button>
      </div>
    </form>
  )
}

function DeleteAccount() {
  const { forget } = useAuth()
  const [open, setOpen] = useState(false)
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const remove = async () => {
    setBusy(true)
    setError(null)
    try {
      await api.deleteAccount(password)
      forget()
    } catch (err) {
      setError(errorMessage(err))
      setBusy(false)
    }
  }

  return (
    <>
      <Button variant="danger" onClick={() => setOpen(true)}>
        Delete account
      </Button>
      <Modal
        open={open}
        title="Delete your account?"
        onClose={() => setOpen(false)}
        footer={
          <>
            <Button variant="secondary" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button variant="danger" loading={busy} disabled={!password} onClick={remove}>
              Delete everything
            </Button>
          </>
        }
      >
        <div className="space-y-4 text-sm">
          <p className="text-ink-soft">
            Your account, devices, readings, thresholds and alerts are deleted permanently.
          </p>
          {error ? <Notice>{error}</Notice> : null}
          <TextField
            label="Confirm with your password"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </div>
      </Modal>
    </>
  )
}

export function AccountPage() {
  useTitle('Account')
  const { user, logout } = useAuth()

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <div>
        <p className="text-xs font-bold uppercase tracking-wide text-forest">Settings</p>
        <h1 className="text-2xl font-extrabold">Account</h1>
      </div>

      <Card>
        <CardHeader title="Profile" actions={<Button variant="secondary" size="sm" onClick={() => void logout()}>Sign out</Button>} />
        <dl className="grid gap-4 p-5 text-sm sm:grid-cols-3">
          <div>
            <dt className="text-muted">Name</dt>
            <dd className="mt-0.5 font-semibold">{user?.name}</dd>
          </div>
          <div className="min-w-0">
            <dt className="text-muted">Email</dt>
            <dd className="mt-0.5 truncate font-semibold">{user?.email}</dd>
          </div>
          <div>
            <dt className="text-muted">Member since</dt>
            <dd className="mt-0.5 font-semibold">{formatDate(user?.created_at)}</dd>
          </div>
        </dl>
      </Card>

      <Card>
        <CardHeader title="Change password" />
        <ChangePassword />
      </Card>

      <Card className="border-danger/30">
        <CardHeader eyebrow="Danger zone" title="Delete account" />
        <div className="flex flex-wrap items-center justify-between gap-3 p-5 text-sm">
          <p className="max-w-md text-ink-soft">Deletes your account and all registered devices and their data.</p>
          <DeleteAccount />
        </div>
      </Card>
    </div>
  )
}
