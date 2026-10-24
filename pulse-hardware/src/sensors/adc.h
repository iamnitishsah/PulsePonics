#pragma once
// ADS1115 16-bit ADC on I2C (I2C_ADDR_ADS1115). Reads the pH and EC board voltages.
// Status: STUB, implement at Checkpoint 8.
// Test before connecting probes: wire A2 -> 3V3 and A3 -> GND; expect ~3300 mV and ~0 mV.

#include <stdint.h>

namespace adc {

bool begin();                 // returns true if the ADS1115 responded
float readMv(uint8_t ch);     // channel 0..3 in millivolts, or NAN on failure

}  // namespace adc
