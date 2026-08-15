# PulsePonics Backend

The backend is a Go service that stores hydroponics telemetry, accepts data over HTTP and MQTT, and broadcasts new readings to dashboards over WebSocket.

Full endpoint-level documentation lives in [API_DOCUMENTATION.md](API_DOCUMENTATION.md).

## Responsibilities

- Load runtime configuration from environment variables.
- Connect to MySQL.
- Validate incoming telemetry readings.
- Persist readings in the `readings` table.
- Serve REST endpoints for create, history, and latest-reading workflows.
- Subscribe to MQTT telemetry from hardware or simulator devices.
- Broadcast newly accepted readings to connected WebSocket clients.
- Shut down cleanly on `SIGINT` or `SIGTERM`.

## Tech Stack

- Go
- MySQL
- `sqlx` for database access
- `go-sql-driver/mysql` for MySQL connectivity
- Eclipse Paho MQTT client
- Gorilla WebSocket
- `godotenv` for local `.env` loading

## Directory Structure

```text
pulse-backend/
├── cmd/api/main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── features/
│   │   └── readings/
│   │       ├── errors.go
│   │       ├── http_handler.go
│   │       ├── model.go
│   │       ├── mqtt_subscriber.go
│   │       ├── repository.go
│   │       └── service.go
│   └── platform/
│       ├── database/
│       │   └── db.go
│       ├── httpmiddleware/
│       │   └── middleware.go
│       └── realtime/
│           ├── handler.go
│           └── hub.go
├── migrations/
│   └── 001_create_readings.sql
├── API_DOCUMENTATION.md
├── websocket_test.html
├── go.mod
└── README.md
```

## Architecture

The backend follows a vertical feature structure.

`internal/features/readings` owns the reading domain:

- JSON and database models
- validation rules
- service orchestration
- repository interface and MySQL implementation
- HTTP handlers
- MQTT subscriber adapter

`internal/platform` contains shared infrastructure:

- database connection setup
- HTTP middleware
- WebSocket hub and handler

`cmd/api/main.go` wires everything together.

## Runtime Flow

### REST Ingestion

```text
POST /readings
  -> HTTP handler decodes JSON
  -> service validates payload
  -> repository inserts row into MySQL
  -> service broadcasts Reading over WebSocket
  -> handler returns created ID
```

### MQTT Ingestion

```text
MQTT hydroponics/+/telemetry
  -> subscriber decodes JSON
  -> service validates payload
  -> repository inserts row into MySQL
  -> service broadcasts Reading over WebSocket
```

### Dashboard Reads

```text
GET /readings
  -> validates device_id
  -> defaults time window and limit when omitted
  -> queries MySQL by device and recorded_at range

GET /readings/latest
  -> validates device_id
  -> returns newest row for that device
```

## Prerequisites

- Go installed
- MySQL running
- MQTT broker running, for MQTT ingestion
- Database user with access to `pulseponics_db` and `readings` table

The module currently declares Go `1.26.5` in `go.mod`.

## Environment Variables

Create `pulse-backend/.env` for local development:

```env
DB_USER=pulseponics_admin
DB_PASSWORD=your_password
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=pulseponics_db
SERVER_PORT=8080
MQTT_BROKER_URL=tcp://localhost:1883
```

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `DB_USER` | Yes | None | MySQL username |
| `DB_PASSWORD` | Yes | None | MySQL password |
| `DB_HOST` | No | `127.0.0.1` | MySQL host |
| `DB_PORT` | No | `3306` | MySQL port |
| `DB_NAME` | Yes | None | MySQL database name |
| `SERVER_PORT` | No | `8080` | HTTP server port |
| `MQTT_BROKER_URL` | No | `tcp://localhost:1883` | MQTT broker URL |

## Database Setup

From `pulse-backend`:

```bash
mysql -u root -p < migrations/001_create_readings.sql
```

The migration creates:

- `pulseponics_db` database         
- `readings` table
- `(device_id, recorded_at)` index for dashboard history queries

## Run Locally

```bash
cd pulse-backend
go mod tidy
go run cmd/api/main.go
```

Expected startup logs include:

```text
connected to MySQL successfully
server listening on :8080
```

The exact log output may include symbols depending on the terminal.

## API Summary

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/readings` | Store a new reading |
| `GET` | `/readings` | Fetch historical readings for a device |
| `GET` | `/readings/latest` | Fetch newest reading for a device |
| `GET` | `/ws` | Subscribe to live readings over WebSocket |

See [API_DOCUMENTATION.md](API_DOCUMENTATION.md) for request bodies, response examples, validation rules, and curl examples.

## MQTT Contract

The backend subscribes to:

```text
hydroponics/+/telemetry
```

Payloads use the same JSON shape as `POST /readings`:

```json
{
  "device_id": "esp32-tank-01",
  "ph": 6.2,
  "ec_ms_cm": 1.8,
  "water_temp_c": 22.5,
  "air_temp_c": 26.1,
  "humidity_pct": 55.0,
  "pressure_hpa": 1012.5,
  "recorded_at": "2026-08-16T03:20:00+05:30"
}
```

## WebSocket Behavior

`GET /ws` upgrades the request to a WebSocket connection. Every newly accepted reading is broadcast as a JSON `Reading`.

Current behavior:

- push-only from server to client
- all readings are broadcast to every connected client
- clients should filter by `device_id`
- all origins are allowed for development

## Validation Rules

| Field | Rule |
| --- | --- |
| `device_id` | Required |
| sensor values | At least one sensor value required |
| `ph` | 0 to 14 |
| `ec_ms_cm` | Cannot be negative |
| `water_temp_c` | -10 to 60 Celsius |
| `air_temp_c` | -20 to 60 Celsius |
| `humidity_pct` | 0 to 100 |
| `pressure_hpa` | No current validation range |

## Manual Testing

Create a reading:

```bash
curl -X POST http://localhost:8080/readings \
  -H "Content-Type: application/json" \
  -d '{"device_id":"esp32-tank-01","ph":6.2,"ec_ms_cm":1.8}'
```

Fetch history:

```bash
curl "http://localhost:8080/readings?device_id=esp32-tank-01&limit=100"
```

Fetch latest:

```bash
curl "http://localhost:8080/readings/latest?device_id=esp32-tank-01"
```


## Known Limitations

- No authentication or authorization.
- Development WebSocket origin policy accepts all origins.
- No production CORS policy.
- No server-side WebSocket device filtering.
- No automated test suite yet.
- Invalid `from` and `to` query values silently fall back to defaults.

## Related Docs

- [Full API documentation](API_DOCUMENTATION.md)
- [Frontend README](../pulse-frontend/README.md)
- [Hardware README](../pulse-hardware/README.md)
