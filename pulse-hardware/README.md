# PulsePonics Hardware (ESP32 Firmware)

ESP32 firmware for the **Automated Real-Time Hydroponics Telemetry and Alerting System**. It reads the hydroponic sensors and publishes telemetry over MQTT to the PulsePonics backend.

```
DS18B20 ──1-Wire──▶ ESP32 ──Wi-Fi / MQTT──▶ Mosquitto ──▶ pulse-backend (Go) ──▶ MySQL ──▶ REST / WebSocket ──▶ pulse-frontend (React)
```

The firmware publishes the **same JSON shape as the backend's `POST /api/readings`**, so the backend and frontend need no changes. See `../pulse-backend/API_DOCUMENTATION.md`.

---

## 1. Hardware

### Current build

| Component | Interface | ESP32 pin | Status |
|---|---|---|---|
| ESP32 DevKit (ESP32-D0WD-V3) | USB (CP2102/CH340) | — | ✅ Working |
| DS18B20 waterproof probe | 1-Wire | GPIO 4 | ✅ Working |
| 4.7 kΩ resistor | DATA → 3V3 pull-up | — | ✅ Required |

### Planned

| Component | Interface | Measures | Payload field |
|---|---|---|---|
| ADS1115 16-bit ADC | I2C (SDA 21, SCL 22) | Digitises pH and EC voltages | — |
| DFRobot Analog pH Kit V2 | Analog → ADS1115 | pH | `ph` |
| DFRobot Analog EC Kit V2 (K=1.0) | Analog → ADS1115 | EC (mS/cm) | `ec_ms_cm` |
| BME280 | I2C (SDA 21, SCL 22) | Air temp, humidity, pressure | `air_temp_c`, `humidity_pct`, `pressure_hpa` |

### DS18B20 wiring

| Probe wire | Connects to |
|---|---|
| Red (VDD) | ESP32 **3V3** |
| Black (GND) | ESP32 **GND** |
| Yellow (DATA) | ESP32 **GPIO 4** |
| 4.7 kΩ resistor | Between DATA and 3V3 |

```
3V3 ───┬──────────── Red
       │
     [4.7kΩ]
       │
GPIO4 ─┴──────────── Yellow (DATA)

GND ──────────────── Black
```

- Use **3V3, never 5V/VIN**. ESP32 GPIOs are not 5V tolerant.
- Breadboard rails on full-size boards are often **split in the middle**. Keep power, ground and the sensor on the same half.
- Rows A–E and F–J are **not** connected across the centre channel.

---

Full pin map, I²C addresses and wiring for every sensor: **[docs/wiring.md](docs/wiring.md)**.

---

## 2. Project layout

```
pulse-hardware/
├── platformio.ini                 # ESP32 build + native unit-test environment, pinned libraries
├── .gitignore
├── README.md
├── include/
│   ├── config.h                   # device ID, timing, pins, I2C addresses, backend ranges
│   ├── calibration.h              # pH / EC calibration constants
│   ├── secrets.example.h          # template → copy to secrets.h
│   └── secrets.h                  # Wi-Fi + broker (gitignored, never commit)
├── src/
│   ├── main.cpp                   # setup() + loop() only: orchestration
│   ├── diag/
│   │   └── diagnostics.h/.cpp     # boot info, reset reason, I2C scan (BME280 vs BMP280)
│   ├── net/
│   │   ├── wifi_link.h/.cpp       # Wi-Fi connect / reconnect
│   │   ├── ntp_clock.h/.cpp       # NTP time, RFC3339 formatting
│   │   └── mqtt_link.h/.cpp       # MQTT connect, LWT, QoS 1 publish, retry slot
│   ├── sensors/
│   │   ├── water_temp.h/.cpp      # DS18B20                       ✅ working
│   │   ├── air.h/.cpp             # BME280                        ⏳ stub (Checkpoint 7)
│   │   ├── adc.h/.cpp             # ADS1115                       ⏳ stub (Checkpoint 8)
│   │   └── probes.h/.cpp          # pH + EC via ADS1115           ⏳ math ready, needs adc + calibration
│   └── telemetry/
│       ├── readings.h             # SensorReadings struct (NAN = no value)
│       └── payload.h/.cpp         # JSON in the backend's POST /readings format
├── lib/
│   └── hydro_math/
│       └── hydro_math.h           # pure pH/EC/range math (no Arduino) → unit-testable
├── test/
│   └── test_hydro_math/
│       └── test_main.cpp          # unit tests, run on the Mac
└── docs/
    ├── wiring.md                  # pin map + per-sensor wiring
    └── calibration-log.csv        # record every calibration (report evidence)
```

