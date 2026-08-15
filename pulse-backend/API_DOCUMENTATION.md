# PulsePonics Backend API Documentation

PulsePonics backend is a Go HTTP API for storing hydroponics telemetry, reading recent sensor history, and pushing new readings to connected dashboards in real time.

## Base URL

Local development:

```text
http://localhost:8080
```

The server port is configured with `SERVER_PORT`. If unset, it defaults to `8080`.

## Content Type

REST endpoints consume and return JSON.

```http
Content-Type: application/json
Accept: application/json
```

## Data Model

### Reading

```json
{
  "id": 1,
  "device_id": "esp32-tank-01",
  "ph": 6.25,
  "ec_ms_cm": 1.82,
  "water_temp_c": 22.4,
  "air_temp_c": 26.1,
  "humidity_pct": 58.5,
  "pressure_hpa": 1012.4,
  "recorded_at": "2026-08-16T03:20:00+05:30",
  "created_at": "2026-08-16T03:20:00+05:30"
}
```

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `id` | integer | Response only | Database primary key. |
| `device_id` | string | Yes | Device identifier, for example `esp32-tank-01`. |
| `ph` | number or null | No | Nutrient solution pH. Valid range: `0` to `14`. |
| `ec_ms_cm` | number or null | No | Electrical conductivity in mS/cm. Must be `>= 0`. |
| `water_temp_c` | number or null | No | Water temperature in Celsius. Valid range: `-10` to `60`. |
| `air_temp_c` | number or null | No | Air temperature in Celsius. Valid range: `-20` to `60`. |
| `humidity_pct` | number or null | No | Relative humidity percentage. Valid range: `0` to `100`. |
| `pressure_hpa` | number or null | No | Atmospheric pressure in hPa. No validation range is currently enforced. |
| `recorded_at` | RFC3339 string | No on create | Time the device captured the reading. Defaults to server time when omitted or zero. |
| `created_at` | RFC3339 string | Response only | Time the backend stored the reading. |

At least one sensor value must be provided when creating a reading.

## Error Format

All HTTP errors return this shape:

```json
{
  "error": "device_id is required"
}
```

Common status codes:

| Status | Meaning |
| --- | --- |
| `400 Bad Request` | Invalid JSON, missing `device_id`, missing sensor data, or failed validation. |
| `404 Not Found` | No latest reading exists for the requested device. |
| `500 Internal Server Error` | Database or unexpected server failure. |

## REST Endpoints

### Create Reading

Stores a new telemetry reading.

```http
POST /readings
```

#### Request Body

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

`recorded_at` may be omitted:

```json
{
  "device_id": "esp32-tank-01",
  "ph": 6.2,
  "ec_ms_cm": 1.8
}
```

#### Success Response

```http
201 Created
```

```json
{
  "id": 1
}
```

#### Validation Errors

```http
400 Bad Request
```

Examples:

```json
{
  "error": "device_id is required"
}
```

```json
{
  "error": "at least one sensor value must be provided"
}
```

```json
{
  "error": "ph must be between 0 and 14"
}
```

#### curl Example

```bash
curl -X POST http://localhost:8080/readings \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "esp32-tank-01",
    "ph": 6.2,
    "ec_ms_cm": 1.8,
    "water_temp_c": 22.5,
    "air_temp_c": 26.1,
    "humidity_pct": 55,
    "pressure_hpa": 1012.5
  }'
```

### Get Reading History

Returns historical readings for one device.

```http
GET /readings
```

#### Query Parameters

| Parameter | Type | Required | Default | Description |
| --- | --- | --- | --- | --- |
| `device_id` | string | Yes | None | Device ID to query. |
| `from` | RFC3339 string | No | `to - 24h` | Start of the recorded-at time window. Invalid values are ignored. |
| `to` | RFC3339 string | No | Server current time | End of the recorded-at time window. Invalid values are ignored. |
| `limit` | integer | No | `100` | Max rows. Values `<= 0` or `> 1000` become `100`. |
| `offset` | integer | No | `0` | Rows to skip. |

Rows are returned by the database in descending `recorded_at` order.

#### Success Response

```http
200 OK
```

```json
[
  {
    "id": 12,
    "device_id": "esp32-tank-01",
    "ph": 6.21,
    "ec_ms_cm": 1.79,
    "water_temp_c": 22.6,
    "air_temp_c": 26.4,
    "humidity_pct": 56.1,
    "pressure_hpa": 1012.2,
    "recorded_at": "2026-08-16T03:25:00+05:30",
    "created_at": "2026-08-16T03:25:01+05:30"
  }
]
```

If no rows match, the response is an empty array.

#### curl Example

```bash
curl "http://localhost:8080/readings?device_id=esp32-tank-01&limit=100"
```

With an explicit time window:

```bash
curl "http://localhost:8080/readings?device_id=esp32-tank-01&from=2026-08-16T00:00:00%2B05:30&to=2026-08-16T23:59:59%2B05:30&limit=250"
```

### Get Latest Reading

Returns the most recent reading for one device.

```http
GET /readings/latest
```

#### Query Parameters

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `device_id` | string | Yes | Device ID to query. |

#### Success Response

```http
200 OK
```

```json
{
  "id": 12,
  "device_id": "esp32-tank-01",
  "ph": 6.21,
  "ec_ms_cm": 1.79,
  "water_temp_c": 22.6,
  "air_temp_c": 26.4,
  "humidity_pct": 56.1,
  "pressure_hpa": 1012.2,
  "recorded_at": "2026-08-16T03:25:00+05:30",
  "created_at": "2026-08-16T03:25:01+05:30"
}
```

