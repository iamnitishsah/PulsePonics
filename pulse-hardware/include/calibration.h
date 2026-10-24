#pragma once
// Probe calibration constants. Update after each calibration and record the
// session in docs/calibration-log.csv (date, solution, mV, temperature).
//
// While *_CALIBRATED is false the firmware sends that value as null,
// so uncalibrated pH/EC numbers never reach the database.

// ---------- pH: two-point calibration (Checkpoint 9) ----------
// Put the probe in each buffer, wait for a stable reading, record the mV.
// Placeholders below are DFRobot's factory defaults for the Analog pH Kit V2.
constexpr bool  PH_CALIBRATED      = false;
constexpr float PH_CAL_MV_AT_7     = 1500.00f;   // mV in pH 7.00 buffer
constexpr float PH_CAL_MV_AT_4     = 2032.44f;   // mV in pH 4.00 buffer
constexpr float PH_CAL_TEMP_C      = 25.0f;      // buffer temperature during calibration

// ---------- EC: K value (Checkpoint 10) ----------
// Calibrate with the 1.413 mS/cm (and 12.88 mS/cm) standards.
// DFRobot's library uses separate K values for low and high ranges; one K is enough to start.
constexpr bool  EC_CALIBRATED      = false;
constexpr float EC_K_VALUE         = 1.0f;
constexpr float EC_TEMP_COEFF      = 0.02f;      // ~2 %/degC (DFRobot uses 0.0185)
