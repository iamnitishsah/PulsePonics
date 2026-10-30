# Mid-semester report (LaTeX)

## Compile on Overleaf

1. Zip this `report/` folder and use **New Project → Upload Project**. Keep the folder layout: `main.tex`, `figures/`, `data/`.
2. Compiler: **pdfLaTeX** (the Overleaf default). Main document: `main.tex`.
3. Compile twice so the contents, lists of figures and tables, and cross-references resolve. Overleaf does this automatically.

Optional: to add the institute logo on the title page, upload it as `figures/nitr_logo.png` and uncomment the two lines marked "Optional institute logo" in `main.tex`.

## Where the numbers come from

All results in Chapter 4 were measured on the running Docker stack on 30 October 2026 (development laptop, Apple Silicon, Docker Desktop). Client and server ran on the same machine, so no network latency is included.

| File | Contents |
|---|---|
| `data/measurements.txt` | Raw output of the measurement run: API latencies (p50/p95, 300 requests each), MQTT-to-WebSocket latency (200 messages), alert open/resolve times, aggregation result |
| `data/telemetry_6h.dat` | The 6-hour history exactly as returned by `GET /api/devices/{id}/readings` (2,160 readings at 10 s, aggregated to 361 one-minute points). Plotted in Figure 4.1 |
| `figures/*.png` | Screenshots of the dashboard running against the stack, with the simulator as the device |

The telemetry comes from the simulator, not from a physical tank, because the pH/EC/BME280 hardware is scheduled for the second half of the semester. The report says this in Sections 1.4 and 5.3.

Test counts: backend 48 cases (`go test -v ./...`), frontend 8 (`npm test`), firmware 12 (`pio test -e native`), end-to-end 38 (`scripts/smoke-test.sh`).