#### Not Found Response

```http
404 Not Found
```

```json
{
  "error": "no readings found for this device"
}
```

#### curl Example

```bash
curl "http://localhost:8080/readings/latest?device_id=esp32-tank-01"
```

## WebSocket API

### Subscribe to Live Readings

Opens a websocket connection that receives each newly stored reading.

```text
ws://localhost:8080/ws
```

The server is push-only for application data. Clients do not need to send messages after connecting.

#### Message Format

Every websocket message is a JSON `Reading` object:

```json
{
  "id": 13,
  "device_id": "esp32-tank-01",
  "ph": 6.23,
  "ec_ms_cm": 1.81,
  "water_temp_c": 22.7,
  "air_temp_c": 26.5,
  "humidity_pct": 56.4,
  "pressure_hpa": 1012.1,
  "recorded_at": "2026-08-16T03:26:00+05:30",
  "created_at": "2026-08-16T03:26:01+05:30"
}
```

#### Browser Example

```js
const socket = new WebSocket('ws://localhost:8080/ws')

socket.onmessage = (event) => {
  const reading = JSON.parse(event.data)
  console.log(reading.device_id, reading.ph, reading.recorded_at)
}
```

#### Important Behavior

- The websocket broadcasts readings submitted through `POST /readings`.
- The websocket also broadcasts readings accepted from MQTT ingestion.
- The websocket endpoint currently accepts all origins in development.
- There is no per-device websocket filter; frontend clients should filter by `device_id`.

## MQTT Ingestion

The backend subscribes to telemetry published by devices or simulators.

### Broker

Configured by `MQTT_BROKER_URL`.

Default:

```text
tcp://localhost:1883
```

### Topic Pattern

```text
hydroponics/+/telemetry
```

The `+` segment allows multiple device-specific topics, for example:

```text
hydroponics/esp32-tank-01/telemetry
hydroponics/esp32-tank-02/telemetry
```

### Payload

The MQTT payload uses the same JSON shape as `POST /readings`.

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

Accepted MQTT readings are saved to MySQL and broadcast to websocket clients.

## Validation Rules

| Field | Rule | Error message |
| --- | --- | --- |
| `device_id` | Required | `device_id is required` |
| Sensor values | At least one sensor field required | `at least one sensor value must be provided` |
| `ph` | `0 <= ph <= 14` | `ph must be between 0 and 14` |
| `ec_ms_cm` | `ec_ms_cm >= 0` | `ec_ms_cm cannot be negative` |
| `water_temp_c` | `-10 <= water_temp_c <= 60` | `water_temp_c must be between -10 and 60` |
| `air_temp_c` | `-20 <= air_temp_c <= 60` | `air_temp_c must be between -20 and 60` |
| `humidity_pct` | `0 <= humidity_pct <= 100` | `humidity_pct must be between 0 and 100` |
| `pressure_hpa` | No current validation rule | None |

## Storage Notes

Readings are stored in the `readings` table.

Important columns:

| Column | Type | Description |
| --- | --- | --- |
| `id` | `BIGINT UNSIGNED AUTO_INCREMENT` | Primary key. |
| `device_id` | `VARCHAR(64)` | Device identifier. |
| `ph` | `DECIMAL(4,2)` | Nullable pH value. |
| `ec_ms_cm` | `DECIMAL(6,3)` | Nullable EC value. |
| `water_temp_c` | `DECIMAL(5,2)` | Nullable water temperature. |
| `air_temp_c` | `DECIMAL(5,2)` | Nullable air temperature. |
| `humidity_pct` | `DECIMAL(5,2)` | Nullable humidity. |
| `pressure_hpa` | `DECIMAL(7,2)` | Nullable pressure. |
| `recorded_at` | `DATETIME(3)` | Device reading timestamp. |
| `created_at` | `DATETIME(3)` | Server insertion timestamp. |

The table has an index on `(device_id, recorded_at)` for dashboard history queries.

## Environment Variables

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `DB_USER` | Yes | None | MySQL user. |
| `DB_PASSWORD` | Yes | None | MySQL password. |
| `DB_HOST` | No | `127.0.0.1` | MySQL host. |
| `DB_PORT` | No | `3306` | MySQL port. |
| `DB_NAME` | Yes | None | MySQL database name. |
| `SERVER_PORT` | No | `8080` | HTTP server port. |
| `MQTT_BROKER_URL` | No | `tcp://localhost:1883` | MQTT broker URL. |

## Frontend Integration Notes

For local Vite development, the frontend can proxy requests:

```text
/api/readings -> http://localhost:8080/readings
/api/readings/latest -> http://localhost:8080/readings/latest
/ws -> ws://localhost:8080/ws
```

Recommended frontend startup flow:

1. Fetch `GET /readings?device_id=...&limit=100` for chart history.
2. Fetch `GET /readings/latest?device_id=...` for the current snapshot.
3. Connect to `GET /ws`.
4. Filter websocket messages by `device_id`.
5. Append matching websocket messages to the chart buffer.

## Current Limitations

- No authentication or authorization is implemented.
- No production CORS/origin allowlist is implemented.
- The websocket stream is global and not filtered by device server-side.
- `from` and `to` query parameters silently fall back to defaults when invalid.
- Alerting and biological safety thresholds are not implemented in the backend yet.
