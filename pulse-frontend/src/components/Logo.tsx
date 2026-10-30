/** A leaf over a water line. */
export function Logo({ className = 'size-7' }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 32 32" aria-hidden="true">
      <rect width="32" height="32" rx="8" fill="#17211d" />
      <path d="M16 6c5 4 6.5 8.5 4.2 12.2A5 5 0 0 1 16 21a5 5 0 0 1-4.2-2.8C9.5 14.5 11 10 16 6Z" fill="#8ed4b8" />
      <path d="M16 11v10" stroke="#17211d" strokeWidth="1.6" strokeLinecap="round" />
      <path d="M7 25c3-1.6 6-1.6 9 0s6 1.6 9 0" stroke="#8ed4b8" strokeWidth="2" fill="none" strokeLinecap="round" />
    </svg>
  )
}
