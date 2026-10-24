#include "wifi_link.h"

#include <Arduino.h>
#include <WiFi.h>

#include "config.h"
#include "secrets.h"

namespace wifi_link {

void connect() {
  WiFi.mode(WIFI_STA);
  WiFi.setTxPower(WIFI_POWER_8_5dBm);  // lower peak current (USB power is marginal); signal is strong
  WiFi.begin(WIFI_SSID, WIFI_PASSWORD);
  Serial.printf("Connecting to %s", WIFI_SSID);

  const uint32_t start = millis();
  while (WiFi.status() != WL_CONNECTED && millis() - start < WIFI_CONNECT_TIMEOUT_MS) {
    delay(500);
    Serial.print(".");
  }

  if (WiFi.status() == WL_CONNECTED) {
    Serial.printf("\nWi-Fi OK | IP: %s | RSSI: %d dBm\n",
                  WiFi.localIP().toString().c_str(), WiFi.RSSI());
  } else {
    // status 1 = SSID not found (5 GHz / typo), 4 = wrong password
    Serial.printf("\nWi-Fi FAILED (status %d)\n", WiFi.status());
  }
}

bool connected() { return WiFi.status() == WL_CONNECTED; }

bool ensure() {
  if (connected()) return true;
  Serial.println("Wi-Fi lost, reconnecting...");
  connect();
  return connected();
}

}  // namespace wifi_link
