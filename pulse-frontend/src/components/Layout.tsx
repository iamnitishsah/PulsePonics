import { Link, NavLink, Outlet } from 'react-router'
import { useAuth } from '../auth/useAuth'
import { useRealtime } from '../realtime/useRealtime'
import { Logo } from './Logo'

const STATUS = {
  live: { label: 'Live', dot: 'bg-ok shadow-[0_0_0_4px_rgb(19_115_78/0.15)]' },
  connecting: { label: 'Connecting', dot: 'bg-muted' },
  offline: { label: 'Reconnecting', dot: 'bg-danger' },
} as const

function navClass({ isActive }: { isActive: boolean }) {
  return `rounded-lg px-3 py-2 text-sm font-semibold transition-colors ${
    isActive ? 'bg-ink text-white' : 'text-ink-soft hover:bg-ink/5'
  }`
}

export function Layout() {
  const { user, logout } = useAuth()
  const { status } = useRealtime()
  const s = STATUS[status]

  return (
    <div className="min-h-screen">
      <header className="sticky top-0 z-10 border-b border-line bg-surface/90 backdrop-blur">
        <div className="mx-auto flex max-w-7xl flex-wrap items-center gap-x-4 gap-y-2 px-4 py-3 sm:px-6">
          <Link to="/" className="mr-2 flex items-center gap-2 font-extrabold text-ink">
            <Logo />
            <span>PulsePonics</span>
          </Link>
          <nav className="flex items-center gap-1" aria-label="Main">
            <NavLink to="/" end className={navClass}>
              Devices
            </NavLink>
            <NavLink to="/account" className={navClass}>
              Account
            </NavLink>
          </nav>
          <div className="ml-auto flex items-center gap-3">
            <span
              className="inline-flex items-center gap-2 rounded-full border border-line px-3 py-1 text-xs font-semibold text-ink-soft"
              title="Realtime connection to the server"
            >
              <span className={`size-2 rounded-full ${s.dot}`} />
              {s.label}
            </span>
            <span className="hidden max-w-40 truncate text-sm text-muted sm:inline" title={user?.email}>
              {user?.name}
            </span>
            <button
              type="button"
              onClick={() => void logout()}
              className="rounded-lg px-3 py-2 text-sm font-semibold text-ink-soft hover:bg-ink/5"
            >
              Sign out
            </button>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-7xl px-4 py-6 sm:px-6 sm:py-8">
        <Outlet />
      </main>
    </div>
  )
}
