import { useState } from 'react'

/** Read-only value with a copy button (API keys, snippets). */
export function CopyField({ value, label, multiline = false }: { value: string; label: string; multiline?: boolean }) {
  const [copied, setCopied] = useState(false)

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1500)
    } catch {
      // Clipboard blocked: the value is still selectable.
    }
  }

  return (
    <div>
      <p className="mb-1.5 text-sm font-semibold text-ink">{label}</p>
      <div className="flex items-stretch gap-2">
        <pre
          className={`min-w-0 flex-1 overflow-x-auto rounded-lg border border-line bg-canvas px-3 py-2.5 font-mono text-xs text-ink ${
            multiline ? 'whitespace-pre' : 'whitespace-nowrap'
          }`}
        >
          {value}
        </pre>
        <button
          type="button"
          onClick={copy}
          className="shrink-0 rounded-lg border border-line bg-surface px-3 text-sm font-semibold text-ink hover:bg-canvas"
        >
          {copied ? 'Copied' : 'Copy'}
        </button>
      </div>
    </div>
  )
}
