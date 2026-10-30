# pulse-frontend

The PulsePonics dashboard: React 19, TypeScript, Vite, Tailwind CSS 4, React Router and Recharts.

## Pages

| Route | What it shows |
|---|---|
| `/login`, `/signup` | Account access |
| `/` | Your devices: live values, online/offline state, open alert count, and registration of new devices (the API key is shown once) |
| `/devices/:deviceId` | Metric cards coloured by threshold state, a history chart (1H–30D, threshold band, outage gaps, min/avg/max), alerts with acknowledge, recent readings |
| `/devices/:deviceId/settings` | Threshold editor, rename, API key rotation, delete |
| `/account` | Profile, change password, sign out, delete account |

Everything updates live over one WebSocket per tab. After a reconnect, pages refetch whatever they missed.

## Development

```bash
npm install
npm run dev        # http://localhost:5173
```

The dev server proxies `/api` (including the WebSocket) to `http://localhost:8080`, the Docker backend. Start it from the repo root with `docker compose up -d`. To point at another backend locally, set `VITE_PROXY_TARGET`.

```bash
npm run lint
npm test           # vitest: metric state, formatting, chart gap detection
npm run build      # type-check + production build into dist/
```

## Deploying to Vercel

1. Import the repository and set **Root Directory** to `pulse-frontend`. The Vite preset is detected automatically.
2. Add the environment variable `VITE_API_URL=https://<API_DOMAIN>` (the EC2 backend, no trailing slash).
3. On the backend, add the Vercel URL to `CORS_ALLOWED_ORIGINS`, e.g. `https://pulseponics.vercel.app,https://pulseponics-*.vercel.app` for preview deployments.

`vercel.json` rewrites every path to `index.html`, so deep links like `/devices/esp32-hydro-01` work on reload.

## How auth works

- The access token (15 min) is held in memory only. The refresh token is in `localStorage` (`src/lib/session.ts`), because the API is on another site and third-party cookies are unreliable.
- `src/lib/api.ts` attaches the token, refreshes it when it is about to expire or a request returns 401, and retries the request once.
- Refreshes are serialised across tabs with the Web Locks API. The backend rotates refresh tokens and treats reuse as theft, so two tabs must never refresh with the same token.
- Signing out in one tab signs out the others.

## Layout

```text
src/
  lib/          api client, session/token store, config, metric metadata, formatting, chart helpers
  auth/         AuthProvider + useAuth
  realtime/     RealtimeProvider (WebSocket with backoff) + useRealtimeEvents / useOnReconnect
  hooks/        useQuery (abortable loads), useNow, useTitle
  components/   layout, guards, cards, chart, alerts, tables, dialogs, ui/ primitives
  pages/        one file per route
```
