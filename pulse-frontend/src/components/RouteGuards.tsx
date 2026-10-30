import { Navigate, Outlet, useLocation } from 'react-router'
import { useAuth } from '../auth/useAuth'
import { RealtimeProvider } from '../realtime/RealtimeProvider'
import { Button } from './ui/Button'
import { PageSpinner } from './ui/Spinner'

function Unreachable({ retry }: { retry: () => void }) {
  return (
    <div className="grid min-h-screen place-items-center px-4">
      <div className="max-w-sm text-center">
        <h1 className="text-xl font-bold">Can't reach the server</h1>
        <p className="mt-2 text-sm text-muted">
          PulsePonics couldn't contact the backend to restore your session. Check that it is running and try again.
        </p>
        <Button className="mt-5" onClick={retry}>
          Try again
        </Button>
      </div>
    </div>
  )
}

/** Signed-in area: redirects to /login otherwise and opens the realtime connection. */
export function RequireAuth() {
  const { status, retry } = useAuth()
  const location = useLocation()

  if (status === 'loading') return <PageSpinner label="Restoring your session" />
  if (status === 'unreachable') return <Unreachable retry={retry} />
  if (status === 'anonymous') return <Navigate to="/login" replace state={{ from: location }} />

  return (
    <RealtimeProvider>
      <Outlet />
    </RealtimeProvider>
  )
}

/** Login/signup: signed-in users go straight to where they were heading. */
export function PublicOnly() {
  const { status } = useAuth()
  const location = useLocation()
  const from = (location.state as { from?: { pathname: string } } | null)?.from?.pathname ?? '/'

  if (status === 'loading') return <PageSpinner label="Restoring your session" />
  if (status === 'authenticated') return <Navigate to={from} replace />
  return <Outlet />
}
