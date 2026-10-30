import { useEffect, useId, useRef, type ReactNode } from 'react'

/** Accessible dialog built on the native <dialog> element (focus trap, Esc, backdrop). */
export function Modal({
  open,
  title,
  onClose,
  children,
  footer,
}: {
  open: boolean
  title: string
  onClose: () => void
  children: ReactNode
  footer?: ReactNode
}) {
  const ref = useRef<HTMLDialogElement>(null)
  const titleId = useId()

  useEffect(() => {
    const dialog = ref.current
    if (!dialog) return
    if (open && !dialog.open) dialog.showModal()
    if (!open && dialog.open) dialog.close()
  }, [open])

  return (
    <dialog
      ref={ref}
      aria-labelledby={titleId}
      onCancel={(event) => {
        event.preventDefault()
        onClose()
      }}
      onClick={(event) => {
        if (event.target === ref.current) onClose() // backdrop click
      }}
      className="m-auto w-[calc(100%-2rem)] max-w-lg rounded-xl border border-line bg-surface p-0 text-ink shadow-2xl backdrop:bg-ink/40"
    >
      <div className="px-5 pt-5">
        <h2 id={titleId} className="text-lg font-bold">
          {title}
        </h2>
      </div>
      <div className="px-5 py-4">{children}</div>
      {footer ? <div className="flex flex-wrap justify-end gap-2 border-t border-line px-5 py-4">{footer}</div> : null}
    </dialog>
  )
}
