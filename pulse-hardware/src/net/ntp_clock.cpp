#include "ntp_clock.h"

#include <Arduino.h>

#include "config.h"

namespace ntp_clock {

void begin() {
  // UTC, no offset. SNTP keeps retrying in the background, so this also works
  // if Wi-Fi was down at boot and connects later.
  configTime(0, 0, "pool.ntp.org", "time.google.com");
}

void waitForSync() {
  Serial.print("Syncing time");
  const uint32_t start = millis();
  while (!valid() && millis() - start < NTP_SYNC_TIMEOUT_MS) {
    delay(500);
    Serial.print(".");
  }
  Serial.println(valid() ? "\nTime OK" : "\nTime sync pending (will keep trying)");
}

bool valid() { return time(nullptr) > VALID_TIME_AFTER; }

time_t now() { return time(nullptr); }

void formatRFC3339(time_t t, char *out, size_t len) {
  struct tm utc;
  gmtime_r(&t, &utc);
  strftime(out, len, "%Y-%m-%dT%H:%M:%SZ", &utc);
}

}  // namespace ntp_clock
