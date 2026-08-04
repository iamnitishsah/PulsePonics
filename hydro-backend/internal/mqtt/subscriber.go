// internal/mqtt/subscriber.go
package mqtt

import (
	"context"
	"encoding/json"
	"log"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"

	"hydro-backend/internal/domain"
	"hydro-backend/internal/service"
)

// TopicPattern uses MQTT's "+" wildcard, which matches exactly one topic
// level. So "hydroponics/+/telemetry" matches "hydroponics/esp32-tank-01/telemetry",
// "hydroponics/esp32-tank-02/telemetry", etc. — this is what lets you add
// more devices later without touching this subscriber code at all.
const TopicPattern = "hydroponics/+/telemetry"

// Subscriber wraps a paho MQTT client and feeds incoming messages into the
// same ReadingService your HTTP handler already uses. Notice this struct
// only depends on *service.ReadingService — not on repository or MySQL
// directly. That's the payoff of the layered design: this file has zero
// idea how or where data eventually gets stored.
type Subscriber struct {
	client  paho.Client
	service *service.ReadingService
}

// NewSubscriber creates (but does not yet connect) an MQTT subscriber.
// brokerURL looks like "tcp://localhost:1883".
func NewSubscriber(brokerURL string, svc *service.ReadingService) *Subscriber {
	s := &Subscriber{service: svc}

	opts := paho.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID("hydro-backend-subscriber")

	// AutoReconnect matters a lot here: your ESP32/broker connection in
	// the real world WILL drop occasionally (Wi-Fi hiccup, broker
	// restart). Without this, a dropped connection means your backend
	// silently stops receiving data until you notice and restart it
	// manually — exactly the kind of silent failure a "serious project"
	// evaluation would catch.
	opts.SetAutoReconnect(true)
	opts.SetConnectRetryInterval(5 * time.Second)

	opts.OnConnect = func(c paho.Client) {
		log.Println("📡 MQTT connected to broker")
	}
	opts.OnConnectionLost = func(c paho.Client, err error) {
		log.Printf("⚠️  MQTT connection lost: %v (will auto-reconnect)", err)
	}

	s.client = paho.NewClient(opts)
	return s
}

// Start connects to the broker and subscribes to TopicPattern. This blocks
// briefly for the connection handshake, then returns — message handling
// happens asynchronously in the background via paho's own goroutines.
func (s *Subscriber) Start() error {
	token := s.client.Connect()
	token.Wait()
	if err := token.Error(); err != nil {
		return err
	}

	// QoS 1 = "at least once delivery". The broker will keep retrying
	// until it gets an acknowledgment from us. QoS 0 ("fire and forget")
	// is faster but can silently drop messages on a flaky connection —
	// for real sensor data you care about, QoS 1 is the right tradeoff.
	// (QoS 2, "exactly once", adds overhead you don't need here.)
	token = s.client.Subscribe(TopicPattern, 1, s.handleMessage)
	token.Wait()
	return token.Error()
}

// Stop disconnects cleanly. The 250 (ms) is how long paho waits for
// in-flight work to finish before forcing the disconnect.
func (s *Subscriber) Stop() {
	s.client.Disconnect(250)
}

// handleMessage is called by paho for every message matching TopicPattern.
// It has the exact same signature paho requires: (client, message) → void.
// paho does NOT let this function return an error — so we handle every
// failure by logging it. This is a real constraint of the library, not a
// design choice we made.
func (s *Subscriber) handleMessage(client paho.Client, msg paho.Message) {
	var input domain.NewReadingInput

	if err := json.Unmarshal(msg.Payload(), &input); err != nil {
		log.Printf("mqtt: failed to parse payload on topic %s: %v", msg.Topic(), err)
		return
	}

	// context.Background() here because this isn't tied to any HTTP
	// request lifecycle — it's a background subscription that lives for
	// as long as the whole app runs. We could add a timeout, but a plain
	// background context is the right default for "long-lived background
	// worker" as opposed to "per-request" work.
	id, err := s.service.SubmitReading(context.Background(), input)
	if err != nil {
		// Same validation logic as your HTTP endpoint runs here — a
		// malformed or out-of-range payload from a buggy sensor gets
		// rejected the same way a bad Postman request would.
		log.Printf("mqtt: rejected reading from topic %s: %v", msg.Topic(), err)
		return
	}

	log.Printf("mqtt: stored reading id=%d from topic %s", id, msg.Topic())
}
