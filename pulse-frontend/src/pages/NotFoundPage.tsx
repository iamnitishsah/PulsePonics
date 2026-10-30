import { Link } from 'react-router'
import { buttonClass } from '../components/ui/buttonClass'
import { EmptyState } from '../components/ui/Notice'
import { useTitle } from '../hooks/useTitle'

export function NotFoundPage() {
  useTitle('Not found')
  return (
    <EmptyState
      title="Page not found"
      action={
        <Link to="/" className={buttonClass('secondary')}>
          Back to devices
        </Link>
      }
    >
      The page you're looking for doesn't exist.
    </EmptyState>
  )
}
