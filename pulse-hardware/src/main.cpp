#include <esp_system.h>
#include <Arduino.h>
#include <WiFi.h>
#include <time.h>
#include <OneWire.h>
#include <DallasTemperature.h>
#include "secrets.h"

constexpr int ONE_WIRE_PIN = 4;
constexpr time_t VALID_TIME_AFTER = 1700000000;  // Nov 2023: anything earlier means "not synced"

OneWire oneWire(ONE_WIRE_PIN);
DallasTemperature sensors(&oneWire);

void connectWiFi() {
  WiFi.mode(WIFI_STA);
  WiFi.setTxPower(WIFI_POWER_8_5dBm);  // lower radio peak current; your signal (-42 dBm) is strong anyway
  WiFi.begin(WIFI_SSID, WIFI_PASSWORD);
  Serial.printf("Connecting to %s", WIFI_SSID);

  uint32_t start = millis();
  while (WiFi.status() != WL_CONNECTED && millis() - start < 20000) {
    delay(500);
    Serial.print(".");
  }
  if (WiFi.status() == WL_CONNECTED) {
    Serial.printf("\nWi-Fi OK | IP: %s | RSSI: %d dBm\n",
                  WiFi.localIP().toString().c_str(), WiFi.RSSI());
  } else {
    Serial.printf("\nWi-Fi FAILED (status %d)\n", WiFi.status());
  }
}

bool timeIsValid() {
  return time(nullptr) > VALID_TIME_AFTER;
}

void syncTime() {
  configTime(0, 0, "pool.ntp.org", "time.google.com");  // UTC, no offset
  Serial.print("Syncing time");
  uint32_t start = millis();
  while (!timeIsValid() && millis() - start < 15000) {
    delay(500);
    Serial.print(".");
  }
  Serial.println(timeIsValid() ? "\nTime OK" : "\nTime sync FAILED");
}

void setup() {
  Serial.begin(115200);
  delay(500);
  Serial.printf("\nReset reason: %d\n", (int)esp_reset_reason());
  sensors.begin();
  sensors.setResolution(12);
  connectWiFi();
  if (WiFi.status() == WL_CONNECTED) syncTime();
}

void loop() {
  if (WiFi.status() != WL_CONNECTED) {
    Serial.println("Wi-Fi lost, reconnecting...");
    connectWiFi();
  }

  sensors.requestTemperatures();
  float t = sensors.getTempCByIndex(0);
  bool fault = (t == DEVICE_DISCONNECTED_C || t == 85.0f);

  if (timeIsValid()) {
    time_t now = time(nullptr);
    char buf[25];
    strftime(buf, sizeof(buf), "%Y-%m-%d %H:%M:%S", gmtime(&now));
    if (fault) Serial.printf("ts=%ld (%s UTC) water_temp=null\n", (long)now, buf);
    else       Serial.printf("ts=%ld (%s UTC) water_temp=%.2f\n", (long)now, buf, t);
  } else {
    Serial.println("No valid time yet, reading skipped");  // README rule: no ts → don't send
  }
  delay(5000);
}