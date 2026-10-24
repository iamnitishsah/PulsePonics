// Unit tests for the conversion math. Run on the Mac (no ESP32 needed):
//   pio test -e native
#include <unity.h>
#include <hydro_math.h>

using namespace hydro;

void setUp() {}
void tearDown() {}

// ---------- helpers ----------
void test_in_range_keeps_valid_value() {
  TEST_ASSERT_EQUAL_FLOAT(6.2f, inRangeOrNan(6.2f, 0.0f, 14.0f));
}

void test_in_range_rejects_out_of_range_and_nan() {
  TEST_ASSERT_TRUE(std::isnan(inRangeOrNan(-127.0f, -10.0f, 60.0f)));  // DS18B20 "disconnected"
  TEST_ASSERT_TRUE(std::isnan(inRangeOrNan(15.0f, 0.0f, 14.0f)));
  TEST_ASSERT_TRUE(std::isnan(inRangeOrNan(NAN, 0.0f, 14.0f)));
}

void test_round2() {
  TEST_ASSERT_EQUAL_FLOAT(28.44f, round2(28.4375f));
  TEST_ASSERT_TRUE(std::isnan(round2(NAN)));
}

// ---------- pH ----------
void test_ph_hits_both_calibration_points() {
  TEST_ASSERT_FLOAT_WITHIN(0.001f, 7.0f, phFromMv(1500.0f, 1500.0f, 2032.44f, 25.0f, 25.0f));
  TEST_ASSERT_FLOAT_WITHIN(0.001f, 4.0f, phFromMv(2032.44f, 1500.0f, 2032.44f, 25.0f, 25.0f));
}

void test_ph_is_neutral_at_any_temperature() {
  // Temperature compensation pivots around the pH 7 point.
  TEST_ASSERT_FLOAT_WITHIN(0.001f, 7.0f, phFromMv(1500.0f, 1500.0f, 2032.44f, 35.0f, 25.0f));
}

void test_ph_warmer_water_moves_reading_toward_7() {
  // Same voltage in warmer water = smaller pH deviation (electrode gives more mV per pH unit).
  const float at25 = phFromMv(2032.44f, 1500.0f, 2032.44f, 25.0f, 25.0f);
  const float at35 = phFromMv(2032.44f, 1500.0f, 2032.44f, 35.0f, 25.0f);
  TEST_ASSERT_TRUE(at35 > at25);
  TEST_ASSERT_TRUE(at35 < 7.0f);
}

void test_ph_unknown_temperature_returns_uncompensated() {
  TEST_ASSERT_FLOAT_WITHIN(0.001f, 4.0f, phFromMv(2032.44f, 1500.0f, 2032.44f, NAN, 25.0f));
}

void test_ph_invalid_inputs_return_nan() {
  TEST_ASSERT_TRUE(std::isnan(phFromMv(NAN, 1500.0f, 2032.44f, 25.0f, 25.0f)));
  TEST_ASSERT_TRUE(std::isnan(phFromMv(1500.0f, 1500.0f, 1500.0f, 25.0f, 25.0f)));  // bad calibration
}

// ---------- EC ----------
void test_ec_no_compensation_at_25c() {
  TEST_ASSERT_FLOAT_WITHIN(0.0001f, ecRawFromMv(1000.0f), ecFromMv(1000.0f, 1.0f, 25.0f, 0.02f));
}

void test_ec_compensation_direction() {
  // Warmer water conducts better, so the 25 degC-equivalent value must be lower.
  const float at25 = ecFromMv(1000.0f, 1.0f, 25.0f, 0.02f);
  const float at30 = ecFromMv(1000.0f, 1.0f, 30.0f, 0.02f);
  TEST_ASSERT_FLOAT_WITHIN(0.0001f, at25 / 1.10f, at30);
}

void test_ec_k_value_scales_linearly() {
  TEST_ASSERT_FLOAT_WITHIN(0.0001f, 2.0f * ecFromMv(1000.0f, 1.0f, 25.0f, 0.02f),
                           ecFromMv(1000.0f, 2.0f, 25.0f, 0.02f));
}

void test_ec_requires_temperature() {
  TEST_ASSERT_TRUE(std::isnan(ecFromMv(1000.0f, 1.0f, NAN, 0.02f)));
}

int main() {
  UNITY_BEGIN();
  RUN_TEST(test_in_range_keeps_valid_value);
  RUN_TEST(test_in_range_rejects_out_of_range_and_nan);
  RUN_TEST(test_round2);
  RUN_TEST(test_ph_hits_both_calibration_points);
  RUN_TEST(test_ph_is_neutral_at_any_temperature);
  RUN_TEST(test_ph_warmer_water_moves_reading_toward_7);
  RUN_TEST(test_ph_unknown_temperature_returns_uncompensated);
  RUN_TEST(test_ph_invalid_inputs_return_nan);
  RUN_TEST(test_ec_no_compensation_at_25c);
  RUN_TEST(test_ec_compensation_direction);
  RUN_TEST(test_ec_k_value_scales_linearly);
  RUN_TEST(test_ec_requires_temperature);
  return UNITY_END();
}
