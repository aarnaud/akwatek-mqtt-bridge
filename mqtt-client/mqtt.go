package mqtt_client

import (
	"akwatek-mqtt-bridge/models"
	"akwatek-mqtt-bridge/utils"
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog/log"
	"strconv"
	"time"
)

type Client struct {
	config               *utils.ConfigMQTT
	instance             mqtt.Client
	baseTopic            string
	onConnectWatchValves map[string]chan bool
}

func NewMQTT(config *utils.Config) *Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", config.MQTT.BrokerHost, config.MQTT.BrokerPort))
	opts.SetClientID(config.MQTT.ClientID)
	opts.SetUsername(config.MQTT.Username)
	opts.SetPassword(config.MQTT.Password)

	onConnectWatchValves := make(map[string]chan bool, 1)
	opts.OnConnect = func(client mqtt.Client) {
		log.Info().Msg("MQTT Connected")
		for _, onConnectWatchValve := range onConnectWatchValves {
			onConnectWatchValve <- true
		}
	}
	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		log.Err(err).Msgf("MQTT broker connection lost")
	}

	opts.ConnectRetryInterval = 5 * time.Second

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	return &Client{
		config:               config.MQTT,
		instance:             client,
		baseTopic:            config.MQTT.BaseTopic,
		onConnectWatchValves: onConnectWatchValves,
	}
}

func (c *Client) WatchValve(topicID string, callback func(action models.ValveAction)) {
	for {
		// https://www.home-assistant.io/integrations/button.mqtt/
		token := c.instance.Subscribe(topicID, 1, func(client mqtt.Client, message mqtt.Message) {
			value, err := strconv.Atoi(string(message.Payload()))
			if err != nil {
				log.Error().Err(err).Msgf("failed to parse valve value recieved")
			}
			if value > 50 {
				callback(models.VALVE_ACTION_OPEN)
			} else {
				callback(models.VALVE_ACTION_CLOSE)
			}

		})
		token.WaitTimeout(5 * time.Second)
		if !token.WaitTimeout(2 * time.Second) {
			log.Warn().Msgf("timeout to subscribe to topic %s", topicID)
		}
		if token.Error() != nil {
			log.Error().Err(token.Error()).Msgf("failed to subscribe to topic %s", topicID)
		}
		log.Info().Msgf("Subscribed to topic: %s", topicID)
		// wait for re-connection
		<-c.onConnectWatchValves[topicID]
	}
}

func (c *Client) PublishState(topic string, payload json.Marshaler) {
	log.Debug().Msgf("PublishState to topic: %s", topic)
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Msgf("failed to marshall %s", topic)
	}
	token := c.instance.Publish(topic, 0, false, jsonPayload)
	if !token.WaitTimeout(2 * time.Second) {
		log.Warn().Msgf("timeout to publish state to topic %s", topic)
	}
	if token.Error() != nil {
		log.Error().Err(token.Error()).Msgf("failed to publish state to topic %s", topic)
	}
}

func (c *Client) PublishAvailability(topicID string) {
	log.Debug().Msgf("PublishAvailability to topic: %s", topicID)
	token := c.instance.Publish(topicID, 0, false, "online")
	if !token.WaitTimeout(2 * time.Second) {
		log.Warn().Msgf("timeout to publish availability to topic %s", topicID)
	}
	if token.Error() != nil {
		log.Error().Err(token.Error()).Msgf("failed to publish availability to topic %s", topicID)
	}
}
