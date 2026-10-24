#include <Arduino.h>
#include <WiFi.h>
#include <MQTT.h>
#include <ArduinoJson.h>
#include <OneWire.h>
#include <DallasTemperature.h>
#include <esp_system.h>
#include <time.h>
#include "secrets.h"

// ---------- Config ----------
constexpr char     DEVICE_ID[]         = "esp32-hydro-01";
constexpr char     FW_VERSION[]        = "1.0.0";
constexpr int      ONE_WIRE_PIN        = 4;
constexpr uint32_t PUBLISH_INTERVAL_MS = 10000;      // 10 s while testing; README says 60 s
constexpr time_t   VALID_TIME_AFTER    = 1700000000; // before Nov 2023 = clock not synced
constexpr int      MQTT_TIMEOUT_MS     = 5000;       // wait up to 5 s for PUBACK over the hotspot

// fault_flags bits (README section 2)
constexpr uint8_t FAULT_PH = 1, FAULT_EC = 2, FAULT_WATER_TEMP = 4, FAULT_BME280 = 8;

const String TOPIC_TELEMETRY = String("hydroponics/") + DEVICE_ID + "/telemetry";
const String TOPIC_STATUS    = String("hydroponics/") + DEVICE_ID + "/status";

WiFiClient net;
MQTTClient mqtt(512);                 // 512-byte buffer: topic + JSON fit comfortably
OneWire oneWire(ONE_WIRE_PIN);
DallasTemperature sensors(&oneWire);
uint32_t lastPublish = 0;

// One-slot retry buffer: a reading whose PUBACK never arrived.
// It keeps its original ts, so if the broker did receive it the first time,
// the backend's UNIQUE (device_id, recorded_at) silently drops the duplicate.
// If a second reading fails before this one is re-sent, the newer one replaces it.
char   pendingPayload[384];
size_t pendingLen = 0;                // > 0 means one reading is waiting to be re-sent

// ---------- Wi-Fi + time ----------
void connectWiFi() {
  WiFi.mode(WIFI_STA);
  WiFi.setTxPower(WIFI_POWER_8_5dBm); // lower peak current; signal is strong anyway
  WiFi.begin(WIFI_SSID, WIFI_PASSWORD);
  Serial.printf("Connecting to %s", WIFI_SSID);
  uint32_t start = millis();
  while (WiFi.status() != WL_CONNECTED && millis() - start < 20000) {
    delay(500);
    Serial.print(".");
  }
  if (WiFi.status() == WL_CONNECTED)
    Serial.printf("\nWi-Fi OK | IP: %s | RSSI: %d dBm\n",
                  WiFi.localIP().toString().c_str(), WiFi.RSSI());
  else
    Serial.printf("\nWi-Fi FAILED (status %d)\n", WiFi.status());
}

bool timeIsValid() { return time(nullptr) > VALID_TIME_AFTER; }

void syncTime() {
  configTime(0, 0, "pool.ntp.org", "time.google.com");  // UTC, no offset
  Serial.print("Syncing time");
  uint32_t start = millis();
  while (!timeIsValid() && millis() - start < 30000) {
    delay(500);
    Serial.print(".");
  }
  Serial.println(timeIsValid() ? "\nTime OK" : "\nTime sync pending (will keep trying)");
}

// ---------- MQTT ----------
bool connectMQTT() {
  Serial.printf("MQTT connecting to %s:%d ... ", MQTT_HOST, MQTT_PORT);
  mqtt.setWill(TOPIC_STATUS.c_str(), "offline", true, 1);   // LWT: retained, QoS 1
  if (!mqtt.connect(DEVICE_ID)) {
    Serial.printf("FAILED (err %d, rc %d)\n", (int)mqtt.lastError(), (int)mqtt.returnCode());
    return false;
  }
  mqtt.publish(TOPIC_STATUS.c_str(), "online", true, 1);    // retained "online"
  Serial.println("OK");
  return true;
}

void publishTelemetry() {
  sensors.requestTemperatures();
  float t = sensors.getTempCByIndex(0);
  bool tempFault = (t == DEVICE_DISCONNECTED_C || t == 85.0f);

  // pH, EC (ADS1115) and BME280 aren't wired yet -> null + fault bit, per README
  uint8_t faults = FAULT_PH | FAULT_EC | FAULT_BME280;
  if (tempFault) faults |= FAULT_WATER_TEMP;

  JsonDocument doc;
  doc["device_id"] = DEVICE_ID;
  doc["ts"]        = (long)time(nullptr);
  doc["fw"]        = FW_VERSION;
  doc["ph"]        = nullptr;
  doc["ec"]        = nullptr;
  if (tempFault) doc["water_temp"] = nullptr;
  else           doc["water_temp"] = t;
  doc["air_temp"]    = nullptr;
  doc["humidity"]    = nullptr;
  doc["pressure"]    = nullptr;
  doc["ph_mv"]       = nullptr;
  doc["ec_mv"]       = nullptr;
  doc["fault_flags"] = faults;

  char payload[384];
  size_t n = serializeJson(doc, payload, sizeof(payload));
  bool ok = mqtt.publish(TOPIC_TELEMETRY.c_str(), payload, n, false, 1);  // QoS 1

  if (ok) {
    Serial.printf("Published (%u bytes): %s\n", (unsigned)n, payload);
  } else {
    memcpy(pendingPayload, payload, n);
    pendingLen = n;
    Serial.printf("PUBLISH FAILED, queued for retry: %s\n", payload);
  }
}

void retryPending() {
  if (pendingLen == 0) return;
  if (mqtt.publish(TOPIC_TELEMETRY.c_str(), pendingPayload, pendingLen, false, 1)) {
    Serial.println("Retried OK (backend dedupes if it was a duplicate)");
    pendingLen = 0;
  }
}

// ---------- Main ----------
void setup() {
  Serial.begin(115200);
  delay(500);
  Serial.printf("\nReset reason: %d\n", (int)esp_reset_reason());

  sensors.begin();
  sensors.setResolution(12);

  connectWiFi();
  if (WiFi.status() == WL_CONNECTED) syncTime();

  mqtt.begin(MQTT_HOST, MQTT_PORT, net);
  mqtt.setTimeout(MQTT_TIMEOUT_MS);
  connectMQTT();
}

void loop() {
  if (WiFi.status() != WL_CONNECTED) {
    Serial.println("Wi-Fi lost, reconnecting...");
    connectWiFi();
    return;
  }
  if (!mqtt.connected() && !connectMQTT()) {
    delay(5000);                       // broker down: retry every 5 s
    return;
  }
  mqtt.loop();                         // keeps the connection alive
  retryPending();                      // re-send a reading whose PUBACK timed out

  if (millis() - lastPublish >= PUBLISH_INTERVAL_MS) {
    lastPublish = millis();
    if (timeIsValid()) publishTelemetry();
    else Serial.println("No valid time yet, publish skipped");
  }
  delay(10);
}