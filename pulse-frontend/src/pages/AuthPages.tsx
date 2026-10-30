import { useState, type FormEvent, type ReactNode } from 'react'
import { Link } from 'react-router'
import { useAuth } from '../auth/useAuth'
import { Logo } from '../components/Logo'
import { Button } from '../components/ui/Button'
import { Notice } from '../components/ui/Notice'
import { TextField } from '../components/ui/TextField'
import { useTitle } from '../hooks/useTitle'
import { ApiError, errorMessage } from '../lib/api'

function AuthShell({ title, subtitle, children, footer }: { title: string; subtitle: string; children: ReactNode; footer: ReactNode }) {
  return (
    <div className="grid min-h-screen lg:grid-cols-2">
      <aside className="hidden flex-col justify-between bg-ink p-10 text-white/80 lg:flex">
        <div className="flex items-center gap-2 font-extrabold text-white">
          <Logo />
          PulsePonics
        </div>
        <div>
          <p className="text-xs font-bold uppercase tracking-wide text-mint">Hydroponics telemetry</p>
          <p className="mt-3 max-w-md text-3xl font-extrabold leading-tight text-white">
            pH, EC and temperature from your tank, live, with alerts when they drift.
          </p>
          <p className="mt-4 max-w-md text-sm leading-6">
            An ESP32 publishes sensor readings over MQTT. The backend stores them, checks your thresholds and pushes
            every update to this dashboard in real time.
          </p>
        </div>
        <p className="text-xs text-white/50">ESP32 · MQTT · Go · MySQL · React</p>
      </aside>
      <main className="flex items-center justify-center px-4 py-12">
        <div className="w-full max-w-sm">
          <div className="mb-8 flex items-center gap-2 font-extrabold lg:hidden">
            <Logo />
            PulsePonics
          </div>
          <h1 className="text-2xl font-extrabold">{title}</h1>
          <p className="mt-1 text-sm text-muted">{subtitle}</p>
          <div className="mt-6">{children}</div>
          <p className="mt-6 text-sm text-muted">{footer}</p>
        </div>
      </main>
    </div>
  )
}

function fieldError(err: unknown, field: string) {
  return err instanceof ApiError && err.field === field ? err.message : null
}

export function LoginPage() {
  useTitle('Sign in')
  const { login } = useAuth()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<unknown>(null)
  const [busy, setBusy] = useState(false)

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await login(email, password)
    } catch (err) {
      setError(err)
      setBusy(false)
    }
  }

  return (
    <AuthShell
      title="Sign in"
      subtitle="Welcome back to your grow dashboard."
      footer={
        <>
          New to PulsePonics?{' '}
          <Link to="/signup" className="font-semibold text-forest hover:underline">
            Create an account
          </Link>
        </>
      }
    >
      <form className="space-y-4" onSubmit={submit} noValidate>
        {error ? <Notice>{errorMessage(error)}</Notice> : null}
        <TextField label="Email" type="email" autoComplete="email" required value={email} onChange={(e) => setEmail(e.target.value)} />
        <TextField
          label="Password"
          type="password"
          autoComplete="current-password"
          required
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        <Button type="submit" className="w-full" loading={busy}>
          Sign in
        </Button>
      </form>
    </AuthShell>
  )
}

export function SignupPage() {
  useTitle('Create account')
  const { signup } = useAuth()
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState<unknown>(null)
  const [busy, setBusy] = useState(false)

  const mismatch = confirm.length > 0 && confirm !== password

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (password !== confirm) return
    setBusy(true)
    setError(null)
    try {
      await signup(email, password, name)
    } catch (err) {
      setError(err)
      setBusy(false)
    }
  }

  const general = error && !(error instanceof ApiError && error.field) ? errorMessage(error) : null

  return (
    <AuthShell
      title="Create your account"
      subtitle="Register your ESP32 devices and monitor them from anywhere."
      footer={
        <>
          Already have an account?{' '}
          <Link to="/login" className="font-semibold text-forest hover:underline">
            Sign in
          </Link>
        </>
      }
    >
      <form className="space-y-4" onSubmit={submit} noValidate>
        {general ? <Notice>{general}</Notice> : null}
        <TextField
          label="Name"
          autoComplete="name"
          placeholder="Optional"
          value={name}
          onChange={(e) => setName(e.target.value)}
          error={fieldError(error, 'name')}
        />
        <TextField
          label="Email"
          type="email"
          autoComplete="email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          error={fieldError(error, 'email')}
        />
        <TextField
          label="Password"
          type="password"
          autoComplete="new-password"
          required
          minLength={8}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          hint="At least 8 characters."
          error={fieldError(error, 'password')}
        />
        <TextField
          label="Confirm password"
          type="password"
          autoComplete="new-password"
          required
          value={confirm}
          onChange={(e) => setConfirm(e.target.value)}
          error={mismatch ? 'Passwords do not match' : null}
        />
        <Button type="submit" className="w-full" loading={busy} disabled={mismatch || !confirm}>
          Create account
        </Button>
      </form>
    </AuthShell>
  )
}
