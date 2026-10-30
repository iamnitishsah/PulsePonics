#!/bin/sh
# Builds the password file and ACL from environment variables, then starts
# Mosquitto. Two accounts:
#   backend: reads every hydroponics/ topic (+ $SYS stats, for the healthcheck)
#   device:  shared by all ESP32s; may only publish telemetry and status
#            (pulse-hardware README, "MQTT contract")
set -eu

: "${MQTT_BACKEND_USERNAME:?MQTT_BACKEND_USERNAME is not set}"
: "${MQTT_BACKEND_PASSWORD:?MQTT_BACKEND_PASSWORD is not set}"
: "${MQTT_DEVICE_USERNAME:?MQTT_DEVICE_USERNAME is not set}"
: "${MQTT_DEVICE_PASSWORD:?MQTT_DEVICE_PASSWORD is not set}"

RUN_DIR=/mosquitto/run
mkdir -p "$RUN_DIR"
rm -f "$RUN_DIR/passwd"

mosquitto_passwd -c -b "$RUN_DIR/passwd" "$MQTT_BACKEND_USERNAME" "$MQTT_BACKEND_PASSWORD"
mosquitto_passwd -b "$RUN_DIR/passwd" "$MQTT_DEVICE_USERNAME" "$MQTT_DEVICE_PASSWORD"

cat > "$RUN_DIR/acl" <<ACL
user $MQTT_BACKEND_USERNAME
topic read hydroponics/#
topic read \$SYS/#

user $MQTT_DEVICE_USERNAME
topic write hydroponics/+/telemetry
topic write hydroponics/+/status
ACL

chown -R mosquitto:mosquitto "$RUN_DIR" /mosquitto/data
chmod 0700 "$RUN_DIR"
chmod 0600 "$RUN_DIR/passwd" "$RUN_DIR/acl"

exec /usr/sbin/mosquitto -c /mosquitto/config/mosquitto.conf
