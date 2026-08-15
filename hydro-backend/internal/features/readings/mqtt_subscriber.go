package readings

import (
	"context"
	"encoding/json"
	"log"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

const TopicPattern = "hydroponics/+/telemetry"

type Subscriber struct {
	client  paho.Client
	service *ReadingService
}

func NewMQTTSubscriber(brokerURL string, svc *ReadingService) *Subscriber {
	s := &Subscriber{service: svc}

	opts := paho.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID("hydro-backend-subscriber")

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

func (s *Subscriber) Start() error {
	token := s.client.Connect()
	token.Wait()
	if err := token.Error(); err != nil {
		return err
	}

	token = s.client.Subscribe(TopicPattern, 1, s.handleMessage)
	token.Wait()
	return token.Error()
}

func (s *Subscriber) Stop() {
	s.client.Disconnect(250)
}

func (s *Subscriber) handleMessage(client paho.Client, msg paho.Message) {
	var input NewReadingInput

	if err := json.Unmarshal(msg.Payload(), &input); err != nil {
		log.Printf("mqtt: failed to parse payload on topic %s: %v", msg.Topic(), err)
		return
	}

	id, err := s.service.SubmitReading(context.Background(), input)
	if err != nil {
		log.Printf("mqtt: rejected reading from topic %s: %v", msg.Topic(), err)
		return
	}

	log.Printf("mqtt: stored reading id=%d from topic %s", id, msg.Topic())
}
