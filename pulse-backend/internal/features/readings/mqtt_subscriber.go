package readings

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"

	"pulse-backend/internal/platform/apperr"
)

// TelemetryTopic matches hydroponics/{device_id}/telemetry (pulse-hardware README, "MQTT contract").
const TelemetryTopic = "hydroponics/+/telemetry"

type SubscriberConfig struct {
	BrokerURL string
	Username  string
	Password  string
	ClientID  string
}

type Subscriber struct {
	client  paho.Client
	service *Service
}

func NewMQTTSubscriber(cfg SubscriberConfig, svc *Service) *Subscriber {
	s := &Subscriber{service: svc}

	opts := paho.NewClientOptions().
		AddBroker(cfg.BrokerURL).
		SetClientID(cfg.ClientID).
		SetUsername(cfg.Username).
		SetPassword(cfg.Password).
		SetAutoReconnect(true).
		SetConnectRetry(true). // keep retrying if the broker isn't up yet at boot
		SetConnectRetryInterval(5 * time.Second).
		SetMaxReconnectInterval(30 * time.Second)

	// Subscribe on every (re)connect: with a clean session the broker forgets
	// subscriptions when the connection drops, so subscribing once at startup
	// would silently stop ingestion after the first broker restart.
	opts.SetOnConnectHandler(func(c paho.Client) {
		log.Printf("mqtt: connected to %s, subscribing to %s", cfg.BrokerURL, TelemetryTopic)
		token := c.Subscribe(TelemetryTopic, 1, s.handleMessage)
		go func() {
			if token.WaitTimeout(10*time.Second) && token.Error() != nil {
				log.Printf("mqtt: subscribe failed: %v", token.Error())
			}
		}()
	})
	opts.SetConnectionLostHandler(func(_ paho.Client, err error) {
		log.Printf("mqtt: connection lost: %v (reconnecting)", err)
	})

	s.client = paho.NewClient(opts)
	return s
}

// Start begins connecting in the background; it does not wait for the broker.
func (s *Subscriber) Start() {
	s.client.Connect()
}

func (s *Subscriber) Stop() {
	s.client.Disconnect(250)
}

func (s *Subscriber) handleMessage(_ paho.Client, msg paho.Message) {
	topicDevice := deviceFromTopic(msg.Topic())

	var in Input
	if err := json.Unmarshal(msg.Payload(), &in); err != nil {
		log.Printf("mqtt: %s: invalid JSON: %v", msg.Topic(), err)
		return
	}
	if in.DeviceID == "" {
		in.DeviceID = topicDevice
	}
	// A device may only publish its own data.
	if in.DeviceID != topicDevice {
		log.Printf("mqtt: %s: payload device_id %q does not match topic, dropped", msg.Topic(), in.DeviceID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reading, err := s.service.Ingest(ctx, in)
	switch {
	case err == nil:
		log.Printf("mqtt: stored reading id=%d device=%s", reading.ID, reading.DeviceID)
	case errors.Is(err, ErrUnknownDevice):
		log.Printf("mqtt: device %q is not registered, reading dropped", in.DeviceID)
	case errors.Is(err, ErrDuplicate):
		log.Printf("mqtt: duplicate reading device=%s recorded_at=%s ignored (firmware retry)",
			in.DeviceID, in.RecordedAt.UTC().Format(time.RFC3339))
	case apperr.IsKind(err, apperr.KindInvalid):
		log.Printf("mqtt: rejected reading from %s: %v", in.DeviceID, err)
	default:
		log.Printf("mqtt: failed to store reading from %s: %v", in.DeviceID, err)
	}
}

func deviceFromTopic(topic string) string {
	parts := strings.Split(topic, "/")
	if len(parts) == 3 {
		return parts[1]
	}
	return ""
}
