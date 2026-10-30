# PulsePonics

**Automated real-time hydroponics telemetry and alerting.** An ESP32 reads pH, EC, water temperature and air conditions and publishes them over MQTT. A Go backend authenticates devices, stores every reading in MySQL, checks per-device alert thresholds and pushes updates over WebSocket. A React dashboard shows each user their own devices live.

```text
ESP32 (DS18B20, BME280, ADS1115 + pH/EC)            React dashboard (Vercel)
        │ MQTT 1883, username/password                    │  HTTPS + WSS
        ▼                                                 ▼
   Mosquitto ──► Go backend ──► MySQL               Caddy (auto-HTTPS)
                    │  thresholds → alerts                │
                    └──── WebSocket push ◄────────────────┘
        └─────────────── docker compose (EC2 or local) ───────────────┘
```

## Repository layout

| Path | What it is |
|---|---|
| `pulse-hardware/` | ESP32 firmware (PlatformIO). Its README defines the MQTT contract. |
| `pulse-backend/` | Go API: auth, devices, ingestion (MQTT + HTTP), thresholds, alerts, WebSocket. [API reference](pulse-backend/API_DOCUMENTATION.md) |
| `pulse-frontend/` | React 19 + TypeScript + Tailwind + Recharts dashboard. Deploys to Vercel. |
| `pulse-simulator/` | Stand-in for the ESP32, speaking the same MQTT contract. |
| `deploy/` | Mosquitto config/entrypoint and the Caddyfile. |
| `docker-compose.yml` | MySQL, Mosquitto, backend, plus optional Caddy (`prod`) and simulator (`sim`). |
| `scripts/smoke-test.sh` | 38 end-to-end checks against a running stack. |

## Features

- **Accounts:** email + password (bcrypt), 15-minute JWT access tokens, rotating 7-day refresh tokens with reuse detection, change password (signs out other sessions), delete account.
- **Devices:** each user registers their ESP32s by `DEVICE_ID`. One device belongs to one user; users only ever see their own data. Telemetry from unregistered IDs is dropped.
- **Ingestion:** MQTT (`hydroponics/{device_id}/telemetry`, QoS 1) exactly as the firmware sends it, plus `POST /api/readings` with a per-device API key for testing. Duplicate retries (same `recorded_at`) are stored once.
- **Alerts:** per-device min/max for every metric, editable in the dashboard, with sensible defaults. Alerts open when a reading leaves the band and resolve with hysteresis, and can be acknowledged. Pushed live.
- **Dashboard:** live device list, per-device cards with threshold state, charts with 1h/6h/24h/7d/30d ranges (averaged server-side for long ranges), outage gaps, threshold bands, recent readings, alerts.

## Quick start (local)

Requirements: Docker Desktop, Node 22+ (for the frontend).

```bash
cp .env.example .env            # then replace every change-me (openssl rand -hex 32)
docker compose up -d --build    # MySQL :3307, Mosquitto :1883, backend :8080
curl localhost:8080/api/healthz # {"status":"ok"}

cd pulse-frontend
npm install
npm run dev                     # http://localhost:5173 (proxies /api to :8080)
```

Then sign up in the dashboard, and register a device:

- **Real ESP32:** register its `DEVICE_ID` (default `esp32-hydro-01`), then set `MQTT_HOST` (your Mac's IP), `MQTT_USER` and `MQTT_PASSWORD` (the `MQTT_DEVICE_*` values from `.env`) in `pulse-hardware/include/secrets.h` and flash it.
- **No hardware:** register `esp32-sim-01`, then `docker compose --profile sim up -d simulator`.

Verify everything end to end at any time:

```bash
./scripts/smoke-test.sh
```

## Deploying

The backend stack goes on one EC2 instance; the frontend goes on Vercel.

### EC2 (backend, database, broker)

1. Launch Ubuntu (t3.micro is enough; MySQL is tuned for 1 GB of RAM), attach an **Elastic IP**, and install Docker.
2. Security group inbound: **22** (SSH, your IP), **80** and **443** (HTTPS, Let's Encrypt), **1883** (MQTT for the ESP32). Do not open 3306 or 8080.
3. Copy the repo and create `.env` from `.env.example` with strong secrets, plus:
   ```env
   API_DOMAIN=13-233-10-20.sslip.io          # your Elastic IP, dots → dashes
   CORS_ALLOWED_ORIGINS=https://<your-app>.vercel.app,https://<your-app>-*.vercel.app
   TRUST_PROXY_HEADERS=true
   ```
   [sslip.io](https://sslip.io) resolves that name to the IP with no signup. Let's Encrypt won't issue certificates for `*.amazonaws.com` names, and an HTTPS Vercel page can't call a plain-HTTP API.
4. Start it:
   ```bash
   docker compose --profile prod up -d --build
   curl https://$API_DOMAIN/api/healthz
   ```
   Caddy obtains and renews the certificate automatically.

For a CI pipeline, build `pulse-backend/` into an image, push it to a registry, and set `BACKEND_IMAGE` in `.env` so the server runs `docker compose pull && docker compose --profile prod up -d`.

### Vercel (frontend)

Import the repository, set **Root Directory** to `pulse-frontend`, and add the environment variable `VITE_API_URL=https://<API_DOMAIN>`. `vercel.json` already handles client-side routes.

### ESP32 against EC2

In `secrets.h`, set `MQTT_HOST` to the Elastic IP (or the sslip.io name) and keep `MQTT_PORT 1883`.

## Testing

| Suite | Command |
|---|---|
| Backend unit tests | `cd pulse-backend && go test -race ./...` |
| Frontend unit tests | `cd pulse-frontend && npm test` |
| Firmware math tests | `cd pulse-hardware && pio test -e native` |
| End-to-end (running stack) | `./scripts/smoke-test.sh` |

## Status

| Area | Status |
|---|---|
| Docker stack (MySQL, Mosquitto with auth, backend, Caddy HTTPS, simulator) | Done |
| Backend: auth, devices, MQTT + HTTP ingestion, history, thresholds, alerts, WebSocket | Done |
| Frontend: auth, devices, live dashboard, charts, thresholds, alerts, account | Done |
| Firmware: DS18B20, Wi-Fi, NTP, MQTT with credentials | Done |
| Firmware: BME280, ADS1115, pH/EC calibration | Planned (see `pulse-hardware/README.md`) |
| External notifications (Telegram / email) | Planned |
| MQTT over TLS (8883) | Planned |

## Security notes

This is a student-project deployment, sized for a demo:

- MQTT uses username/password over plain TCP on 1883. All devices share one login; the backend enforces device ownership and topic/payload consistency.
- Refresh tokens live in `localStorage`, because the frontend and API are on different sites. Access tokens stay in memory only.
- Never commit `.env` or `pulse-hardware/include/secrets.h` (both are gitignored). Submit code with `git archive`, not a zip of the working folder.
