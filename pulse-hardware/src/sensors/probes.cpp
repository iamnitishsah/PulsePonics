#include "probes.h"

#include <Arduino.h>
#include <hydro_math.h>

#include "adc.h"
#include "calibration.h"
#include "config.h"

namespace probes {

namespace {

// Optional EC power switching (Checkpoint 11): the EC probe's excitation current
// disturbs the pH reading when both probes share the same water. Powering the EC board
// only while reading EC (via a transistor on PIN_EC_POWER) is the cheap workaround;
// a signal isolator is the reliable fix.
constexpr bool EC_POWER_SWITCHED = PIN_EC_POWER >= 0;
constexpr uint32_t EC_SETTLE_MS = 500;   // tune at Checkpoint 11

void ecPower(bool on) {
  if (EC_POWER_SWITCHED) digitalWrite(static_cast<uint8_t>(PIN_EC_POWER), on ? HIGH : LOW);
}

}  // namespace

bool begin() {
  if (EC_POWER_SWITCHED) {
    pinMode(static_cast<uint8_t>(PIN_EC_POWER), OUTPUT);
    ecPower(false);
  }
  return adc::begin();
}

Sample read(float waterTempC) {
  Sample s;

  // pH first, with the EC board switched off (if switching is wired).
  s.phMv = adc::readMv(ADC_CH_PH);

  if (EC_POWER_SWITCHED) { ecPower(true); delay(EC_SETTLE_MS); }
  s.ecMv = adc::readMv(ADC_CH_EC);
  if (EC_POWER_SWITCHED) ecPower(false);

  if (PH_CALIBRATED) {
    s.ph = hydro::phFromMv(s.phMv, PH_CAL_MV_AT_7, PH_CAL_MV_AT_4, waterTempC, PH_CAL_TEMP_C);
  }
  if (EC_CALIBRATED) {
    s.ecMsCm = hydro::ecFromMv(s.ecMv, EC_K_VALUE, waterTempC, EC_TEMP_COEFF);
  }

  // Raw voltages for calibration (copy into docs/calibration-log.csv).
  if (!isnan(s.phMv) || !isnan(s.ecMv)) {
    Serial.printf("probes: ph_mv=%.1f ec_mv=%.1f water_temp=%.2f\n", s.phMv, s.ecMv, waterTempC);
  }
  return s;
}

}  // namespace probes
