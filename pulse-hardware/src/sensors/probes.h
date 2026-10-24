#pragma once
// DFRobot Analog pH Kit V2 (ADS1115 A0) and Analog EC Kit V2 (ADS1115 A1).
// Status: math and calibration flow ready; returns NAN until adc.cpp is implemented
// and the *_CALIBRATED flags in include/calibration.h are set.

#include <math.h>

namespace probes {

struct Sample {
  float ph     = NAN;
  float ecMsCm = NAN;   // compensated to 25 degC
  float phMv   = NAN;   // raw voltages: printed for the calibration log
  float ecMv   = NAN;
};

bool begin();
Sample read(float waterTempC);   // water temperature is used for compensation

}  // namespace probes
