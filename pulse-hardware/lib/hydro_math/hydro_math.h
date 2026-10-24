#pragma once
// Pure conversion math: no Arduino, no hardware.
// Kept separate so it can be unit-tested on the Mac (`pio test -e native`).
// NAN means "no valid value"; it is sent to the backend as null.

#include <cmath>

namespace hydro {

constexpr float KELVIN_OFFSET = 273.15f;

// Returns v if it is a number inside [lo, hi], otherwise NAN.
inline float inRangeOrNan(float v, float lo, float hi) {
  return (std::isnan(v) || v < lo || v > hi) ? NAN : v;
}

// Rounds to 2 decimals (matches the DB's DECIMAL(x,2) columns). NAN stays NAN.
inline float round2(float v) {
  return std::isnan(v) ? NAN : std::round(v * 100.0f) / 100.0f;
}

// ---------- pH ----------
// Two-point calibration: a straight line through (mvAt7 -> 7.00) and (mvAt4 -> 4.00).
// The electrode's mV-per-pH slope grows with absolute temperature (Nernst equation),
// so the slope is rescaled from the calibration temperature to the current one.
// If the water temperature is unknown, the uncompensated value is returned
// (the error is small near pH 7).
inline float phFromMv(float mv, float mvAt7, float mvAt4, float tempC, float calTempC) {
  if (std::isnan(mv) || mvAt7 == mvAt4) return NAN;
  float phPerMv = (7.0f - 4.0f) / (mvAt7 - mvAt4);
  if (!std::isnan(tempC)) {
    phPerMv *= (calTempC + KELVIN_OFFSET) / (tempC + KELVIN_OFFSET);
  }
  return 7.0f + (mv - mvAt7) * phPerMv;
}

// ---------- EC ----------
// DFRobot Gravity Analog EC Meter V2 (K = 1), same math as DFRobot's reference library:
// raw EC from the board voltage (mV), scaled by the calibrated K value,
// then compensated to the 25 degC reference.
constexpr float EC_BOARD_RES2 = 820.0f;
constexpr float EC_BOARD_REF  = 200.0f;

inline float ecRawFromMv(float mv) {
  return 1000.0f * mv / EC_BOARD_RES2 / EC_BOARD_REF;
}

// EC changes ~2 % per degC, so without a water temperature the value isn't
// comparable between readings: returns NAN instead of guessing.
inline float ecFromMv(float mv, float kValue, float tempC, float tempCoeff) {
  if (std::isnan(mv) || std::isnan(tempC)) return NAN;
  const float ecAtT = ecRawFromMv(mv) * kValue;
  return ecAtT / (1.0f + tempCoeff * (tempC - 25.0f));
}

}  // namespace hydro
