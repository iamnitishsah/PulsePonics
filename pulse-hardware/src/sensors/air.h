#pragma once
// BME280 on I2C (I2C_ADDR_BME280): air temperature, humidity, pressure.
// Status: STUB, implement at Checkpoint 7.
//
// If the boot I2C scan reports chip ID 0x58, the board is a BMP280 (no humidity):
// keep humidityPct = NAN.

#include <math.h>

namespace air {

struct Sample {
  float tempC       = NAN;
  float humidityPct = NAN;
  float pressureHpa = NAN;
};

bool begin();     // returns true if the sensor responded
Sample read();    // fields are NAN when unavailable

}  // namespace air
