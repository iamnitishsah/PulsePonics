#!/usr/bin/env bash
# End-to-end check of the running Docker stack: auth, device registry, MQTT
# ingestion with the firmware's exact payload, duplicates, unregistered
# devices, HTTP ingestion, alerts, thresholds, history, WebSocket and CORS.
#
#   docker compose up -d --build
#   ./scripts/smoke-test.sh            # against http://localhost:8080
#   API=https://x.sslip.io ./scripts/smoke-test.sh
#
# Creates throwaway users and devices (random suffix). Needs curl, jq and
# Node 22+ (for the WebSocket check).
set -euo pipefail

cd "$(dirname "$0")/.."
API="${API:-http://localhost:8080}"
set -a; source .env; set +a

RUN="$(date +%s)$RANDOM"
DEVICE="esp32-smoke-$RUN"
PASS=0

ok()   { PASS=$((PASS + 1)); printf '  \033[32m✓\033[0m %s\n' "$1"; }
fail() { printf '  \033[31m✗ %s\033[0m\n' "$1"; exit 1; }
expect() { [ "$1" = "$2" ] && ok "$3" || fail "$3 (got '$1', want '$2')"; }

# api METHOD PATH [JSON] -> sets $STATUS and $BODY (uses $TOKEN if set)
api() {
  local args=(-s -o /tmp/pp_body -w '%{http_code}' -X "$1" "$API$2" -H 'Content-Type: application/json')
  [ -n "${TOKEN:-}" ] && args+=(-H "Authorization: Bearer $TOKEN")
  [ $# -ge 3 ] && args+=(-d "$3")
  STATUS=$(curl "${args[@]}")
  BODY=$(cat /tmp/pp_body)
}

mqtt_pub() { # topic payload [user] [password]
  docker compose exec -T mosquitto mosquitto_pub -h 127.0.0.1 -q 1 \
    -u "${3:-$MQTT_DEVICE_USERNAME}" -P "${4:-$MQTT_DEVICE_PASSWORD}" -t "$1" -m "$2"
}

now() { date -u +%Y-%m-%dT%H:%M:%SZ; }

echo "Smoke test against $API"

echo "Health"
api GET /api/healthz;                      expect "$STATUS" 200 "healthz"

echo "Auth"
TOKEN=""
api POST /api/auth/signup "{\"email\":\"smoke-$RUN@example.com\",\"password\":\"hydro-pass-1\",\"name\":\"Smoke\"}"
expect "$STATUS" 201 "signup"
TOKEN=$(jq -r .access_token <<<"$BODY"); REFRESH=$(jq -r .refresh_token <<<"$BODY")
api POST /api/auth/signup "{\"email\":\"smoke-$RUN@example.com\",\"password\":\"hydro-pass-1\"}"
expect "$STATUS" 409 "duplicate email rejected"
api POST /api/auth/login "{\"email\":\"smoke-$RUN@example.com\",\"password\":\"wrong-password\"}"
expect "$STATUS" 401 "wrong password rejected"
api GET /api/me;                           expect "$STATUS" 200 "GET /api/me with token"
SAVED=$TOKEN; TOKEN=""
api GET /api/devices;                      expect "$STATUS" 401 "devices require a token"
TOKEN=$SAVED

echo "Devices"
api POST /api/devices "{\"device_id\":\"$DEVICE\",\"name\":\"Smoke tank\"}"
expect "$STATUS" 201 "register device"
KEY=$(jq -r .api_key <<<"$BODY")
[[ "$KEY" == ppk_* ]] && ok "API key returned once" || fail "no API key"
api POST /api/devices "{\"device_id\":\"$DEVICE\"}"
expect "$STATUS" 409 "device ID can only be registered once"
api POST /api/devices '{"device_id":"bad/id"}'
expect "$STATUS" 400 "topic-unsafe device ID rejected"

echo "MQTT ingestion (firmware payload from pulse-hardware README)"
T1=$(now)
PAYLOAD="{\"device_id\":\"$DEVICE\",\"ph\":null,\"ec_ms_cm\":null,\"water_temp_c\":28.44,\"air_temp_c\":null,\"humidity_pct\":null,\"pressure_hpa\":null,\"recorded_at\":\"$T1\"}"
mqtt_pub "hydroponics/$DEVICE/telemetry" "$PAYLOAD"
sleep 1
api GET "/api/devices/$DEVICE/readings/latest"
expect "$STATUS" 200 "reading stored"
expect "$(jq -r .water_temp_c <<<"$BODY")" 28.44 "water_temp_c stored"
expect "$(jq -r .ph <<<"$BODY")" null "null sensors stay null"

mqtt_pub "hydroponics/$DEVICE/telemetry" "$PAYLOAD"   # PUBACK-retry duplicate
sleep 1
api GET "/api/devices/$DEVICE/readings/recent"
expect "$(jq length <<<"$BODY")" 1 "duplicate (same recorded_at) dropped"

mqtt_pub "hydroponics/esp32-unregistered-$RUN/telemetry" \
  "{\"device_id\":\"esp32-unregistered-$RUN\",\"water_temp_c\":20,\"recorded_at\":\"$(now)\"}"
sleep 1
docker compose logs --since 10s backend | grep -q "esp32-unregistered-$RUN\" is not registered" \
  && ok "unregistered device dropped" || fail "unregistered device not dropped"

if mqtt_pub "hydroponics/$DEVICE/telemetry" "$PAYLOAD" pulse-device wrong-password 2>/dev/null; then
  fail "broker accepted a wrong password"
else
  ok "broker rejects wrong MQTT password"
fi

echo "HTTP ingestion + alerts"
TOKEN=""
STATUS=$(curl -s -o /tmp/pp_body -w '%{http_code}' -X POST "$API/api/readings" \
  -H "X-Device-Key: $KEY" -H 'Content-Type: application/json' \
  -d "{\"ph\":7.1,\"water_temp_c\":21,\"recorded_at\":\"$(now)\"}")
expect "$STATUS" 201 "POST /api/readings with device key"
STATUS=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$API/api/readings" \
  -H "X-Device-Key: ppk_invalid" -H 'Content-Type: application/json' -d '{"ph":6}')
expect "$STATUS" 401 "invalid device key rejected"
TOKEN=$SAVED

api GET "/api/alerts?device_id=$DEVICE&status=open"
expect "$(jq length <<<"$BODY")" 1 "pH 7.1 opened a high alert (default max 6.5)"
expect "$(jq -r '.[0].kind' <<<"$BODY")" high "alert kind = high"
ALERT_ID=$(jq -r '.[0].id' <<<"$BODY")

api POST "/api/alerts/$ALERT_ID/acknowledge"
expect "$STATUS" 200 "acknowledge alert"

api PUT "/api/devices/$DEVICE/thresholds" '{"thresholds":[{"metric":"ph","min":5.5,"max":7.5,"enabled":true}]}'
expect "$STATUS" 200 "update pH threshold"
api GET "/api/alerts?device_id=$DEVICE&status=open"
expect "$(jq length <<<"$BODY")" 0 "raising max to 7.5 resolved the alert immediately"
api PUT "/api/devices/$DEVICE/thresholds" '{"thresholds":[{"metric":"ph","min":7,"max":6,"enabled":true}]}'
expect "$STATUS" 400 "min >= max rejected"

echo "History"
api GET "/api/devices/$DEVICE/readings?from=$(date -u -v-1H +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d '-1 hour' +%Y-%m-%dT%H:%M:%SZ)"
expect "$STATUS" 200 "history"
expect "$(jq '.points | length' <<<"$BODY")" 2 "history has both readings"
api GET "/api/devices/$DEVICE/readings?from=not-a-time"
expect "$STATUS" 400 "malformed from rejected"

echo "Isolation between users"
SAVED_A=$TOKEN; TOKEN=""
api POST /api/auth/signup "{\"email\":\"smoke-b-$RUN@example.com\",\"password\":\"hydro-pass-1\"}"
TOKEN=$(jq -r .access_token <<<"$BODY")
api GET "/api/devices/$DEVICE/readings/latest"
expect "$STATUS" 404 "another user cannot read this device"
api DELETE /api/me '{"password":"hydro-pass-1"}'
expect "$STATUS" 204 "delete second account"
TOKEN=$SAVED_A

echo "WebSocket"
WS_URL="${API/http/ws}/api/ws?token=$TOKEN"
node -e '
  const ws = new WebSocket(process.argv[1]);
  const timer = setTimeout(() => { console.error("no event"); process.exit(1) }, 8000);
  ws.onopen = () => console.log("open");
  ws.onmessage = (e) => { const m = JSON.parse(e.data); if (m.type === "reading") { clearTimeout(timer); console.log(m.data.water_temp_c); process.exit(0) } };
  ws.onerror = () => { console.error("ws error"); process.exit(1) };
' "$WS_URL" > /tmp/pp_ws &
WS_PID=$!
sleep 1.5
mqtt_pub "hydroponics/$DEVICE/telemetry" "{\"device_id\":\"$DEVICE\",\"water_temp_c\":22.5,\"recorded_at\":\"$(date -u -v+2S +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d '+2 sec' +%Y-%m-%dT%H:%M:%SZ)\"}"
wait $WS_PID && expect "$(tail -1 /tmp/pp_ws)" 22.5 "reading pushed over WebSocket" || fail "no WebSocket event"
STATUS=$(curl -s -o /dev/null -w '%{http_code}' "$API/api/ws?token=bad")
expect "$STATUS" 401 "WebSocket rejects a bad token"

echo "CORS"
ALLOWED=$(cut -d, -f1 <<<"$CORS_ALLOWED_ORIGINS")
H=$(curl -s -o /dev/null -D - -X OPTIONS "$API/api/devices" -H "Origin: $ALLOWED" -H 'Access-Control-Request-Method: GET' | tr -d '\r' | grep -i '^access-control-allow-origin' | cut -d' ' -f2)
expect "$H" "$ALLOWED" "preflight allowed for $ALLOWED"
H=$(curl -s -o /dev/null -D - -X OPTIONS "$API/api/devices" -H "Origin: https://evil.example" -H 'Access-Control-Request-Method: GET' | tr -d '\r' | grep -ci '^access-control-allow-origin' || true)
expect "$H" 0 "preflight denied for other origins"

echo "Sessions"
TOKEN=""
api POST /api/auth/refresh "{\"refresh_token\":\"$REFRESH\"}"
expect "$STATUS" 200 "refresh rotates tokens"
NEW_REFRESH=$(jq -r .refresh_token <<<"$BODY")
api POST /api/auth/logout "{\"refresh_token\":\"$NEW_REFRESH\"}"
expect "$STATUS" 204 "logout"
api POST /api/auth/refresh "{\"refresh_token\":\"$NEW_REFRESH\"}"
expect "$STATUS" 401 "logged-out refresh token rejected"

echo "Cleanup"
TOKEN=$SAVED_A
api DELETE "/api/devices/$DEVICE";         expect "$STATUS" 204 "delete device (cascades readings/alerts)"
api DELETE /api/me '{"password":"hydro-pass-1"}'; expect "$STATUS" 204 "delete account"

printf '\n\033[32mAll %d checks passed.\033[0m\n' "$PASS"
