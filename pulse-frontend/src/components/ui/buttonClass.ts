export type Variant = 'primary' | 'secondary' | 'ghost' | 'danger'
export type Size = 'sm' | 'md'

const variants: Record<Variant, string> = {
  primary: 'bg-forest text-white hover:bg-forest-dark disabled:bg-forest/60',
  secondary: 'border border-line bg-surface text-ink hover:bg-canvas disabled:text-muted',
  ghost: 'text-ink-soft hover:bg-ink/5 disabled:text-muted',
  danger: 'bg-danger text-white hover:bg-danger/90 disabled:bg-danger/60',
}

const sizes: Record<Size, string> = {
  sm: 'h-9 px-3 text-sm',
  md: 'h-11 px-4 text-sm',
}

/** Button styling, also used on links: <Link className={buttonClass('secondary')}>. */
export function buttonClass(variant: Variant = 'primary', size: Size = 'md') {
  return `inline-flex items-center justify-center gap-2 rounded-lg font-semibold transition-colors ${variants[variant]} ${sizes[size]}`
}
