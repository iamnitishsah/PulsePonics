# pulse-backend

Go service for PulsePonics: user accounts, device registry, telemetry ingestion over MQTT and HTTP, history queries, per-device thresholds and alerts, and a WebSocket feed.

The telemetry contract (topics, payload, ranges) comes from `../pulse-hardware/README.md`. The full HTTP reference is in [API_DOCUMENTATION.md](API_DOCUMENTATION.md).

## Layout

```text
cmd/api/main.go                 wiring, HTTP server, graceful shutdown
migrations/                     embedded SQL, applied automatically at startup
internal/
  config/                       environment variables
  features/
    auth/                       signup/login, JWT access tokens, rotating refresh tokens
    devices/                    device registry, ownership checks, API keys
    readings/                   validation, storage, history/aggregation, MQTT subscriber
    alerts/                     thresholds (defaults + overrides), alert open/resolve logic
  platform/
    apperr/                     typed errors → HTTP status codes
    httpx/                      JSON helpers, logging, recovery, CORS, rate limiting
    database/                   MySQL connection (UTC) and migration runner
    realtime/                   WebSocket hub: per-user fan-out, ping/pong, slow-client eviction
    authctx/                    user ID in the request context
```

## Running

With Docker (recommended), from the repository root:

```bash
docker compose up -d --build
```

Outside Docker (MySQL and Mosquitto still running in Compose), create `pulse-backend/.env`:

```env
DB_HOST=127.0.0.1
DB_PORT=3307
DB_NAME=pulseponics
DB_USER=pulse
DB_PASSWORD=...                 # from ../.env
MQTT_BROKER_URL=tcp://localhost:1883
MQTT_USERNAME=pulse-backend
MQTT_PASSWORD=...               # MQTT_BACKEND_PASSWORD from ../.env
JWT_SECRET=...                  # at least 32 characters
SERVER_PORT=8081                # 8080 is taken by the container
```

```bash
go run ./cmd/api
```

### Environment

| Variable | Default | Notes |
|---|---|---|
| `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD` | `127.0.0.1`, `3306`, required, required, required | MySQL 8. The session time zone is forced to UTC. |
| `MQTT_BROKER_URL` | `tcp://localhost:1883` | |
| `MQTT_USERNAME`, `MQTT_PASSWORD` | required | The backend's broker account (read `hydroponics/#`). |
| `MQTT_CLIENT_ID` | `pulse-backend-<random>` | |
| `JWT_SECRET` | required | At least 32 characters. Changing it signs everyone out. |
| `ACCESS_TOKEN_TTL`, `REFRESH_TOKEN_TTL` | `15m`, `168h` | Go durations. |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173` | Comma-separated; `*` wildcards allowed (`https://app-*.vercel.app`). Also used for WebSocket origins. |
| `TRUST_PROXY_HEADERS` | `false` | `true` behind Caddy, so rate limiting sees real client IPs. |
| `SERVER_PORT` | `8080` | |

## Design notes

- **Ingestion path.** MQTT and `POST /api/readings` both call `readings.Service.Ingest`. It validates the reading, drops unregistered devices, inserts it, pushes it to the owner's WebSockets, then evaluates alerts. An alert failure never loses the reading.
- **Contract.** Accepted ranges match the firmware's `include/config.h` (pH 0–14, EC 0–20, water −10–60 °C, air −20–60 °C, humidity 0–100 %, pressure 300–1100 hPa). The firmware sends anything outside these as `null`, so real devices are never rejected. A `recorded_at` more than 5 minutes in the future is rejected (bad device clock).
- **Duplicates.** `UNIQUE (device_id, recorded_at)` stores a firmware PUBACK retry once and logs the duplicate (hardware README, Known issue 4).
- **MQTT resilience.** The subscriber subscribes in `OnConnect`, so ingestion resumes after a broker restart. It retries the initial connection instead of exiting.
- **Topic ownership.** A message's payload `device_id` must equal the topic segment.
- **History.** Ranges up to 31 days. If a range holds more readings than `max_points` (default 500), readings are averaged into epoch-aligned buckets of a "nice" size (1, 5, 15 min…) chosen to stay within the limit.
- **Alerts.** One open alert per device and metric, enforced by a unique key on a generated column. An alert resolves only once the value is back inside the band by 5 % of its width, so sensor noise at a limit doesn't flap. Null sensor values leave alert state unchanged. Changing thresholds re-evaluates the latest reading immediately.
- **Refresh tokens.** Stored as SHA-256 hashes and rotated on every refresh. Presenting an already-rotated token revokes all of that user's sessions.
- **WebSocket.** Each client has a buffered queue and its own writer goroutine. A client that can't keep up is disconnected rather than blocking ingestion. Ping/pong detects dead connections.

## Tests

```bash
go test -race ./...           # unit tests (no database needed)
../scripts/smoke-test.sh      # end to end against the running Docker stack
```
