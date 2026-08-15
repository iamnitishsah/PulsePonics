-- migrations/001_create_readings.sql
--
-- Design notes:
-- - One wide table (not one table per sensor). Since every reading from your
--   ESP32 arrives as a single JSON payload with all sensor values at once
--   (same timestamp), storing them as one row keeps writes O(1) and reads
--   simple. This matches how your firmware will publish data (Sprint 1).
-- - `recorded_at` is when the ESP32 took the reading; `created_at` is when
--   your server inserted it. They're usually the same, but you want both if
--   there's ever network delay or replay.
-- - Nullable sensor columns: if one sensor fails/is unplugged, you still want
--   to store the rest of the payload rather than rejecting the whole insert.

CREATE DATABASE IF NOT EXISTS pulseponics_db
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;

USE pulseponics_db;

CREATE TABLE IF NOT EXISTS readings (
                                        id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,

                                        device_id     VARCHAR(64)     NOT NULL,        -- e.g. "esp32-tank-01", lets you scale to multiple devices later

    ph            DECIMAL(4,2)    NULL,             -- e.g. 6.50
    ec_ms_cm      DECIMAL(6,3)    NULL,             -- electrical conductivity, mS/cm
    water_temp_c  DECIMAL(5,2)    NULL,             -- DS18B20
    air_temp_c    DECIMAL(5,2)    NULL,             -- BME280
    humidity_pct  DECIMAL(5,2)    NULL,             -- BME280
    pressure_hpa  DECIMAL(7,2)    NULL,             -- BME280

    recorded_at   DATETIME(3)     NOT NULL,         -- timestamp from the device/payload
    created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    -- This composite index is the single most important design decision here:
    -- your dashboard's main query pattern is "give me readings for device X
    -- between time A and B, ordered by time" — this index serves that
    -- directly without a filesort.
    INDEX idx_device_time (device_id, recorded_at)
    ) ENGINE=InnoDB;