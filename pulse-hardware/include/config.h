#pragma once
// All tunable settings in one place: device identity, timing, pins, I2C addresses,
// and the backend's validation ranges. Calibration constants live in calibration.h.

#include <stdint.h>
#include <stddef.h>
#include <time.h>

// ---------- Device ----------
// Must match the device selected on the dashboard.
constexpr char DEVICE_ID[]  = "esp32-hydro-01";
constexpr char FW_VERSION[] = "1.1.0";        // printed at boot (the backend has no field for it)

// ---------- Timing ----------
constexpr uint32_t PUBLISH_INTERVAL_MS     = 10000;  // 10 s for development; 60000 for deployment
constexpr uint32_t WIFI_CONNECT_TIMEOUT_MS = 20000;
constexpr uint32_t NTP_SYNC_TIMEOUT_MS     = 30000;  // initial wait; sync keeps running in background
constexpr int      MQTT_ACK_TIMEOUT_MS     = 5000;   // wait for PUBACK (slow over a phone hotspot)
constexpr uint32_t MQTT_RETRY_DELAY_MS     = 5000;   // broker unreachable: retry interval
constexpr time_t   VALID_TIME_AFTER        = 1700000000;  // before Nov 2023 = clock not synced yet

// ---------- Pins (ESP32 DevKit) ----------
constexpr int PIN_ONE_WIRE = 4;    // DS18B20 DATA (+ 4.7 kOhm pull-up to 3V3)
constexpr int PIN_I2C_SDA  = 21;   // BME280 + ADS1115
constexpr int PIN_I2C_SCL  = 22;   // BME280 + ADS1115
constexpr int PIN_EC_POWER = -1;   // optional: switches EC board power (via a transistor) to stop
                                   // pH/EC probe interference. -1 = not used. Decide at Checkpoint 11.

// ---------- I2C addresses ----------
constexpr uint8_t I2C_ADDR_ADS1115 = 0x48;   // ADDR pin -> GND
constexpr uint8_t I2C_ADDR_BME280  = 0x76;   // some boards use 0x77 (check the boot I2C scan)

// ---------- ADS1115 channels ----------
constexpr uint8_t ADC_CH_PH = 0;   // A0 <- pH board signal
constexpr uint8_t ADC_CH_EC = 1;   // A1 <- EC board signal

// ---------- Backend validation ranges (pulse-backend API_DOCUMENTATION.md) ----------
// The backend rejects the WHOLE reading if any value is outside these,
// so the firmware sends an out-of-range value as null instead.
constexpr float PH_MIN = 0.0f,            PH_MAX = 14.0f;
constexpr float EC_MIN_MS_CM = 0.0f,      EC_MAX_MS_CM = 20.0f;      // backend: >= 0; 20 = probe limit
constexpr float WATER_TEMP_MIN_C = -10.0f, WATER_TEMP_MAX_C = 60.0f;
constexpr float AIR_TEMP_MIN_C = -20.0f,   AIR_TEMP_MAX_C = 60.0f;
constexpr float HUMIDITY_MIN_PCT = 0.0f,   HUMIDITY_MAX_PCT = 100.0f;
constexpr float PRESSURE_MIN_HPA = 300.0f, PRESSURE_MAX_HPA = 1100.0f;  // backend: no rule; sanity limit

// ---------- MQTT / payload ----------
constexpr int    MQTT_BUFFER_SIZE  = 512;   // topic + JSON must fit
constexpr size_t PAYLOAD_MAX_BYTES = 384;
