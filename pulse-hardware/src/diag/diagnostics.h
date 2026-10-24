#pragma once
// Boot-time diagnostics printed to the serial monitor.

namespace diag {

void printBootInfo();   // firmware version, device ID, reset reason (9 = brownout)
void scanI2C();         // lists I2C devices; tells a BME280 from a BMP280 by chip ID

}  // namespace diag
