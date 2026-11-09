# PulsePonics

**Automated real-time hydroponics telemetry and alerting.** An ESP32 reads pH, EC, water temperature and air conditions and publishes them over MQTT. A Python FastAPI backend authenticates devices, stores every reading in MySQL, checks per-device alert thresholds and pushes updates over WebSocket. A React dashboard shows each user their own devices live.

```text
ESP32 (DS18B20, BME280, ADS1115 + pH/EC)            React dashboard (Vercel)
        │ MQTT 1883, username/password                    │  HTTPS + WSS
        ▼                                                 ▼
   Mosquitto ──► FastAPI backend ──► MySQL          Caddy (auto-HTTPS)
                     │  thresholds → alerts               │
                     └──── WebSocket push ◄───────────────┘
        └─────────────── docker compose (EC2 or local) ───────────────┘
```

## Repository layout

| Path | What it is |
|---|---|
| `pulse-hardware/` | ESP32 firmware (PlatformIO / C++). Its README defines the MQTT contract. |
| `pulse-backend/` | FastAPI (Python 3.12) API: auth, devices, ingestion (MQTT + HTTP), thresholds, alerts, WebSocket. Interactive OpenAPI docs at `/docs`. [API reference](pulse-backend/API_DOCUMENTATION.md) |
| `pulse-frontend/` | React 19 + TypeScript + Tailwind CSS 4 + Recharts dashboard. Deploys to Vercel. |
| `pulse-simulator/` | Stand-in for the ESP32 (Go), speaking the exact same MQTT contract. |
| `deploy/` | Mosquitto config/entrypoint and the Caddyfile. |
| `docker-compose.yml` | MySQL 8.4, Mosquitto, FastAPI backend, plus optional Caddy (`prod`) and simulator (`sim`). |
| `scripts/smoke-test.sh` | 38 end-to-end checks against a running stack. |

## Features

- **Accounts:** email + password (bcrypt with dummy hash for timing attack prevention), 15-minute JWT access tokens, rotating 7-day refresh tokens with reuse detection, change password (signs out other sessions), delete account.
- **Devices:** each user registers their ESP32s by `DEVICE_ID`. One device belongs to one user; users only ever see their own data. Telemetry from unregistered IDs is dropped.
- **Ingestion:** MQTT (`hydroponics/{device_id}/telemetry`, QoS 1) exactly as the firmware sends it, plus `POST /api/readings` with a per-device API key for testing. Duplicate retries (same `recorded_at`) are stored once.
- **Alerts:** per-device min/max for every metric, editable in the dashboard, with sensible defaults. Alerts open when a reading leaves the band and resolve with 5% hysteresis deadband, and can be acknowledged. Pushed live over WebSocket.
- **Dashboard:** live device list, per-device cards with threshold state, charts with 1h/6h/24h/7d/30d ranges (dynamically downsampled server-side for long ranges), outage gaps, threshold bands, recent readings, alerts.

## Quick start (local)

Requirements: Docker Desktop, Node 22+ (for the frontend), Python 3.12+ (if running backend locally).

```bash
cp .env.example .env            # then replace every change-me (openssl rand -hex 32)
docker compose up -d --build    # MySQL :3307, Mosquitto :1883, backend :8080
curl localhost:8080/api/healthz # {"status":"ok"}

cd pulse-frontend
npm install
npm run dev                     # http://localhost:5173 (proxies /api to :8080)
```

Interactive API documentation is immediately available locally at:
- Swagger UI: `http://localhost:8080/docs`
- ReDoc: `http://localhost:8080/redoc`

Then sign up in the dashboard, and register a device:

