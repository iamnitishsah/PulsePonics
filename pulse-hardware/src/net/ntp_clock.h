#pragma once
// Real time from NTP, in UTC. Readings are only published once the clock is valid,
// so recorded_at is never a 1970 timestamp.

#include <stddef.h>
#include <time.h>

namespace ntp_clock {

void begin();                 // starts background NTP sync (safe to call before Wi-Fi is up)
void waitForSync();           // blocks up to NTP_SYNC_TIMEOUT_MS; sync continues afterwards
bool valid();
time_t now();

// RFC3339 UTC, e.g. "2026-10-24T08:20:00Z" (what the backend's Go time.Time decoder expects)
void formatRFC3339(time_t t, char *out, size_t len);

}  // namespace ntp_clock
