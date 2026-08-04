# Hydroponics Telemetry Backend

Automated real-time hydroponics monitoring backend — Go + MySQL.

## Prerequisites

- Go 1.22+
- MySQL running locally
- A `.env` file at the project root (see below)

## Setup

1. Create the database:
   ```bash
   mysql -u root -p < migrations/001_create_readings.sql
   ```

2. Create `.env` at the project root:
   ```
   DB_USER=hydrodb_admin
   DB_PASSWORD=your_password
   DB_HOST=127.0.0.1
   DB_PORT=3306
   DB_NAME=hydrodb
   SERVER_PORT=8080
   ```

3. Install dependencies:
   ```bash
   go mod tidy
   ```

4. Run:
   ```bash
   go run cmd/api/main.go
   ```

You should see:
```
✅ connected to MySQL successfully
🚀 server listening on :8080
```

Press `Ctrl+C` to stop — the server drains in-flight requests before shutting down.

## Architecture

```
cmd/api/main.go        — entrypoint, wires everything together
internal/config        — env var loading
internal/domain        — data models (Reading, NewReadingInput)
internal/repository    — MySQL access via sqlx (interface + impl)
internal/service        — validation & business logic
internal/handler        — HTTP handlers + logging middleware
migrations/             — raw SQL schema files
```

Each layer only depends on the layer directly below it. The repository layer
is defined as an interface so the service layer never depends on MySQL
directly — this matters for testing and for future extensibility (e.g. this
is also the reason it'll be easy to add MQTT ingestion later, since MQTT
messages will flow through this same service layer).

## API Reference

### `POST /readings`
Submit a new sensor reading.

**Body:**
```json
{
  "device_id": "esp32-tank-01",
  "ph": 6.2,
  "ec_ms_cm": 1.8,
  "water_temp_c": 22.5,
  "air_temp_c": 26.1,
  "humidity_pct": 55.0,
  "pressure_hpa": 1012.5,
  "recorded_at": "2026-07-31T17:48:40+05:30"
}
```
All sensor fields are optional individually, but at least one must be present.
`recorded_at` is optional — defaults to server time if omitted.

**Response:** `201 Created`
```json
{ "id": 1 }
```

### `GET /readings?device_id=X&from=RFC3339&to=RFC3339&limit=N&offset=N`
Fetch historical readings for a device. Only `device_id` is required —
defaults to last 24 hours, limit 100.

### `GET /readings/latest?device_id=X`
Fetch the most recent reading for a device. Returns `404` if none exist.

## Validation rules

| Field | Rule |
|---|---|
| `device_id` | required |
| at least one sensor value | required |
| `ph` | 0–14 |
| `ec_ms_cm` | ≥ 0 |
| `water_temp_c` | -10 to 60 °C |
| `air_temp_c` | -20 to 60 °C |
| `humidity_pct` | 0–100 |

These are sanity bounds to catch bad sensor data, not "safe growing range"
thresholds — the alerting engine (planned) will handle biological safety
thresholds separately, on top of already-stored data.

## Status

- [x] Phase 1: Core REST API (config, DB, repository, service, handlers)
- [ ] Phase 2: MQTT ingestion + simulated publisher
- [ ] Phase 3: WebSocket real-time push
- [ ] Phase 4: Alerting engine
- [ ] Phase 5: Frontend dashboard