- **Real ESP32:** register its `DEVICE_ID` (default `esp32-hydro-01`), then set `MQTT_HOST` (your Mac's IP or EC2 IP), `MQTT_USER` and `MQTT_PASSWORD` (the `MQTT_DEVICE_*` values from `.env`) in `pulse-hardware/include/secrets.h` and flash it.
- **No hardware:** register `esp32-sim-01`, then `docker compose --profile sim up -d simulator`.

Verify everything end to end at any time:

```bash
./scripts/smoke-test.sh
```

## Deploying

The backend stack is deployed on AWS EC2; the frontend is deployed on Vercel.

### Live Deployment Configuration

| Component | Target / Host | Details |
|---|---|---|
| **EC2 Public IPv4** | `3.26.224.103` | Ubuntu instance running Docker Compose (`prod` profile) |
| **API Domain** | `3-26-224-103.sslip.io` | Automatic SSL certificate provisioned by Caddy via Let's Encrypt |
| **API Base URL** | `https://3-26-224-103.sslip.io` | Reverse proxied to backend container (:8080) |
| **Swagger UI** | `https://3-26-224-103.sslip.io/docs` | Interactive OpenAPI documentation |
| **MQTT Broker** | `3.26.224.103:1883` | Eclipse Mosquitto with credential authentication & ACLs |
| **Web Dashboard** | Vercel | React 19 SPA configured with `VITE_API_URL=https://3-26-224-103.sslip.io` |

### EC2 (backend, database, broker)

1. Launch Ubuntu (t3.micro is enough; MySQL is tuned for 1 GB of RAM), attach an **Elastic IP** (`3.26.224.103`), and install Docker.
2. Security group inbound: **22** (SSH, your IP), **80** and **443** (HTTPS, Let's Encrypt), **1883** (MQTT for the ESP32). Do not open 3306 or 8080 to the internet.
3. Copy the repo and create `.env` from `.env.example` with strong secrets:
   ```env
   API_DOMAIN=3-26-224-103.sslip.io          # your Elastic IP, dots → dashes
   CORS_ALLOWED_ORIGINS=https://<your-app>.vercel.app,https://<your-app>-*.vercel.app
   TRUST_PROXY_HEADERS=true
   ```
   [sslip.io](https://sslip.io) resolves `3-26-224-103.sslip.io` directly to `3.26.224.103` without separate DNS configuration.
4. Start it:
   ```bash
   docker compose --profile prod up -d --build
   curl https://3-26-224-103.sslip.io/api/healthz
   ```
   Caddy obtains and renews the certificate automatically.

Deployments are automated via GitHub Actions (`.github/workflows/deploy.yml`): every push to `main` executes unit tests, SSHes into the EC2 instance, rebuilds the stack, runs a health check polling loop, and prunes old images.

### Vercel (frontend)

Import the repository, set **Root Directory** to `pulse-frontend`, and set environment variable:
```env
VITE_API_URL=https://3-26-224-103.sslip.io
```
`vercel.json` handles client-side routes.

### ESP32 against EC2

In `pulse-hardware/include/secrets.h`:
```cpp
#define MQTT_HOST     "3.26.224.103"       // or "3-26-224-103.sslip.io"
#define MQTT_PORT     1883
#define MQTT_USER     "pulse-device"       // from .env
#define MQTT_PASSWORD "<MQTT_DEVICE_PASSWORD>"
```

## Testing

| Suite | Command | Notes |
|---|---|---|
| Backend unit tests | `cd pulse-backend && pytest tests/ -v` | Pytest async tests for auth, alerts, readings, middleware |
| Frontend unit tests | `cd pulse-frontend && npm test` | Vitest: metrics formatting, status classifier, outage gaps |
| Firmware math tests | `cd pulse-hardware && pio test -e native` | Unity tests: Nernst pH compensation, EC temp math |
| End-to-end (running stack) | `./scripts/smoke-test.sh` | 38 integration assertions against running API and broker |

## Status

| Area | Status |
|---|---|
| Cloud Stack (EC2 `3.26.224.103`, Caddy HTTPS, Mosquitto, MySQL 8.4, FastAPI) | Live / Done |
| Backend: FastAPI (Python 3.12), SQLAlchemy 2.0 Asyncio, auth, devices, MQTT + HTTP, thresholds, alerts, WebSocket | Done |
| Frontend: React 19, TypeScript, Tailwind 4, Recharts, live WebSocket sync, Vercel | Done |
| Firmware: DS18B20, Wi-Fi, NTP, MQTT QoS 1 with credentials | Done |
| Firmware: BME280, ADS1115, pH/EC calibration | Planned (see `pulse-hardware/README.md`) |
| External notifications (Telegram / email) | Planned |
| MQTT over TLS (8883) | Planned |

## Security notes

This is a student-project deployment, sized for a demo:

- MQTT uses username/password over plain TCP on 1883. All devices share one login; the backend enforces device ownership and topic/payload consistency.
- Refresh tokens live in `localStorage`, because the frontend and API are on different sites. Access tokens stay in memory only.
- Never commit `.env`, `pulse-hardware/include/secrets.h`, or SSH keys (`*.pem`). Submit code with `git archive`, not a zip of the working folder.
