# PulsePonics Frontend

The frontend is a React + TypeScript dashboard for monitoring hydroponics telemetry in real time.

It uses Tailwind CSS for styling, fetches initial data from the backend REST API, and then listens for live readings over WebSocket.

## Responsibilities

- Let the user select a telemetry device ID.
- Fetch recent reading history for the selected device.
- Fetch the latest reading snapshot.
- Subscribe to backend WebSocket updates.
- Filter live readings by active `device_id`.
- Render current sensor cards, a trend chart, a snapshot panel, and recent readings table.
- Display backend connectivity and realtime connection state.

## Tech Stack

- React
- TypeScript
- Vite
- Tailwind CSS
- Native SVG for lightweight charts
- Native browser WebSocket API

No external charting library is currently used.

## Directory Structure

```text
pulse-frontend/
├── src/
│   ├── api/
│   │   └── readings.ts
│   ├── components/
│   │   ├── DeviceSelector.tsx
│   │   ├── LineChart.tsx
│   │   ├── ReadingTable.tsx
│   │   ├── SensorCard.tsx
│   │   ├── StatusPill.tsx
│   │   └── TrendPanel.tsx
│   ├── hooks/
│   │   └── useTelemetry.ts
│   ├── types/
│   │   └── reading.ts
│   ├── utils/
│   │   ├── sensors.ts
│   │   └── time.ts
│   ├── App.tsx
│   ├── index.css
│   └── main.tsx
├── index.html
├── package.json
├── vite.config.ts
└── README.md
```

## Design Approach

The interface is intentionally compact and operations-focused:

- no landing page
- no marketing hero section
- no custom CSS design system
- Tailwind utility classes only
- dense sensor cards for quick scanning
- a responsive chart area for trend inspection
- a recent readings table for raw values

`src/index.css` only imports Tailwind:

```css
@import "tailwindcss";
```

## Data Flow

```text
App
  -> useTelemetry(deviceId)
      -> GET /readings?device_id=...&limit=120
      -> GET /readings/latest?device_id=...
      -> WebSocket /ws
      -> filter messages by device_id
      -> merge realtime reading into history buffer
```

The backend returns historical rows in descending time order. The frontend sorts them ascending for chart rendering.

## Backend Integration

The Vite dev server proxies API calls to the Go backend:

```text
/api/readings         -> http://localhost:8080/readings
/api/readings/latest  -> http://localhost:8080/readings/latest
/ws                   -> ws://localhost:8080/ws
```

This keeps local development simple without requiring backend CORS changes.

## Environment Variables

Optional variables:

```env
VITE_API_BASE_URL=/api
VITE_WS_URL=ws://localhost:8080/ws
```

Defaults:

- `VITE_API_BASE_URL` defaults to `/api`
- `VITE_WS_URL` defaults to the current host with `/ws`

For normal local Vite development, no frontend `.env` file is required.

## Install

```bash
cd pulse-frontend
npm install
```

## Run Locally

Start the backend first on `localhost:8080`, then:

```bash
npm run dev
```

Open the URL printed by Vite, usually:

```text
http://localhost:5173
```

The dashboard defaults to:

```text
esp32-tank-01
```

Change the device field if your backend has readings for another device.

## Scripts

| Command | Purpose |
| --- | --- |
| `npm run dev` | Start Vite dev server |
| `npm run build` | Type-check and build production assets |
| `npm run lint` | Run ESLint |
| `npm run preview` | Preview the production build locally |

## Component Notes

### `useTelemetry`

The main data hook. It:

- loads history and latest reading
- maintains loading and error state
- opens a WebSocket connection
- reconnects after disconnect
- ignores readings from other devices
- keeps a bounded chart buffer

### `SensorCard`

Displays the latest value for one sensor and a state label:

- `Optimal`
- `Low`
- `High`
- `Tracking`
- `No data`

The target ranges are frontend presentation ranges, not backend validation rules.

### `LineChart`

Renders a responsive SVG line chart for one selected sensor. It supports missing sensor values by filtering out non-numeric points.

### `ReadingTable`

Shows the most recent readings in reverse chronological order for quick inspection.

## API Contract

The frontend expects the backend `Reading` shape documented in:

[../pulse-frontend/API_DOCUMENTATION.md](../pulse-frontend/API_DOCUMENTATION.md)

Important fields:

- `device_id`
- `recorded_at`
- `created_at`
- `ph`
- `ec_ms_cm`
- `water_temp_c`
- `air_temp_c`
- `humidity_pct`
- `pressure_hpa`

## Troubleshooting

### Dashboard says backend unavailable

Check that the backend is running:

```bash
cd ../pulse-backend
go run cmd/api/main.go
```

### WebSocket proxy shows `ECONNREFUSED`

The frontend is running, but the backend is not listening on `localhost:8080`.

### No chart data appears

The selected device may not have enough numeric readings for the active chart sensor. Publish simulator data or switch the selected sensor tab.

### New simulator data does not show

Verify all three services are running:

- MQTT broker
- Go backend
- hardware simulator

Also verify the simulator device ID matches the frontend device field.

## Production Notes

Before deploying:

- set explicit API and WebSocket URLs
- configure backend CORS and WebSocket origin allowlists
- add authentication
- serve frontend assets through a production web server or hosting platform
- add monitoring for backend and MQTT connectivity
