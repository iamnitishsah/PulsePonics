#pragma once
// Copy this file to include/secrets.h and fill in real values.
// include/secrets.h is gitignored: never commit it.

#define WIFI_SSID     "your-2.4GHz-network"   // ESP32 supports 2.4 GHz only
#define WIFI_PASSWORD "your-wifi-password"

#define MQTT_HOST     "192.168.1.100"          // broker IP: your Mac (`ipconfig getifaddr en0`) or EC2
#define MQTT_PORT     1883