### Design rules
- **One module per sensor.** Each exposes `begin()` and a read function that returns `NAN` for "no valid value". Adding a sensor never touches the network code.
- **`telemetry/payload` owns the backend contract.** Field names, validation ranges and rounding are in one place. Out-of-range values become `null` there.
- **Hardware-free math lives in `lib/hydro_math`**, so calibration logic is unit-tested without an ESP32.
- **Uncalibrated probes send `null`.** pH/EC stay `null` until `PH_CALIBRATED` / `EC_CALIBRATED` are set in `calibration.h`.
- **Settings are not hardcoded in modules.** Pins and timings go in `config.h`, calibration in `calibration.h`, credentials in `secrets.h`.

---

## 3. Setup

### Toolchain
- **PlatformIO Core**: `brew install platformio`
- **CLion** with the bundled *PlatformIO for CLion* plugin. Open the folder containing `platformio.ini` as the project root.

### `platformio.ini`
Two environments:
- **`esp32dev`** (default) builds the firmware. All sensor libraries are pinned, including the BME280 and ADS1X15 libraries needed for the next checkpoints.
- **`native`** runs the unit tests on the Mac. Keep **`esp32dev`** selected in CLion's environment dropdown for normal work.

### Secrets
Copy `include/secrets.example.h` to `include/secrets.h` and fill it in:

```cpp
#pragma once
#define WIFI_SSID     "your-2.4GHz-network"
#define WIFI_PASSWORD "your-password"
#define MQTT_HOST     "192.168.x.x"   // broker IP (your Mac on the same network, or EC2)
#define MQTT_PORT     1883
#define MQTT_USER     "pulse-device"  // MQTT_DEVICE_USERNAME from the repo's .env
#define MQTT_PASSWORD "..."           // MQTT_DEVICE_PASSWORD from the repo's .env
```

- The broker rejects anonymous clients. The firmware stops compiling with a clear `#error` if `MQTT_USER` / `MQTT_PASSWORD` are missing.

- The ESP32 supports **2.4 GHz only**. Campus networks with a login page or an enterprise login will not work; use a phone hotspot (on iPhone, enable "Maximize Compatibility").
- `.gitignore` must contain `.pio/`, `.idea/` and `include/secrets.h`.

### Device ID
`DEVICE_ID` in `include/config.h` (default `esp32-hydro-01`) sets the MQTT topic and the `device_id` field. **Register this exact ID in the dashboard** (Devices → Register device). The backend drops telemetry from unregistered IDs.

### Build, upload, monitor
In CLion's PlatformIO panel, use **Upload and Monitor**, or from the terminal:

```bash
pio run -t upload
pio device monitor
```

### Unit tests (no ESP32 needed)
```bash
pio test -e native
```
Tests the pH/EC conversion, temperature compensation and range checks in `lib/hydro_math`. Run them after every change to the calibration math.

### Upload tips
- If upload hangs at `Connecting....`, hold the **BOOT** button.
- `Could not exclusively lock port` means another serial monitor already holds the USB port. Stop it (■ / `Ctrl+C`) first.

---

## 4. MQTT contract

### Topics

| Topic | Direction | QoS | Retained | Payload |
|---|---|---|---|---|
| `hydroponics/{device_id}/telemetry` | ESP32 → broker | 1 | No | JSON reading (below) |
| `hydroponics/{device_id}/status` | ESP32 → broker | 1 | Yes | `online` on connect; `offline` sent by the broker as the **Last Will (LWT)** |

The backend subscribes to `hydroponics/+/telemetry`. It drops messages whose payload `device_id` differs from the topic, and messages from device IDs nobody has registered. The status topic is not consumed by the backend yet; it is available for monitoring and future offline alerts.

