# PulsePonics

PulsePonics is a smart hydroponics monitoring system that collects nutrient and environment telemetry from an ESP32-style device, stores it in a Go backend, streams updates in real time, and displays the system state in a React dashboard.

The project is designed as an end-to-end product prototype: hardware or simulator publishes telemetry, the backend validates and persists it, and the frontend gives a grower a live operational view of the hydroponic setup.

## Product Idea

Hydroponic systems can produce fast, high-quality plant growth, but they depend on narrow operating ranges for water chemistry and environment. A grower needs to know whether pH, EC, temperature, humidity, and pressure are stable without manually checking sensors throughout the day.

PulsePonics solves the first stage of that problem:

- collect sensor data from a tank or grow module
- store every reading with device and timestamp context
- show current conditions and recent trends
- stream new telemetry to the dashboard as it arrives
- keep the architecture ready for alerting, control loops, and multi-device expansion

## Current Capabilities

- REST API for creating readings and querying history/latest values
- MQTT ingestion for hardware or simulated device telemetry
- WebSocket broadcast for real-time dashboard updates
- MySQL persistence with a dashboard-oriented `(device_id, recorded_at)` index
- React + TypeScript frontend using Tailwind CSS
- Real-time frontend data flow: REST bootstrap plus WebSocket append
- Standalone hardware simulator that publishes realistic MQTT telemetry

## System Architecture

```text
ESP32 / Simulator
        |
        | MQTT: hydroponics/{device}/telemetry
        v
MQTT Broker
        |
        v
Go Backend  <---- REST reads ---- React Dashboard
        |                         ^
        | WebSocket push          |
        +-------------------------+
        |
        v
MySQL readings table
```

## Repository Layout

```text
.
├── pulse-backend/      Go API, MySQL persistence, MQTT subscriber, WebSocket hub
├── pulse-frontend/     React, TypeScript, Vite, Tailwind dashboard
├── pulse-hardware/     Hardware-facing simulator and future firmware workspace
├── README.md           Project overview
└── LICENSE
```

## Data Collected

Each reading belongs to a `device_id` and may include:

| Field | Meaning |
| --- | --- |
| `ph` | Nutrient solution pH |
| `ec_ms_cm` | Electrical conductivity in mS/cm |
| `water_temp_c` | Water temperature in Celsius |
| `air_temp_c` | Air temperature in Celsius |
| `humidity_pct` | Relative humidity percentage |
| `pressure_hpa` | Atmospheric pressure in hPa |
| `recorded_at` | Time the device captured the reading |

Sensor fields are nullable so one failed or missing sensor does not prevent storing the rest of the payload.

## Quick Start

### 1. Start MySQL and Create the Database

From `pulse-backend`:

```bash
mysql -u root -p < migrations/001_create_readings.sql
```

### 2. Configure Backend Environment

Create `pulse-backend/.env`:

```env
DB_USER=hydrodb_admin
DB_PASSWORD=your_password
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=hydrodb
SERVER_PORT=8080
MQTT_BROKER_URL=tcp://localhost:1883
```

### 3. Start an MQTT Broker

Use any MQTT broker reachable by the backend. For local development, Mosquitto on `localhost:1883` works well.

### 4. Run the Backend

```bash
cd pulse-backend
go run cmd/api/main.go
```

The backend listens on `http://localhost:8080` by default.

### 5. Run the Frontend

```bash
cd pulse-frontend
npm install
npm run dev
```

Open the Vite URL shown in the terminal, usually:

```text
http://localhost:5173
```

### 6. Publish Simulated Hardware Data

In another terminal:

```bash
cd pulse-hardware/simulator
go run . -device esp32-tank-01 -interval 10s
```

The frontend defaults to `esp32-tank-01`, so simulated readings should appear automatically once the backend, MQTT broker, and frontend are running.

## Documentation

- Backend details: [pulse-backend/README.md](pulse-backend/README.md)
- Full API reference: [pulse-backend/API_DOCUMENTATION.md](pulse-backend/API_DOCUMENTATION.md)
- Frontend details: [pulse-frontend/README.md](pulse-frontend/README.md)
- Hardware and simulator details: [pulse-hardware/README.md](pulse-hardware/README.md)

## Development Status

| Area | Status |
| --- | --- |
| Backend REST API | Implemented |
| MySQL persistence | Implemented |
| MQTT ingestion | Implemented |
| WebSocket streaming | Implemented |
| React dashboard | Implemented |
| Hardware simulator | Implemented |
| Physical ESP32 firmware | Planned |
| Alerting engine | Planned |
| Auth and production hardening | Planned |
| Automated test suite | Planned |

## Roadmap

1. Add alert thresholds for pH, EC, and temperature.
2. Add device registration and friendly device names.
3. Add authentication for dashboard and write endpoints.
4. Add production CORS and WebSocket origin allowlists.
5. Add firmware for the physical ESP32 sensor node.
6. Add exportable reports and longer-term analytics.
7. Add automated backend and frontend tests.

## Engineering Notes

- The backend uses a vertical feature structure under `internal/features/readings`.
- The frontend uses a small typed API client and a realtime hook rather than global state.
- The hardware simulator does not import backend code; it only shares the JSON wire contract, like a real device would.
- The API currently trusts local clients. Treat it as a development prototype until auth, rate limits, and production origin checks are added.
