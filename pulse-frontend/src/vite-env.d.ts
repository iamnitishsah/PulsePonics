/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Backend origin, e.g. https://13-233-10-20.sslip.io. Empty = same origin (Vite dev proxy). */
  readonly VITE_API_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
