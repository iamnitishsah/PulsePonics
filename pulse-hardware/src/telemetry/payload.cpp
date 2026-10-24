#include "payload.h"

#include <ArduinoJson.h>
#include <hydro_math.h>

#include "config.h"
#include "../net/ntp_clock.h"

namespace telemetry {

namespace {

// Out-of-range or missing -> null. Returns true if a real value was written.
bool put(JsonDocument &doc, const char *key, float value, float lo, float hi) {
  const float v = hydro::inRangeOrNan(value, lo, hi);
  if (isnan(v)) {
    doc[key] = nullptr;
    return false;
  }
  doc[key] = hydro::round2(v);
  return true;
}

}  // namespace

size_t buildPayload(const SensorReadings &r, time_t recordedAt, char *out, size_t outLen) {
  JsonDocument doc;
  bool any = false;

  doc["device_id"] = DEVICE_ID;
  any |= put(doc, "ph",           r.ph,          PH_MIN,           PH_MAX);
  any |= put(doc, "ec_ms_cm",     r.ecMsCm,      EC_MIN_MS_CM,     EC_MAX_MS_CM);
  any |= put(doc, "water_temp_c", r.waterTempC,  WATER_TEMP_MIN_C, WATER_TEMP_MAX_C);
  any |= put(doc, "air_temp_c",   r.airTempC,    AIR_TEMP_MIN_C,   AIR_TEMP_MAX_C);
  any |= put(doc, "humidity_pct", r.humidityPct, HUMIDITY_MIN_PCT, HUMIDITY_MAX_PCT);
  any |= put(doc, "pressure_hpa", r.pressureHpa, PRESSURE_MIN_HPA, PRESSURE_MAX_HPA);

  char ts[25];
  ntp_clock::formatRFC3339(recordedAt, ts, sizeof(ts));
  doc["recorded_at"] = ts;

  if (!any) return 0;                                  // backend rejects all-null readings
  if (measureJson(doc) >= outLen) return 0;            // would be truncated
  return serializeJson(doc, out, outLen);
}

}  // namespace telemetry
