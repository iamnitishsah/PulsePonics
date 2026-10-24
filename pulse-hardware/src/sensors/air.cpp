#include "air.h"

#include <Arduino.h>

#include "config.h"

// TODO (Checkpoint 7): implement with the Adafruit BME280 library (already in lib_deps):
//
//   #include <Adafruit_BME280.h>
//   Adafruit_BME280 bme;
//   begin():  ready = bme.begin(I2C_ADDR_BME280, &Wire);
//   read():   s.tempC       = bme.readTemperature();         // degC
//             s.humidityPct = bme.readHumidity();            // % RH (NAN on a BMP280)
//             s.pressureHpa = bme.readPressure() / 100.0f;   // Pa -> hPa
//
// Wire.begin() is already called in main.cpp before sensors start.

namespace air {

namespace {
bool ready = false;
}  // namespace

bool begin() {
  ready = false;
  Serial.println("BME280: not implemented yet (air values will be null)");
  return ready;
}

Sample read() {
  Sample s;              // all NAN -> sent as null
  if (!ready) return s;
  // TODO (Checkpoint 7): fill s from the sensor.
  return s;
}

}  // namespace air
