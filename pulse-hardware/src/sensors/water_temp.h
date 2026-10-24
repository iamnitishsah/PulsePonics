#pragma once
// DS18B20 waterproof probe on 1-Wire (PIN_ONE_WIRE).   Status: WORKING
// Also used for pH/EC temperature compensation.

namespace water_temp {

bool begin();      // returns true if at least one probe was found
float readC();     // degC, or NAN if disconnected / power-on reset value

}  // namespace water_temp
