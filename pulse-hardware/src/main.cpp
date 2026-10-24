// PulsePonics ESP32 firmware: reads the hydroponic sensors and publishes
// telemetry to the backend over MQTT.
//
// main.cpp only orchestrates. The real work lives in:
//   net/        Wi-Fi, NTP clock, MQTT
//   sensors/    one module per sensor (DS18B20, BME280, ADS1115, pH/EC probes)
//   telemetry/  JSON payload in the backend's format
//   diag/       boot diagnostics
// Settings: include/config.h   Calibration: include/calibration.h   Secrets: include/secrets.h

#include <Arduino.h>
#include <Wire.h>

#include "config.h"
#include "diag/diagnostics.h"
#include "net/mqtt_link.h"
#include "net/ntp_clock.h"
#include "net/wifi_link.h"
#include "sensors/air.h"
#include "sensors/probes.h"
#include "sensors/water_temp.h"
#include "telemetry/payload.h"
#include "telemetry/readings.h"

namespace {

uint32_t lastPublishMs = 0;

SensorReadings readSensors() {
  SensorReadings r;

  r.waterTempC = water_temp::readC();          // read first: pH/EC need it for compensation

  const air::Sample a = air::read();
  r.airTempC    = a.tempC;
  r.humidityPct = a.humidityPct;
  r.pressureHpa = a.pressureHpa;

  const probes::Sample p = probes::read(r.waterTempC);
  r.ph     = p.ph;
  r.ecMsCm = p.ecMsCm;

  return r;
}

void publishCycle() {
  if (!ntp_clock::valid()) {
    Serial.println("No valid time yet, publish skipped");
    return;
  }

  const SensorReadings r = readSensors();

  char payload[PAYLOAD_MAX_BYTES];
  const size_t n = telemetry::buildPayload(r, ntp_clock::now(), payload, sizeof(payload));
  if (n == 0) {
    Serial.println("No valid sensor values, publish skipped");
    return;
  }
  mqtt_link::publishTelemetry(payload, n);
}

}  // namespace

void setup() {
  Serial.begin(115200);
  delay(500);
  diag::printBootInfo();

  Wire.begin(PIN_I2C_SDA, PIN_I2C_SCL);
  diag::scanI2C();

  water_temp::begin();
  air::begin();
  probes::begin();

  ntp_clock::begin();                 // starts background sync even if Wi-Fi is down
  wifi_link::connect();
  if (wifi_link::connected()) ntp_clock::waitForSync();

  mqtt_link::begin();
  mqtt_link::ensureConnected();
}

void loop() {
  if (!wifi_link::ensure()) return;

  if (!mqtt_link::ensureConnected()) {
    delay(MQTT_RETRY_DELAY_MS);       // broker unreachable: retry later
    return;
  }
  mqtt_link::loop();                  // keep-alive + re-send a reading whose PUBACK timed out

  if (millis() - lastPublishMs >= PUBLISH_INTERVAL_MS) {
    lastPublishMs = millis();
    publishCycle();
  }
  delay(10);
}
