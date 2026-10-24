#include "water_temp.h"

#include <Arduino.h>
#include <OneWire.h>
#include <DallasTemperature.h>

#include "config.h"

namespace water_temp {

namespace {
OneWire oneWire(PIN_ONE_WIRE);
DallasTemperature sensors(&oneWire);
constexpr float DS18B20_POWER_ON_VALUE = 85.0f;   // returned before the first real conversion
}  // namespace

bool begin() {
  sensors.begin();
  sensors.setResolution(12);                       // 0.0625 degC steps, ~750 ms per reading
  const int count = sensors.getDeviceCount();
  Serial.printf("DS18B20: %d device(s) found\n", count);
  return count > 0;
}

float readC() {
  sensors.requestTemperatures();
  const float t = sensors.getTempCByIndex(0);
  if (t == DEVICE_DISCONNECTED_C || t == DS18B20_POWER_ON_VALUE) return NAN;   // -127 / 85
  return t;
}

}  // namespace water_temp
