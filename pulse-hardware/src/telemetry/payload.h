#pragma once
// Builds the MQTT telemetry JSON in the backend's POST /readings shape
// (pulse-backend API_DOCUMENTATION.md):
//   device_id, ph, ec_ms_cm, water_temp_c, air_temp_c, humidity_pct, pressure_hpa, recorded_at
//
// This module owns the backend contract: field names, validation ranges, rounding.

#include <stddef.h>
#include <time.h>

#include "readings.h"

namespace telemetry {

// Writes JSON into out. Returns its length, or 0 when nothing should be published
// (no valid sensor value, which the backend would reject, or the buffer is too small).
size_t buildPayload(const SensorReadings &r, time_t recordedAt, char *out, size_t outLen);

}  // namespace telemetry
