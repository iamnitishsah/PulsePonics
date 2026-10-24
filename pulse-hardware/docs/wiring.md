# Wiring

All sensors run on **3V3**. Never connect a sensor or the ADS1115 to 5V/VIN: ESP32 pins are not 5V tolerant, and the ADS1115's inputs must never exceed its own supply.

Unplug USB before changing any wiring.

## Pin map

| ESP32 pin | Connected to | Notes |
|---|---|---|
| 3V3 | + rail | Powers every sensor |
| GND | − rail | Common ground for everything |
| GPIO 4 | DS18B20 DATA | 4.7 kΩ pull-up to 3V3 |
| GPIO 21 (SDA) | BME280 SDA, ADS1115 SDA | Shared I²C bus |
| GPIO 22 (SCL) | BME280 SCL, ADS1115 SCL | Shared I²C bus |
| GPIO 2 | Onboard blue LED | Not used by firmware |
| *(optional)* GPIO 25 | Transistor switching EC board power | Only if probe interference needs it (Checkpoint 11); set `PIN_EC_POWER` in `config.h` |

## I²C addresses

| Device | Address | Set by |
|---|---|---|
| ADS1115 | `0x48` | ADDR pin → GND |
| BME280 | `0x76` (some boards `0x77`) | SDO pin → GND gives `0x76` |

The firmware prints an I²C scan at every boot. If a device is missing from the scan, check its wiring before debugging code.

## DS18B20 (water temperature) ✅

| Probe wire | Connects to |
|---|---|
| Red | 3V3 |
| Black | GND |
| Yellow | GPIO 4 |
| 4.7 kΩ resistor | between Yellow (DATA) and 3V3 |

```
3V3 ───┬──────────── Red
     [4.7kΩ]
GPIO4 ─┴──────────── Yellow
GND ──────────────── Black
```

## BME280 (air temp, humidity, pressure): Checkpoint 7

| BME280 pin | Connects to |
|---|---|
| VIN / VCC | 3V3 |
| GND | GND |
| SCL | GPIO 22 |
| SDA | GPIO 21 |
| CSB *(6-pin boards only)* | 3V3 (selects I²C mode) |
| SDO *(6-pin boards only)* | GND (address `0x76`) |

At boot the scan prints the chip ID: **`0x60` = BME280**, **`0x58` = BMP280** (no humidity).

## ADS1115 (ADC for pH + EC): Checkpoint 8

| ADS1115 pin | Connects to |
|---|---|
| VDD | 3V3 |
| GND | GND |
| SCL | GPIO 22 |
| SDA | GPIO 21 |
| ADDR | GND (address `0x48`) |
| A0 | pH board signal |
| A1 | EC board signal |
| A2, A3 | spare (use for the 3V3/GND self-test) |
| ALRT | not connected |

**Self-test before connecting probes:** A2 → 3V3 should read about 3300 mV, A3 → GND about 0 mV.

## pH board (DFRobot Analog pH Kit V2): Checkpoint 9

| Board pin | Connects to |
|---|---|
| + | 3V3 |
| − | GND |
| A (signal) | ADS1115 A0 |
| BNC | pH probe |

Output is 0–3.0 V. Keep the probe bulb wet (KCl storage solution) at all times.

## EC board (DFRobot Analog EC Kit V2): Checkpoint 10

| Board pin | Connects to |
|---|---|
| + | 3V3 (or the switched supply, see Checkpoint 11) |
| − | GND |
| A (signal) | ADS1115 A1 |
| BNC | EC probe |

Output is 0–3.4 V.

## Probe interference: Checkpoint 11

With both probes in the same water, the EC probe can shift the pH reading.

1. Read pH with only the pH probe in the water.
2. Add the EC probe.
3. If pH moves by more than 0.1, fix it with a signal isolator (DFRobot DFR0504) on one probe, or by switching EC power through a transistor on `PIN_EC_POWER` (cheaper, less reliable).

Record the measured shift. It belongs in the report either way.
