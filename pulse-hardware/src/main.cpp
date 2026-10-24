#include <Arduino.h>
#include <OneWire.h>
#include <DallasTemperature.h>

constexpr int ONE_WIRE_PIN = 4;

OneWire oneWire(ONE_WIRE_PIN);
DallasTemperature sensors(&oneWire);

void setup() {
    Serial.begin(115200);
    delay(500);

    sensors.begin();
    Serial.printf("DS18B20 devices found: %d\n", sensors.getDeviceCount());

    DeviceAddress addr;
    if (sensors.getAddress(addr, 0)) {
        Serial.print("Address: ");
        for (uint8_t i = 0; i < 8; i++) Serial.printf("%02X", addr[i]);
        Serial.println();
    }
    sensors.setResolution(12);
}

void loop() {
    sensors.requestTemperatures();
    float t = sensors.getTempCByIndex(0);

    if (t == DEVICE_DISCONNECTED_C) {
        Serial.println("water_temp: null  (fault: disconnected)");
    } else if (t == 85.0f) {
        Serial.println("water_temp: null  (fault: reset value)");
    } else {
        Serial.printf("water_temp: %.2f C\n", t);
    }
    delay(2000);
}