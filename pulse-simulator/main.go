// Command simulator stands in for the ESP32 firmware. It speaks the same MQTT
// contract as pulse-hardware (README section 4): QoS 1 telemetry on
// hydroponics/{device_id}/telemetry, and a retained online/offline status with
// a Last Will. It does not import backend code; it only shares the JSON wire
// format, exactly like the real device.
//
//	go run . -device esp32-hydro-01 -interval 10s
//
// The device must be registered in the dashboard first, or the backend drops
// its telemetry.
package main

import (
	"encoding/json"
	"flag"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

// telemetryPayload mirrors the firmware's JSON (pulse-hardware README,
// "Telemetry payload"). Pointers so a disabled sensor is sent as null.
type telemetryPayload struct {
	DeviceID    string   `json:"device_id"`
	PH          *float64 `json:"ph"`
	ECmScm      *float64 `json:"ec_ms_cm"`
	WaterTempC  *float64 `json:"water_temp_c"`
	AirTempC    *float64 `json:"air_temp_c"`
	HumidityPct *float64 `json:"humidity_pct"`
	PressureHPa *float64 `json:"pressure_hpa"`
	RecordedAt  string   `json:"recorded_at"` // RFC3339 UTC, like the NTP clock on the ESP32
}

// signalGen drifts slowly around a target, like a real tank: a random walk
// pulled back toward the target, clamped to the sensor's range.
type signalGen struct {
	value, target, step, pull, min, max float64
}

func (s *signalGen) next() *float64 {
	s.value += (rand.Float64()*2-1)*s.step + (s.target-s.value)*s.pull
	s.value = math.Max(s.min, math.Min(s.max, s.value))
	v := math.Round(s.value*100) / 100 // 2 decimals, like the firmware's round2
	return &v
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	brokerURL := flag.String("broker", envOr("MQTT_BROKER_URL", "tcp://localhost:1883"), "MQTT broker URL")
	username := flag.String("username", envOr("MQTT_USERNAME", ""), "MQTT username (the shared device login)")
	password := flag.String("password", envOr("MQTT_PASSWORD", ""), "MQTT password")
	deviceID := flag.String("device", envOr("SIM_DEVICE_ID", "esp32-hydro-01"), "device ID (must be registered in the dashboard)")
	interval := flag.Duration("interval", 10*time.Second, "publish interval (firmware: 10s dev, 60s deployment)")
	waterOnly := flag.Bool("water-only", false, "send only water_temp_c, like the current hardware build (DS18B20 only)")
	flag.Parse()

	telemetryTopic := "hydroponics/" + *deviceID + "/telemetry"
	statusTopic := "hydroponics/" + *deviceID + "/status"

	opts := paho.NewClientOptions().
		AddBroker(*brokerURL).
		SetClientID("simulator-"+*deviceID).
		SetUsername(*username).
		SetPassword(*password).
		SetWill(statusTopic, "offline", 1, true).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second)
	opts.SetOnConnectHandler(func(c paho.Client) {
		log.Printf("connected to %s", *brokerURL)
		c.Publish(statusTopic, 1, true, "online")
	})
	opts.SetConnectionLostHandler(func(_ paho.Client, err error) {
		log.Printf("connection lost: %v (reconnecting)", err)
	})

	client := paho.NewClient(opts)
	if token := client.Connect(); token.WaitTimeout(30*time.Second) && token.Error() != nil {
		log.Fatalf("connect: %v", token.Error())
	}

	gens := map[string]*signalGen{
		"ph":       {value: 6.0, target: 6.0, step: 0.04, pull: 0.02, min: 4.5, max: 7.5},
		"ec":       {value: 1.8, target: 1.8, step: 0.03, pull: 0.02, min: 0.8, max: 3.0},
		"water":    {value: 21.0, target: 21.0, step: 0.08, pull: 0.02, min: 15, max: 28},
		"air":      {value: 24.0, target: 24.0, step: 0.12, pull: 0.02, min: 16, max: 34},
		"humidity": {value: 60.0, target: 60.0, step: 0.6, pull: 0.02, min: 35, max: 85},
		"pressure": {value: 1013.0, target: 1013.0, step: 0.15, pull: 0.01, min: 995, max: 1030},
	}

	publish := func() {
		p := telemetryPayload{
			DeviceID:   *deviceID,
			WaterTempC: gens["water"].next(),
			RecordedAt: time.Now().UTC().Format(time.RFC3339),
		}
		if !*waterOnly {
			p.PH = gens["ph"].next()
			p.ECmScm = gens["ec"].next()
			p.AirTempC = gens["air"].next()
			p.HumidityPct = gens["humidity"].next()
			p.PressureHPa = gens["pressure"].next()
		}
		data, err := json.Marshal(p)
		if err != nil {
			log.Printf("marshal: %v", err)
			return
		}
		token := client.Publish(telemetryTopic, 1, false, data)
		if !token.WaitTimeout(5*time.Second) || token.Error() != nil {
			log.Printf("publish failed (no PUBACK): %v", token.Error())
			return
		}
		log.Printf("published %s", data)
	}

	log.Printf("simulating %s every %s on %s", *deviceID, *interval, telemetryTopic)
	publish()

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			publish()
		case <-stop:
			// A clean disconnect does not trigger the Last Will, so publish offline ourselves.
			client.Publish(statusTopic, 1, true, "offline").WaitTimeout(2 * time.Second)
			client.Disconnect(250)
			return
		}
	}
}
