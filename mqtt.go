package main

import (
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"strings"
)

type Mqtt struct {
	client mqtt.Client
	config *MqttConfig
}

type MessageHandler func (payload []byte)

func ConnectMqtt(config *Config) (*Mqtt, error) {
	mqttConfig := config.Mqtt
	options := mqtt.NewClientOptions()

	options.AddBroker(mqttConfig.Host)

	if mqttConfig.Username != "" {
		options.SetUsername(mqttConfig.Username)
	}

	if mqttConfig.Password != "" {
		options.SetPassword(mqttConfig.Password)
	}

	if mqttConfig.StateTopic != "" {
		if mqttConfig.WillMessage != "" {
			options.SetWill(mqttConfig.StateTopic, mqttConfig.WillMessage, 0, true)
		}

		if mqttConfig.BirthMessage != "" {
			options.SetOnConnectHandler(func(client mqtt.Client) {
				client.Publish(mqttConfig.StateTopic, 0, true, mqttConfig.BirthMessage)
			})
		}
	}

	client := mqtt.NewClient(options)

	connToken := client.Connect()

	connToken.Wait()

	return &Mqtt{
		client: client,
		config: &mqttConfig,
	}, nil
}

func (mqtt *Mqtt) BuildTopic(device *Device, suffix string) string {
	topic := strings.Builder{}
	fmt.Fprintf(&topic,"%s/%s/%s", mqtt.config.BaseTopic, device.Config.MqttTopic, suffix)
	return topic.String()
}

func (mqtt *Mqtt) Publish(topic string, qos byte, retained bool, payload interface{}) {
	mqtt.client.Publish(topic, qos, retained, payload)
}

func (m *Mqtt) Subscribe(topic string, qos byte, callback MessageHandler) {
	m.client.Subscribe(topic, qos, func(_ mqtt.Client, message mqtt.Message) {
		callback(message.Payload())
	})
}
