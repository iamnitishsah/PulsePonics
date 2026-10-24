#include "diagnostics.h"

#include <Arduino.h>
#include <Wire.h>
#include <esp_system.h>

#include "config.h"

namespace diag {

namespace {

const char *resetReasonName(esp_reset_reason_t r) {
  switch (r) {
    case ESP_RST_POWERON:   return "power-on";
    case ESP_RST_EXT:       return "external pin";
    case ESP_RST_SW:        return "software restart";
    case ESP_RST_PANIC:     return "crash (panic)";
    case ESP_RST_INT_WDT:   return "interrupt watchdog";
    case ESP_RST_TASK_WDT:  return "task watchdog";
    case ESP_RST_WDT:       return "watchdog";
    case ESP_RST_DEEPSLEEP: return "deep sleep wake";
    case ESP_RST_BROWNOUT:  return "BROWNOUT (power supply too weak)";
    default:                return "other";
  }
}

const char *deviceName(uint8_t addr) {
  if (addr >= 0x48 && addr <= 0x4B) return "ADS1115 (ADC)";
  if (addr == 0x76 || addr == 0x77) return "BME280/BMP280";
  return "unknown";
}

// Bosch sensors report a chip ID in register 0xD0: 0x60 = BME280, 0x58 = BMP280.
void printBoschChipId(uint8_t addr) {
  Wire.beginTransmission(addr);
  Wire.write(0xD0);
  if (Wire.endTransmission(false) != 0 || Wire.requestFrom(addr, (uint8_t)1) != 1) {
    Serial.println("      chip ID: read failed");
    return;
  }
  const uint8_t id = Wire.read();
  const char *kind = id == 0x60 ? "BME280 (temp + humidity + pressure)"
                   : id == 0x58 ? "BMP280 (NO humidity sensor)"
                                : "unrecognised";
  Serial.printf("      chip ID 0x%02X = %s\n", id, kind);
}

}  // namespace

void printBootInfo() {
  const esp_reset_reason_t r = esp_reset_reason();
  Serial.printf("\n=== PulsePonics firmware %s | device %s ===\n", FW_VERSION, DEVICE_ID);
  Serial.printf("Reset reason: %d (%s)\n", (int)r, resetReasonName(r));
}

void scanI2C() {
  Serial.printf("I2C scan (SDA %d, SCL %d):\n", PIN_I2C_SDA, PIN_I2C_SCL);
  int found = 0;
  for (uint8_t addr = 1; addr < 127; addr++) {
    Wire.beginTransmission(addr);
    if (Wire.endTransmission() != 0) continue;
    found++;
    Serial.printf("  0x%02X  %s\n", addr, deviceName(addr));
    if (addr == 0x76 || addr == 0x77) printBoschChipId(addr);
  }
  if (found == 0) Serial.println("  no I2C devices");
}

}  // namespace diag
