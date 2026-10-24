#pragma once
// MQTT connection to the broker (MQTT_HOST / MQTT_PORT in include/secrets.h).
//   hydroponics/{DEVICE_ID}/telemetry  QoS 1, not retained
//   hydroponics/{DEVICE_ID}/status     "online" (retained) / "offline" via Last Will

#include <stddef.h>

namespace mqtt_link {

void begin();               // configure client; call once after Wi-Fi setup
bool ensureConnected();     // connects (with LWT + "online") if needed
void loop();                // keep-alive + re-send a reading whose PUBACK timed out

// Publishes at QoS 1. If the PUBACK doesn't arrive, the payload is kept in a
// one-slot buffer and re-sent after reconnecting (same recorded_at).
bool publishTelemetry(const char *payload, size_t len);

}  // namespace mqtt_link
