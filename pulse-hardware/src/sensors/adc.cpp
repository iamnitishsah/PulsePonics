#include "adc.h"

#include <Arduino.h>
#include <math.h>

#include "config.h"

// TODO (Checkpoint 8): implement with the Adafruit ADS1X15 library (already in lib_deps):
//
//   #include <Adafruit_ADS1X15.h>
//   Adafruit_ADS1115 ads;
//   begin():  ready = ads.begin(I2C_ADDR_ADS1115, &Wire);
//             ads.setGain(GAIN_ONE);        // +/-4.096 V range, 0.125 mV per bit
//   readMv(): int16_t raw = ads.readADC_SingleEnded(ch);
//             return ads.computeVolts(raw) * 1000.0f;
//
// Power the ADS1115 from 3V3: its inputs must never exceed its supply voltage.
// Averaging several samples (e.g. median of 10) gives steadier pH readings.

namespace adc {

namespace {
bool ready = false;
}  // namespace

bool begin() {
  ready = false;
  Serial.println("ADS1115: not implemented yet (pH/EC will be null)");
  return ready;
}

float readMv(uint8_t ch) {
  if (!ready || ch > 3) return NAN;
  // TODO (Checkpoint 8): read channel ch and return millivolts.
  return NAN;
}

}  // namespace adc
