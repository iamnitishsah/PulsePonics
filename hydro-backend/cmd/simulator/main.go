// cmd/simulator/main.go
//
// This simulates your ESP32 firmware (Sprint 1) before real hardware
// exists. It publishes fake-but-realistic sensor readings to the same MQTT
// topic your real device will eventually use — so when hardware arrives,
// you literally just stop running this program and start the real ESP32;
// nothing on the Go backend side changes.
package main

import (
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

// telemetryPayload mirrors the readings feature's NewReadingInput JSON shape.
// We don't import that package here on purpose — this simulator represents
// an entirely separate physical device (the ESP32) that has no knowledge
// of your Go backend's internal types. It only needs to agree on the JSON
// wire format, exactly like real firmware would.
type telemetryPayload struct {
	DeviceID    string  `json:"device_id"`
	PH          float64 `json:"ph"`
	ECmScm      float64 `json:"ec_ms_cm"`
	WaterTempC  float64 `json:"water_temp_c"`
	AirTempC    float64 `json:"air_temp_c"`
	HumidityPct float64 `json:"humidity_pct"`
	PressureHPa float64 `json:"pressure_hpa"`
	RecordedAt  string  `json:"recorded_at"` // RFC3339 — real ESP32 would use its NTP-synced clock
}

func main() {
	// Command-line flags let you simulate multiple devices or tune the
	// interval without editing code — e.g.
	//   go run cmd/simulator/main.go -device esp32-tank-02 -interval 10s
	brokerURL := flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	deviceID := flag.String("device", "esp32-tank-01", "simulated device ID")
	interval := flag.Duration("interval", 10*time.Second, "publish interval")
	flag.Parse()

	topic := "hydroponics/" + *deviceID + "/telemetry"

	opts := paho.NewClientOptions()
	opts.AddBroker(*brokerURL)
	opts.SetClientID("simulator-" + *deviceID)

	client := paho.NewClient(opts)
	token := client.Connect()
	token.Wait()
	if err := token.Error(); err != nil {
		log.Fatalf("simulator: failed to connect to broker: %v", err)
	}
	defer client.Disconnect(250)

	log.Printf("🌱 simulator started — publishing to %s every %s", topic, *interval)

	// time.Tick gives us a channel that fires every `interval`. Combined
	// with the for-range below, this is Go's idiomatic way to run
	// "do X every N seconds forever" without manual sleep-loop bookkeeping.
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	// Publish one reading immediately on startup too, rather than waiting
	// for the first tick — useful so you see output right away instead of
	// waiting a full interval.
	publishReading(client, topic, *deviceID)

	for range ticker.C {
		publishReading(client, topic, *deviceID)
	}
}

// publishReading generates one plausible-looking reading and publishes it.
// Values are randomized within realistic hydroponic ranges (not wild
// noise) so the data looks like something a real system would produce —
// useful for testing your dashboard/alerting logic meaningfully later.
func publishReading(client paho.Client, topic, deviceID string) {
	payload := telemetryPayload{
		DeviceID:    deviceID,
		PH:          round2(5.5 + rand.Float64()*1.5),   // 5.5–7.0, typical hydroponic range
		ECmScm:      round2(1.2 + rand.Float64()*1.0),   // 1.2–2.2 mS/cm
		WaterTempC:  round2(19.0 + rand.Float64()*6.0),  // 19–25°C
		AirTempC:    round2(22.0 + rand.Float64()*6.0),  // 22–28°C
		HumidityPct: round2(45.0 + rand.Float64()*25.0), // 45–70%
		PressureHPa: round2(1008.0 + rand.Float64()*10), // ~1008–1018 hPa
		RecordedAt:  time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("simulator: failed to marshal payload: %v", err)
		return
	}

	// QoS 1 to match the subscriber's expectation. Wait() to confirm the
	// broker actually accepted it before we log success.
	token := client.Publish(topic, 1, false, data)
	token.Wait()
	if err := token.Error(); err != nil {
		log.Printf("simulator: publish failed: %v", err)
		return
	}

	log.Printf("📤 published: ph=%.2f ec=%.2f water=%.1f°C air=%.1f°C humidity=%.1f%% pressure=%.1fhPa",
		payload.PH, payload.ECmScm, payload.WaterTempC, payload.AirTempC, payload.HumidityPct, payload.PressureHPa)
}

func round2(v float64) float64 {
	return float64(int(v*100)) / 100
}
