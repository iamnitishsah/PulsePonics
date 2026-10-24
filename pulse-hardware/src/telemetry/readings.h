#pragma once
// One measurement cycle. NAN = no valid value (sent to the backend as null).

#include <math.h>

struct SensorReadings {
  float ph          = NAN;
  float ecMsCm      = NAN;
  float waterTempC  = NAN;
  float airTempC    = NAN;
  float humidityPct = NAN;
  float pressureHpa = NAN;
};
