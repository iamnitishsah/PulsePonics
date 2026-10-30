-- PulsePonics schema. All DATETIME columns hold UTC (the backend pins the
-- connection time zone to +00:00).

CREATE TABLE IF NOT EXISTS users (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    email         VARCHAR(254)    NOT NULL,
    name          VARCHAR(100)    NOT NULL,
    password_hash VARCHAR(100)    NOT NULL,
    created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Refresh tokens are stored as SHA-256 hashes; the raw token only exists on the client.
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id     BIGINT UNSIGNED NOT NULL,
    token_hash  CHAR(64)        NOT NULL,
    expires_at  DATETIME(3)     NOT NULL,
    revoked_at  DATETIME(3)     NULL,
    -- TRUE when revoked because it was exchanged for a new token. Presenting
    -- a rotated token again means it was copied, which revokes every session.
    rotated     BOOLEAN         NOT NULL DEFAULT FALSE,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_refresh_tokens_hash (token_hash),
    KEY idx_refresh_tokens_user (user_id),
    CONSTRAINT fk_refresh_tokens_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- device_id is the firmware's DEVICE_ID (config.h) and the MQTT topic segment.
-- It is globally unique: one device belongs to exactly one user.
CREATE TABLE IF NOT EXISTS devices (
    id             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    device_id      VARCHAR(64)     NOT NULL,
    user_id        BIGINT UNSIGNED NOT NULL,
    name           VARCHAR(100)    NOT NULL,
    api_key_hash   CHAR(64)        NOT NULL,
    api_key_prefix VARCHAR(16)     NOT NULL,
    created_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_devices_device_id (device_id),
    UNIQUE KEY uq_devices_api_key_hash (api_key_hash),
    KEY idx_devices_user (user_id),
    CONSTRAINT fk_devices_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- UNIQUE (device_id, recorded_at) drops the duplicate a firmware PUBACK retry
-- can cause (pulse-hardware README, Known issue 4).
CREATE TABLE IF NOT EXISTS readings (
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
    UNIQUE KEY uq_readings_device_time (device_id, recorded_at),
    CONSTRAINT fk_readings_device FOREIGN KEY (device_id) REFERENCES devices (device_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- One row per (device, metric) the user has customised; missing rows fall
-- back to the defaults in internal/features/alerts/thresholds.go.
CREATE TABLE IF NOT EXISTS thresholds (
    device_id   VARCHAR(64)  NOT NULL,
    metric      VARCHAR(32)  NOT NULL,
    min_value   DECIMAL(9,3) NULL,
    max_value   DECIMAL(9,3) NULL,
    enabled     BOOLEAN      NOT NULL DEFAULT TRUE,
    updated_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (device_id, metric),
    CONSTRAINT fk_thresholds_device FOREIGN KEY (device_id) REFERENCES devices (device_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- open_marker is 1 while an alert is open and NULL once resolved, so the
-- unique key allows at most one open alert per device and metric while
-- keeping any number of resolved ones.
CREATE TABLE IF NOT EXISTS alerts (
    id               BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    device_id        VARCHAR(64)   NOT NULL,
    metric           VARCHAR(32)   NOT NULL,
    kind             ENUM('low', 'high') NOT NULL,
    threshold_value  DECIMAL(9,3)  NOT NULL,
    trigger_value    DECIMAL(9,3)  NOT NULL,
    latest_value     DECIMAL(9,3)  NOT NULL,
    opened_at        DATETIME(3)   NOT NULL,
    resolved_at      DATETIME(3)   NULL,
    acknowledged_at  DATETIME(3)   NULL,
    open_marker      TINYINT GENERATED ALWAYS AS (IF(resolved_at IS NULL, 1, NULL)) STORED,
    UNIQUE KEY uq_alerts_open (device_id, metric, open_marker),
    KEY idx_alerts_device_opened (device_id, opened_at),
    CONSTRAINT fk_alerts_device FOREIGN KEY (device_id) REFERENCES devices (device_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
