# PulsePonics Hardware

This folder contains hardware-facing code for PulsePonics. The current implementation is a Go-based MQTT simulator that behaves like a future ESP32 sensor node.

The simulator is useful before physical hardware is ready because it publishes realistic telemetry to the same MQTT topic and JSON contract the backend expects.

## Current Scope

- Simulate one or more hydroponic sensor devices.
- Publish telemetry over MQTT.
- Use the same payload shape as real firmware.
- Generate realistic values for pH, EC, temperature, humidity, and pressure.

Physical ESP32 firmware is planned but not implemented in this repository yet.

## Directory Structure

```text
pulse-hardware/
├── simulator/
│   ├── main.go
│   ├── go.mod
│   └── go.sum
└── README.md
```

## Simulator Overview

The simulator publishes to:

```text
hydroponics/{device_id}/telemetry
```

Example:

```text
hydroponics/esp32-tank-01/telemetry
```

Payload example:

```json
{
  "device_id": "esp32-tank-01",
  "ph": 6.25,
  "ec_ms_cm": 1.82,
  "water_temp_c": 22.4,
  "air_temp_c": 26.1,
  "humidity_pct": 58.5,
  "pressure_hpa": 1012.4,
  "recorded_at": "2026-08-16T03:20:00+05:30"
}
```

The backend subscribes to:

```text
hydroponics/+/telemetry
```

So multiple simulated devices can publish to different topics at the same time.

## Simulated Sensor Ranges

| Field | Simulated Range |
| --- | --- |
| `ph` | 5.5 to 7.0 |
| `ec_ms_cm` | 1.2 to 2.2 mS/cm |
| `water_temp_c` | 19 to 25 Celsius |
| `air_temp_c` | 22 to 28 Celsius |
| `humidity_pct` | 45 to 70 percent |
| `pressure_hpa` | 1008 to 1018 hPa |

These values are intentionally plausible rather than random noise, which makes dashboard and alerting development more meaningful.

## Prerequisites

- Go installed
- MQTT broker running, for example Mosquitto on `localhost:1883`
- PulsePonics backend running and connected to the same MQTT broker

The simulator module currently declares Go `1.26.6` in `go.mod`.

## Run the Simulator

From `pulse-hardware/simulator`:

```bash
go run .
```

Default behavior:

- broker: `tcp://localhost:1883`
- device: `esp32-tank-01`
- interval: `10s`

## Command-Line Flags

| Flag | Default | Purpose |
| --- | --- | --- |
| `-broker` | `tcp://localhost:1883` | MQTT broker URL |
| `-device` | `esp32-tank-01` | Simulated device ID |
| `-interval` | `10s` | Publish interval |

Example:

```bash
go run . -device esp32-tank-02 -interval 5s
```

Example with custom broker:

```bash
go run . -broker tcp://192.168.1.50:1883 -device greenhouse-a -interval 15s
```

## Multi-Device Simulation

Run multiple simulator processes with different device IDs:

```bash
go run . -device esp32-tank-01 -interval 10s
go run . -device esp32-tank-02 -interval 12s
go run . -device nursery-rack-01 -interval 8s
```

Each device publishes to a separate MQTT topic while using the same payload contract.

## End-to-End Local Test

1. Start an MQTT broker.
2. Start the backend:

   ```bash
   cd ../../pulse-backend
   go run cmd/api/main.go
   ```

3. Start the frontend:

   ```bash
   cd ../pulse-frontend
   npm run dev
   ```

4. Start the simulator:

   ```bash
   cd ../pulse-hardware/simulator
   go run . -device esp32-tank-01 -interval 10s
   ```

5. Open the frontend and select `esp32-tank-01`.

## Future Hardware Direction

The physical device will likely include:

- ESP32 microcontroller
- pH probe and signal conditioning board
- EC sensor
- DS18B20 or equivalent water temperature sensor
- BME280 or equivalent air temperature, humidity, and pressure sensor
- Wi-Fi connection
- NTP time synchronization
- MQTT publisher

The firmware should publish the same JSON contract documented in:

[../pulse-backend/API_DOCUMENTATION.md](../pulse-backend/API_DOCUMENTATION.md)

## Design Principle

The backend should not know whether telemetry came from a simulator or real hardware. Both should communicate through the same MQTT topic convention and JSON payload.
