package main

import (
	"fmt"
	"github.com/RobertMe/gocec"
	"github.com/eclipse/paho.mqtt.golang"
	"strings"
)

type Mqtt struct {
	client mqtt.Client
	config *MqttConfig
}

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

func (mqtt *Mqtt) BuildTopic(address gocec.LogicalAddress, suffix string) string {
	topic := strings.Builder{}
	fmt.Fprintf(&topic,"%s/%d/%s", mqtt.config.BasePath, address, suffix)
	return topic.String()
}

func (mqtt *Mqtt) Publish(topic string, qos byte, retained bool, payload interface{}) {
	mqtt.client.Publish(topic, qos, retained, payload)
}
