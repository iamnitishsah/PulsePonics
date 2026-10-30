# PulsePonics API

Base URL: `http://localhost:8080` locally, `https://<API_DOMAIN>` on EC2. Every route is under `/api`.

- Bodies are JSON. Times are RFC 3339 in UTC (`2026-10-24T08:20:00Z`).
- **User routes** need `Authorization: Bearer <access_token>`.
- **Device ingestion** needs `X-Device-Key: <device api key>`.
- Errors are returned as `{"error": "message", "field": "optional_field"}` with one of these status codes:

| Status | Meaning |
|---|---|
| 400 | Invalid input (`field` names the offending field when there is one) |
| 401 | Missing, invalid or expired credentials |
| 403 | Authenticated, but not allowed (e.g. `device_id` doesn't match the key) |
| 404 | Not found, or belongs to another user |
| 409 | Conflict (email taken, device already registered, duplicate reading) |
| 429 | Too many auth attempts (about 10 per minute per IP) |
| 500 | Server error (details only in server logs) |

---

## Health

### `GET /api/healthz`
Returns `{"status":"ok"}`, or 503 if the database is unreachable.

### `GET /api/metrics`
The six sensor fields with label, unit and accepted range. Public.

---

## Auth

A **session** response (signup, login, refresh, password change):

```json
{
  "access_token": "eyJhbGciOi...",
  "access_token_expires_at": "2026-10-30T09:36:00Z",
  "refresh_token": "q3J...",
  "refresh_token_expires_at": "2026-11-06T09:21:00Z",
  "user": { "id": 1, "email": "grower@example.com", "name": "Grower", "created_at": "2026-10-30T09:21:00Z" }
}
```

Access tokens last 15 minutes. Exchange the refresh token for a new pair with `/api/auth/refresh`. Each refresh token works **once**: reusing a rotated token revokes every session of that user.

| Method & path | Auth | Body | Success |
|---|---|---|---|
| `POST /api/auth/signup` | none | `{email, password, name?}`: password at least 8 characters; name defaults to the part of the email before `@` | 201 session |
| `POST /api/auth/login` | none | `{email, password}` | 200 session |
| `POST /api/auth/refresh` | none | `{refresh_token}` | 200 session |
| `POST /api/auth/logout` | none | `{refresh_token}` | 204 |
| `GET /api/me` | user | none | 200 user |
| `POST /api/me/password` | user | `{current_password, new_password}` | 200 session (other sessions revoked) |
| `DELETE /api/me` | user | `{password}` | 204 (devices and data deleted) |

---

## Devices

`device_id` is the firmware's `DEVICE_ID`: 3–64 characters from `A–Z a–z 0–9 - _`. It is globally unique, so one device belongs to one user. Telemetry for unregistered IDs is ignored.

```json
{
  "device_id": "esp32-hydro-01",
  "name": "Lettuce tank",
  "api_key_prefix": "ppk_Ab12Cd",
  "created_at": "2026-10-30T09:22:00Z",
  "latest": { "...reading...": "or null" },
  "open_alerts": 1
}
```

| Method & path | Body | Success |
|---|---|---|
| `GET /api/devices` | none | 200 list (with `latest`, `open_alerts`) |
| `POST /api/devices` | `{device_id, name?}` | 201 device + `api_key` (**shown once**) · 409 if already registered |
| `GET /api/devices/{deviceID}` | none | 200 device |
| `PATCH /api/devices/{deviceID}` | `{name}` | 200 device |
| `DELETE /api/devices/{deviceID}` | none | 204 (readings, thresholds, alerts deleted) |
| `POST /api/devices/{deviceID}/api-key` | none | 200 device + new `api_key` (old key stops working) |

All of these require a user token.

---

## Readings

### Ingest over MQTT (the ESP32)

Topic `hydroponics/{device_id}/telemetry`, QoS 1, using the broker's device login. The payload is the one defined in `pulse-hardware/README.md`:

```json
{
  "device_id": "esp32-hydro-01",
  "ph": null, "ec_ms_cm": null, "water_temp_c": 28.44,
  "air_temp_c": null, "humidity_pct": null, "pressure_hpa": null,
  "recorded_at": "2026-10-24T08:20:00Z"
}
```

- At least one sensor value must be non-null.
- `recorded_at` defaults to the server time if omitted. More than 5 minutes in the future is rejected.
- The payload `device_id` must match the topic.
- A second reading with the same `device_id` + `recorded_at` is ignored.
- Rejected messages are logged by the backend, since MQTT has no response channel.

| Field | Accepted range |
|---|---|
| `ph` | 0–14 |
| `ec_ms_cm` | 0–20 mS/cm |
| `water_temp_c` | −10–60 °C |
| `air_temp_c` | −20–60 °C |
| `humidity_pct` | 0–100 % |
| `pressure_hpa` | 300–1100 hPa |

### `POST /api/readings`

Same payload, authenticated with `X-Device-Key`. `device_id` may be omitted (it's taken from the key); if present it must match. Returns 201 with the stored reading, or 409 for a duplicate.

```bash
curl -X POST http://localhost:8080/api/readings \
  -H "X-Device-Key: ppk_..." -H "Content-Type: application/json" \
  -d '{"water_temp_c": 21.5, "ph": 6.1}'
```

A stored reading looks like this:

```json
{
  "id": 42, "device_id": "esp32-hydro-01",
  "ph": 6.1, "ec_ms_cm": null, "water_temp_c": 21.5, "air_temp_c": null, "humidity_pct": null, "pressure_hpa": null,
  "recorded_at": "2026-10-30T09:22:21Z", "created_at": "2026-10-30T09:22:21.361Z"
}
```

### `GET /api/devices/{deviceID}/readings`

Chart history. Query parameters:

| Param | Default | Notes |
|---|---|---|
| `from` | `to` minus 24 h | RFC 3339 |
| `to` | now | RFC 3339 |
| `max_points` | 500 | 10–2000 |

The range may be at most 31 days. If the range holds more readings than `max_points`, they are averaged into equal time buckets.

```json
{
  "device_id": "esp32-hydro-01",
  "from": "2026-10-30T08:22:00Z", "to": "2026-10-30T09:22:00Z",
  "bucket_seconds": 0,
  "points": [ { "recorded_at": "...", "ph": 6.1, "ec_ms_cm": null, "...": "..." } ]
}
```

`bucket_seconds: 0` means raw readings. Otherwise each point is the average of the bucket starting at `recorded_at`.

### `GET /api/devices/{deviceID}/readings/latest`
Returns the newest reading, or 404 if there is none yet.

### `GET /api/devices/{deviceID}/readings/recent?limit=20`
Returns the newest readings, newest first. `limit` can be at most 200.

---

## Thresholds and alerts

### `GET /api/devices/{deviceID}/thresholds`
Returns all six metrics, with defaults where the user hasn't saved a value:

```json
{ "thresholds": [
  { "metric": "ph", "min": 5.5, "max": 6.5, "enabled": true },
  { "metric": "ec_ms_cm", "min": 1.2, "max": 2.4, "enabled": true },
  { "metric": "water_temp_c", "min": 18, "max": 24, "enabled": true },
  { "metric": "air_temp_c", "min": 18, "max": 28, "enabled": true },
  { "metric": "humidity_pct", "min": 45, "max": 75, "enabled": true },
  { "metric": "pressure_hpa", "min": null, "max": null, "enabled": false }
] }
```

### `PUT /api/devices/{deviceID}/thresholds`
The body is `{"thresholds": [...]}` with any subset of metrics. Rules:

- An enabled threshold needs `min`, `max`, or both.
- `min` must be less than `max`.
- Both bounds must lie inside the metric's accepted range.

The latest reading is re-evaluated immediately, and disabling a metric resolves its open alert. Returns the full effective list.

### Alert lifecycle

- An alert **opens** when a reading falls below `min` (`kind: "low"`) or above `max` (`kind: "high"`).
- While it is open, `latest_value` tracks each new reading.
- It **resolves** once the value is back inside the band by 5 % of the band's width.
- A `null` value for the metric leaves the alert as it is.
- There is at most one open alert per device and metric.

```json
{
  "id": 7, "device_id": "esp32-hydro-01", "metric": "ph", "kind": "high",
  "threshold_value": 6.5, "trigger_value": 7.1, "latest_value": 6.9,
  "opened_at": "2026-10-30T09:22:21Z", "resolved_at": null, "acknowledged_at": null,
  "status": "open"
}
```

### `GET /api/alerts?status=open|resolved|all&device_id=&limit=50`
Returns the caller's alerts, open ones first, then newest first. `limit` can be at most 200.

### `POST /api/alerts/{alertID}/acknowledge`
Marks the alert as seen and returns the updated alert. The alert stays open until the value recovers.

---

## WebSocket

`GET /api/ws?token=<access_token>` (`wss://` in production). Browsers can't send headers on WebSocket requests, which is why the token goes in the query string. It is checked once, at connect. The `Origin` must be in `CORS_ALLOWED_ORIGINS`.

The server only sends events for the user's own devices:

```json
{ "type": "reading", "data": { "...stored reading..." : "" } }
{ "type": "alert",   "data": { "...alert, status open or resolved..." : "" } }
```

The server pings every 54 seconds; clients that stop answering are dropped. After reconnecting, refetch the data, because events are not replayed.
