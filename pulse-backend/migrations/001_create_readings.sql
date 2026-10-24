CREATE DATABASE IF NOT EXISTS pulseponics_db
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;

USE pulseponics_db;

CREATE TABLE IF NOT EXISTS readings(
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,

    device_id     VARCHAR(64)     NOT NULL,

    ph            DECIMAL(4,2)    NULL,
    ec_ms_cm      DECIMAL(6,3)    NULL,
    water_temp_c  DECIMAL(5,2)    NULL,
    air_temp_c    DECIMAL(5,2)    NULL,
    humidity_pct  DECIMAL(5,2)    NULL,
    pressure_hpa  DECIMAL(7,2)    NULL,

    recorded_at   DATETIME(3)     NOT NULL,
    created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),


    INDEX idx_device_time (device_id, recorded_at)
    ) ENGINE=InnoDB;