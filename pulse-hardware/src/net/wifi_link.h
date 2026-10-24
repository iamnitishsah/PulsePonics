#pragma once
// Wi-Fi station connection (credentials in include/secrets.h).

namespace wifi_link {

void connect();     // blocking, up to WIFI_CONNECT_TIMEOUT_MS
bool connected();
bool ensure();      // reconnects if the link dropped; returns true when connected

}  // namespace wifi_link
