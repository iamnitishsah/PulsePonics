# pulse-backend

Python service for PulsePonics built with **FastAPI**, **Pydantic v2**, **SQLAlchemy 2.0 (asyncio)**, and **MySQL**: user accounts, device registry, telemetry ingestion over MQTT and HTTP, time-series aggregation, per-device thresholds and alerts, and an async WebSocket feed.

The telemetry contract (topics, payload, ranges) comes from `../pulse-hardware/README.md`. The full HTTP reference is in [API_DOCUMENTATION.md](API_DOCUMENTATION.md).

## Modular Layout

```text
app/
├── core/
│   ├── config.py               # Pydantic Settings & environment variables
│   ├── errors.py               # AppError domain exceptions & unified HTTP handlers
│   ├── security.py             # Bcrypt with dummy hash, JWT HS256 tokens, SHA-256
│   └── middleware.py           # OriginPolicy (glob matching), logger, rate limiter
├── db/
│   ├── session.py              # Async MySQL connection pool (UTC pinned)
│   └── migrations.py           # Auto-applies migrations/*.sql at startup
├── schemas/
│   ├── auth.py                 # Pydantic auth schemas (User, Session, Signup, Login)
│   ├── devices.py              # Device schemas, API key validation
│   ├── readings.py             # Reading, Input, Point, History, Metric bounds
│   └── alerts.py               # Threshold, Alert, defaults, list filters
├── services/
│   ├── auth_service.py         # Passwords, token rotation, reuse detection
│   ├── device_service.py       # Device CRUD, ownership, key rotation
│   ├── reading_service.py      # Telemetry validation, downsampling bucket calculation
│   ├── alert_service.py        # 5% hysteresis calculation, open/resolve alert state
│   └── realtime_hub.py         # Per-user async WebSocket hub with slow-client eviction
├── mqtt/
│   └── subscriber.py           # Async MQTT consumer for hydroponics/+/telemetry
├── api/
│   ├── deps.py                 # FastAPI dependencies (auth, device key, db session)
│   └── routes/
│       ├── auth.py             # /api/auth/* and /api/me/*
│       ├── devices.py          # /api/devices/*
│       ├── readings.py         # /api/readings and /api/devices/{id}/readings/*
│       ├── alerts.py           # /api/alerts and /api/devices/{id}/thresholds
│       ├── health.py           # /api/healthz and /api/metrics
│       └── ws.py               # /api/ws
└── main.py                     # Lifespan, middleware registration, router mounting
migrations/                     # Preserved SQL migrations (001_init.sql)
tests/                          # Pytest async test suite
requirements.txt                # Pinned production dependencies
pyproject.toml                  # Python package configuration
Dockerfile                      # Python 3.12 slim container
```

## Running

### With Docker (recommended)

From the repository root:

```bash
docker compose up -d --build
```

### Outside Docker

Ensure MySQL and Mosquitto are running (e.g. via Compose). Create `pulse-backend/.env`:

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
SERVER_PORT=8081                # 8080 is taken if container is active
```

Install dependencies and start Uvicorn:

```bash
cd pulse-backend
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
python -m uvicorn app.main:app --host 0.0.0.0 --port 8081 --reload
```

## Interactive API Documentation

FastAPI automatically generates interactive OpenAPI documentation:
- Swagger UI: `http://localhost:8080/docs`
- ReDoc: `http://localhost:8080/redoc`

## Tests

```bash
pytest tests/                   # Run all async unit tests
../scripts/smoke-test.sh        # End-to-end smoke test against running stack
```
