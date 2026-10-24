#include "mqtt_link.h"

#include <Arduino.h>
#include <WiFi.h>
#include <MQTT.h>
#include <string.h>

#include "config.h"
#include "secrets.h"

namespace mqtt_link {

namespace {

WiFiClient net;
MQTTClient client(MQTT_BUFFER_SIZE);

const String topicTelemetry = String("hydroponics/") + DEVICE_ID + "/telemetry";
const String topicStatus    = String("hydroponics/") + DEVICE_ID + "/status";

// One-slot retry buffer. If a second reading fails before this one is re-sent,
// the newer one replaces it.
char   pendingPayload[PAYLOAD_MAX_BYTES];
size_t pendingLen = 0;

void retryPending() {
  if (pendingLen == 0) return;
  if (client.publish(topicTelemetry.c_str(), pendingPayload, (int)pendingLen, false, 1)) {
    Serial.println("Retried OK");
    pendingLen = 0;
  }
}

}  // namespace

void begin() {
  client.begin(MQTT_HOST, MQTT_PORT, net);
  client.setTimeout(MQTT_ACK_TIMEOUT_MS);
}

bool ensureConnected() {
  if (client.connected()) return true;

  Serial.printf("MQTT connecting to %s:%d ... ", MQTT_HOST, MQTT_PORT);
  client.setWill(topicStatus.c_str(), "offline", true, 1);   // LWT: retained, QoS 1
  if (!client.connect(DEVICE_ID)) {
    // err -3 = network/broker unreachable (wrong MQTT_HOST, broker down, other network)
    Serial.printf("FAILED (err %d, rc %d)\n", (int)client.lastError(), (int)client.returnCode());
    return false;
  }
  client.publish(topicStatus.c_str(), "online", true, 1);    // retained "online"
  Serial.println("OK");
  return true;
}

void loop() {
  client.loop();
  retryPending();
}

bool publishTelemetry(const char *payload, size_t len) {
  if (len == 0 || len > sizeof(pendingPayload)) return false;

  const bool ok = client.publish(topicTelemetry.c_str(), payload, (int)len, false, 1);  // QoS 1
  if (ok) {
    Serial.printf("Published (%u bytes): %.*s\n", (unsigned)len, (int)len, payload);
  } else {
    memcpy(pendingPayload, payload, len);
    pendingLen = len;
    Serial.printf("PUBLISH FAILED, queued for retry: %.*s\n", (int)len, payload);
  }
  return ok;
}

}  // namespace mqtt_link