The broker requires a login. All devices share one account (`MQTT_DEVICE_USERNAME` / `MQTT_DEVICE_PASSWORD` in the repo's `.env`), which may only publish to `hydroponics/+/telemetry` and `hydroponics/+/status`.

### Telemetry payload
Identical to the backend's `POST /readings` body:

```json
{
  "device_id": "esp32-hydro-01",
  "ph": null,
  "ec_ms_cm": null,
  "water_temp_c": 28.44,
  "air_temp_c": null,
  "humidity_pct": null,
  "pressure_hpa": null,
  "recorded_at": "2026-10-24T08:20:00Z"
}
```

| Field | Type | Unit | Source | Backend rule |
|---|---|---|---|---|
| `device_id` | string | — | firmware constant | required |
| `ph` | number / null | pH | ADS1115 (planned) | 0–14 |
| `ec_ms_cm` | number / null | mS/cm | ADS1115 (planned) | 0–20 |
| `water_temp_c` | number / null | °C | DS18B20 | −10 to 60 |
| `air_temp_c` | number / null | °C | BME280 (planned) | −20 to 60 |
| `humidity_pct` | number / null | % RH | BME280 (planned) | 0–100 |
| `pressure_hpa` | number / null | hPa | BME280 (planned) | 300–1100 |
| `recorded_at` | RFC3339 string, UTC | — | NTP clock | defaults to server time if omitted; rejected if > 5 min in the future |

### Firmware rules
- **Failed or unavailable sensors send `null`**, never placeholder values. DS18B20's `-127` (disconnected) and `85.0` (power-on reset) are treated as failures.
- **Out-of-range values are sent as `null`.** The backend rejects the *whole* reading if any value is outside its range, so one bad sensor must not block the others.
- **No valid sensor value means no publish.** The backend rejects readings where every sensor is `null`, so the firmware skips them and logs to serial instead.
- **No valid clock means no publish.** Readings are sent only after NTP sync, so `recorded_at` is never a 1970 timestamp.
- Values are rounded to 2 decimals to match the DB's `DECIMAL(x,2)` columns.

---

## 5. Firmware behaviour

| Stage | Behaviour |
|---|---|
| Boot | Prints firmware version, device ID and `Reset reason` (9 = brownout), then an I²C scan |
| Wi-Fi | Connects with a 20 s timeout; TX power reduced to 8.5 dBm to lower peak current |
| Time | NTP (`pool.ntp.org`, `time.google.com`), UTC. Sync starts at boot even if Wi-Fi is down, waits 30 s, then continues in the background |
| Sensors | Water temp is read first; pH/EC use it for temperature compensation |
| MQTT connect | Registers LWT, publishes retained `online`; retries every 5 s if the broker is down |
| Publish loop | Non-blocking `millis()` timer (10 s in development); `mqtt.loop()` keeps the connection alive |
| PUBACK timeout | 5 s. A failed reading is kept in a one-slot buffer and re-sent after reconnecting, with its original `recorded_at` |
| Wi-Fi loss | Detected in `loop()` and reconnected automatically |

---

## 6. Local development broker

The broker runs in Docker together with the backend and MySQL (see the repository README):

```bash
cp .env.example .env                 # in the repo root; set the MQTT_* passwords
docker compose up -d --build         # Mosquitto on :1883 with username/password auth
ipconfig getifaddr en0               # Mac's IP → MQTT_HOST in secrets.h
docker compose exec mosquitto mosquitto_sub -h 127.0.0.1 \
  -u pulse-backend -P "<MQTT_BACKEND_PASSWORD>" -t 'hydroponics/#' -v   # watch all messages
```

- The Mac and the ESP32 must be on the **same network**.
- **Only one broker may run on port 1883.** Stop any Homebrew Mosquitto first:
  ```bash
  lsof -nP -iTCP:1883 -sTCP:LISTEN     # must show only Docker
  brew services stop mosquitto
  ```
- The hotspot may assign the Mac a new IP after reconnecting. If MQTT fails, check `MQTT_HOST` first.
- MQTT return code 4 or 5 on connect means `MQTT_USER` / `MQTT_PASSWORD` don't match `.env`.

---

## 7. Test checkpoints

| # | Checkpoint | Pass criteria | Status |
|---|---|---|---|
| 1 | Toolchain + blink | Onboard LED blinks, serial prints | ✅ |
| 2 | DS18B20 | Device found; responds to hand and cold water; unplugging DATA gives a fault | ✅ |
| 3 | Wi-Fi + NTP | Connects, `recorded_at` matches real UTC time | ✅ (power issue, see §8) |
| 4 | MQTT | Telemetry every interval, QoS 1 PUBACKs, retained status, LWT `offline` on unplug | ✅ |
| 5 | Backend → MySQL | Rows appear in `readings`; unused sensors stored as `NULL` (not `0`) | ⏳ |
| 6 | Dashboard | Live water temperature on the React dashboard | ⏳ |
| 7 | pH + EC (ADS1115) | Calibrated against buffer solutions | ⏳ |
| 8 | BME280 | Air temperature, humidity, pressure | ⏳ |

### Accuracy validation (planned for the report)
- **Ice-bath check:** crushed ice + a little water, stirred, is 0.0 °C. The DS18B20 reading there is its measured offset.
- **pH/EC:** two-point calibration with buffer solutions (pH 4.00 / 7.00; EC 1.413 / 12.88 mS/cm). Record the raw voltages and the derived slope and intercept.

---

## 8. Known issues

1. **Boot loops from insufficient USB power.** The board repeatedly resets (`Reset reason: 1`, hundreds of times) when the Wi-Fi radio starts. Confirmed by the DS18B20-only sketch running stably without Wi-Fi. Cause: an unpowered USB hub cannot supply the radio's current spikes.
   *Fix:* connect directly or via a powered hub with a short, good cable, and add a **100–470 µF electrolytic capacitor across 3V3 and GND** (long leg to 3V3). Do **not** disable the brownout detector.
2. **Periodic MQTT reconnects (~every 50 s on the phone hotspot).** The PUBACK sometimes arrives late or not at all; the firmware reconnects and re-sends the reading.
   *Next step:* test `WiFi.setSleep(false)` once power is fixed (it increases current draw).
3. **LWT flapping.** Each reconnect makes the broker publish `offline`, followed by `online` about 1 s later. Any future offline alert must be debounced (alert only if the device stays offline or `recorded_at` goes stale).
4. ~~**Possible duplicate rows after a retry.**~~ **Fixed in the backend:** `(device_id, recorded_at)` is now `UNIQUE`, so a re-sent reading is stored once and the duplicate is logged.
5. **"All sensors failed" is not reported.** Readings with no valid values are skipped (the backend would reject them), so a total sensor failure shows up only as missing data.

---

## 9. Roadmap

- [ ] Fix power (capacitor + direct cable), then re-test reconnect frequency with `WiFi.setSleep(false)`
- [ ] Wire ADS1115 + pH/EC probes; two-point calibration; temperature compensation using `water_temp_c`
- [ ] Wire BME280
- [ ] Switch `PUBLISH_INTERVAL_MS` to 60 s for deployment
- [x] Broker authentication (`MQTT_USER` / `MQTT_PASSWORD`)
- [ ] Move broker to EC2 (`docker compose --profile prod`) and update `MQTT_HOST`; optionally TLS on port 8883
- [ ] If the backend is extended later: send per-sensor fault flags and raw probe voltages for calibration review

---

## 10. Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| No `/dev/cu.usbserial-*` port | Charge-only cable or missing driver | Use a data cable; check the CP2102/CH340 chip |
| `Could not exclusively lock port` | Another monitor holds the port | Stop the other monitor first |
| Upload stuck at `Connecting....` | Board not entering bootloader | Hold BOOT during upload |
| `devices found: 0` / constant `-127` | Missing pull-up or wrong pin | Check the 4.7 kΩ between DATA and 3V3 and that DATA is on GPIO 4 |
| Constant `85.0` | Power problem at the sensor | Check red → 3V3, black → GND |
| `Wi-Fi FAILED (status 1)` | SSID not found / 5 GHz network | Use 2.4 GHz; check the SSID |
| `Wi-Fi FAILED (status 4)` | Wrong password | Fix `secrets.h` |
| Hundreds of `Reset reason` lines | Insufficient USB power | See Known issue 1 |
| MQTT `FAILED (err -3)` | Broker unreachable | Wrong `MQTT_HOST`, broker not running, different network, macOS firewall |
| `mosquitto_sub` shows nothing | Connected to a different broker | `lsof` check; stop the brew service |
| Backend: `at least one sensor value must be provided` | Payload field names don't match the backend | Firmware must send `water_temp_c` etc. (this version does) |
