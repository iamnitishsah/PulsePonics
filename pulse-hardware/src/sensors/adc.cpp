#include "adc.h"

#include <Arduino.h>
#include <Wire.h>
#include <Adafruit_ADS1X15.h>
#include <math.h>

#include "config.h"

namespace adc {

namespace {

Adafruit_ADS1115 ads;
bool ready = false;

constexpr uint16_t MUX[4] = {
  ADS1X15_REG_CONFIG_MUX_SINGLE_0, ADS1X15_REG_CONFIG_MUX_SINGLE_1,
  ADS1X15_REG_CONFIG_MUX_SINGLE_2, ADS1X15_REG_CONFIG_MUX_SINGLE_3};

constexpr int      SAMPLES            = 16;  // per reading; averaging removes probe noise
constexpr uint32_t CONVERSION_TIMEOUT = 50;  // ms; one conversion takes ~8 ms at 128 SPS

// One conversion with a timeout. The library's readADC_SingleEnded() loops forever
// if the chip stops answering (loose SDA/SCL), which would freeze the whole board.
bool convertOnce(uint8_t ch, float &mv) {
  ads.startADCReading(MUX[ch], /*continuous=*/false);
  const uint32_t t0 = millis();
  while (!ads.conversionComplete()) {
    if (millis() - t0 > CONVERSION_TIMEOUT) return false;
    delay(1);
  }
  mv = ads.computeVolts(ads.getLastConversionResults()) * 1000.0f;
  return true;
}

void sortSmall(float *v, int n) {
  for (int i = 1; i < n; i++) {
    const float x = v[i];
    int j = i - 1;
    while (j >= 0 && v[j] > x) { v[j + 1] = v[j]; j--; }
    v[j + 1] = x;
  }
}

}  // namespace

bool begin() {
  ready = ads.begin(I2C_ADDR_ADS1115, &Wire);
  if (!ready) {
    Serial.println("ADS1115: NOT found at 0x48 (pH/EC will be null) - check wiring");
    return false;
  }
  ads.setGain(GAIN_ONE);                  // +/-4.096 V range, 0.125 mV per bit
  ads.setDataRate(RATE_ADS1115_128SPS);

  // Self-test: one raw reading per channel (jumper A2->3V3 ~3300, A3->GND ~0 to verify).
  Serial.print("ADS1115: found. A0..A3 mV:");
  for (uint8_t ch = 0; ch < 4; ch++) {
    float mv;
    if (convertOnce(ch, mv)) Serial.printf(" %.1f", mv);
    else                     Serial.print(" ERR");
  }
  Serial.println();
  return true;
}

float readMv(uint8_t ch) {
  if (!ready || ch > 3) return NAN;

  float s[SAMPLES];
  for (int i = 0; i < SAMPLES; i++) {
    if (!convertOnce(ch, s[i])) {
      Serial.printf("ADS1115: no answer on A%d (check SDA/SCL)\n", (int)ch);
      return NAN;
    }
  }
  // Trimmed mean: drop the lowest and highest quarter, average the middle half.
  sortSmall(s, SAMPLES);
  float sum = 0;
  for (int i = SAMPLES / 4; i < SAMPLES - SAMPLES / 4; i++) sum += s[i];
  return sum / (SAMPLES - 2 * (SAMPLES / 4));
}

}  // namespace adc