# PulsePonics Hardware (ESP32 Firmware)

ESP32 firmware for the **Automated Real-Time Hydroponics Telemetry and Alerting System**. It reads the hydroponic sensors and publishes telemetry over MQTT to the PulsePonics backend.

```
DS18B20 ──1-Wire──▶ ESP32 ──Wi-Fi / MQTT──▶ Mosquitto ──▶ pulse-backend (Go) ──▶ MySQL ──▶ REST / WebSocket ──▶ pulse-frontend (React)
```

The firmware publishes the **same JSON shape as the backend's `POST /readings`**, so the backend and frontend need no changes. See `../pulse-backend/API_DOCUMENTATION.md`.

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

## 2. Project layout

```
pulse-hardware/
├── README.md
├── platformio.ini
├── include/
│   ├── secrets.h            # Wi-Fi + broker settings (gitignored, never commit)
│   └── secrets.example.h    # template for secrets.h
├── src/
│   └── main.cpp             # firmware
├── lib/
└── test/
```

---

## 3. Setup

### Toolchain
- **PlatformIO Core**: `brew install platformio`
- **CLion** with the bundled *PlatformIO for CLion* plugin. Open the folder containing `platformio.ini` as the project root.

### `platformio.ini`

```ini
[env:esp32dev]
platform = espressif32
board = esp32dev
framework = arduino
monitor_speed = 115200
lib_deps =
    paulstoffregen/OneWire
    milesburton/DallasTemperature
    256dpi/MQTT
    bblanchon/ArduinoJson
```

### Secrets
Copy `include/secrets.example.h` to `include/secrets.h` and fill it in:

```cpp
#pragma once
#define WIFI_SSID     "your-2.4GHz-network"
#define WIFI_PASSWORD "your-password"
#define MQTT_HOST     "192.168.x.x"   // broker IP (your Mac on the same network, or EC2)
#define MQTT_PORT     1883
```

- The ESP32 supports **2.4 GHz only**. Campus networks with a login page or an enterprise login will not work; use a phone hotspot (on iPhone, enable "Maximize Compatibility").
- `.gitignore` must contain `.pio/`, `.idea/` and `include/secrets.h`.

### Device ID
`DEVICE_ID` in `src/main.cpp` (default `esp32-hydro-01`) sets the MQTT topic and the `device_id` field. **The dashboard only shows the device it is configured for**, so make sure the frontend's selected device matches.

### Build, upload, monitor
In CLion's PlatformIO panel, use **Upload and Monitor**, or from the terminal:

```bash
pio run -t upload
pio device monitor
```

- If upload hangs at `Connecting....`, hold the **BOOT** button.
- `Could not exclusively lock port` means another serial monitor already holds the USB port. Stop it (■ / `Ctrl+C`) first.

---

## 4. MQTT contract

### Topics

| Topic | Direction | QoS | Retained | Payload |
|---|---|---|---|---|
| `hydroponics/{device_id}/telemetry` | ESP32 → broker | 1 | No | JSON reading (below) |
| `hydroponics/{device_id}/status` | ESP32 → broker | 1 | Yes | `online` on connect; `offline` sent by the broker as the **Last Will (LWT)** |

The backend subscribes to `hydroponics/+/telemetry`. The status topic is not consumed by the backend yet; it is available for monitoring and future offline alerts.

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
| `ec_ms_cm` | number / null | mS/cm | ADS1115 (planned) | ≥ 0 |
| `water_temp_c` | number / null | °C | DS18B20 | −10 to 60 |
| `air_temp_c` | number / null | °C | BME280 (planned) | −20 to 60 |
| `humidity_pct` | number / null | % RH | BME280 (planned) | 0–100 |
| `pressure_hpa` | number / null | hPa | BME280 (planned) | none |
| `recorded_at` | RFC3339 string, UTC | — | NTP clock | defaults to server time if omitted |

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
| Boot | Prints `Reset reason` (1 = power-on, 9 = brownout) for power diagnostics |
| Wi-Fi | Connects with a 20 s timeout; TX power reduced to 8.5 dBm to lower peak current |
| Time | NTP (`pool.ntp.org`, `time.google.com`), UTC, 30 s initial wait, keeps syncing in background |
| MQTT connect | Registers LWT, publishes retained `online`; retries every 5 s if the broker is down |
| Publish loop | Non-blocking `millis()` timer (10 s in development); `mqtt.loop()` keeps the connection alive |
| PUBACK timeout | 5 s. A failed reading is kept in a one-slot buffer and re-sent after reconnecting, with its original `recorded_at` |
| Wi-Fi loss | Detected in `loop()` and reconnected automatically |

---

## 6. Local development broker (macOS)

```bash
brew install mosquitto
printf "listener 1883 0.0.0.0\nallow_anonymous true\n" > ~/mosquitto-dev.conf
mosquitto -c ~/mosquitto-dev.conf -v             # Terminal 1: the broker
ipconfig getifaddr en0                            # Mac's IP → MQTT_HOST in secrets.h
mosquitto_sub -h localhost -t 'hydroponics/#' -v  # Terminal 2: watch all messages
```

- The Mac and the ESP32 must be on the **same network**.
- `allow_anonymous true` is for local testing only. **Never use it on EC2.**
- **Only one broker may run.** Homebrew can start a background Mosquitto at login that listens on `127.0.0.1` / `::1`. It silently captures `localhost` subscribers (including the backend) while the ESP32 talks to the other broker. Check and fix:
  ```bash
  lsof -nP -iTCP:1883 -sTCP:LISTEN     # must show exactly one mosquitto
  brew services stop mosquitto          # stop the background service
  ```
- The hotspot may assign the Mac a new IP after reconnecting. If MQTT fails, check `MQTT_HOST` first.

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
4. **Possible duplicate rows after a retry.** When a PUBACK times out but the broker did receive the message, the re-sent reading is stored twice, because the backend's `(device_id, recorded_at)` index is not `UNIQUE`.
   *Optional DB-only fix:* make that index `UNIQUE`. The duplicate insert then fails and is logged, with no code or frontend changes. Impact without the fix: an occasional duplicate chart point.
5. **"All sensors failed" is not reported.** Readings with no valid values are skipped (the backend would reject them), so a total sensor failure shows up only as missing data.

---

## 9. Roadmap

- [ ] Fix power (capacitor + direct cable), then re-test reconnect frequency with `WiFi.setSleep(false)`
- [ ] Wire ADS1115 + pH/EC probes; two-point calibration; temperature compensation using `water_temp_c`
- [ ] Wire BME280
- [ ] Switch `PUBLISH_INTERVAL_MS` to 60 s for deployment
- [ ] Move broker to EC2 with authentication (and TLS on port 8883); update `MQTT_HOST` and add credentials
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