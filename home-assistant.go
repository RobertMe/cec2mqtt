package main

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"strings"
)

func init() {
	RegisterInitializer(100, InitHomeAssistantBridge)
}

type HomeAssistantBirthHandler func()

type HomeAssistantBridge struct {
	discoveryPrefix string
	mqtt            *Mqtt
	config          *Config
	birthHandlers   []HomeAssistantBirthHandler
}

func InitHomeAssistantBridge(container *Container) {
	config := container.Get("config").(*Config)
	if !config.HomeAssistant.Enable {
		log.Info("Home assistant integration is not enabled, skipping")
		return
	}

	mqtt := container.Get("mqtt").(*Mqtt)
	bridge := &HomeAssistantBridge{
		discoveryPrefix: config.HomeAssistant.DiscoveryPrefix,
		mqtt:            mqtt,
		config:          config,
		birthHandlers:   make([]HomeAssistantBirthHandler, 0),
	}

	mqtt.Subscribe(config.HomeAssistant.BirthTopic, 0, func (payload []byte) {
		log.WithFields(log.Fields{
			"payload": string(payload),
		}).Debug("Received message on Home Assistant birth topic")

		if string(payload) == config.HomeAssistant.BirthPayload {
			log.Info("Received Home Assistant birth message")
			for _, handler := range bridge.birthHandlers {
				handler()
			}
		}
	})

	container.Register("home-assistant", bridge)
}

func (bridge *HomeAssistantBridge) RegisterBirthHandler(handler HomeAssistantBirthHandler) {
	bridge.birthHandlers = append(bridge.birthHandlers, handler)
}

func (bridge *HomeAssistantBridge) RegisterSwitch(device *Device, property string) {
	topic := strings.Builder{}
	fmt.Fprintf(&topic, "%s/switch/%s/%s/config", bridge.discoveryPrefix, device.Id, property)

	config := bridge.createConfig(device, property)
	config["command_topic"] = bridge.mqtt.BuildTopic(device, property+"/set")
	config["payload_on"] = "on"
	config["payload_off"] = "off"

	encoded, err := json.Marshal(config)
	if err != nil {
		log.WithFields(log.Fields{
			"device.id": device.Config.Id,
			"property":  property,
			"config":    config,
			"error":     err,
		}).Error("Failed to convert switch configuration to JSON")

		return
	}

	log.WithFields(log.Fields{
		"device.id": device.Config.Id,
		"property":  property,
		"config":    string(encoded),
	}).Info("Registering switch in Home Assistant")

	bridge.mqtt.Publish(topic.String(), 0, true, encoded)
}

func (bridge *HomeAssistantBridge) RegisterBinarySensor(device *Device, property string) {
	topic := strings.Builder{}
	fmt.Fprintf(&topic, "%s/binary_sensor/%s/%s/config", bridge.discoveryPrefix, device.Id, property)

	config := bridge.createConfig(device, property)
	config["payload_on"] = "on"
	config["payload_off"] = "off"

	encoded, err := json.Marshal(config)
	if err != nil {
		log.WithFields(log.Fields{
			"device.id": device.Config.Id,
			"property":  property,
			"config":    config,
			"error":     err,
		}).Error("Failed to convert binary sensor configuration to JSON")

		return
	}

	log.WithFields(log.Fields{
		"device.id": device.Config.Id,
		"property":  property,
		"config":    string(encoded),
	}).Info("Registering binary switch in Home Assistant")

	bridge.mqtt.Publish(topic.String(), 0, true, encoded)
}

func (bridge *HomeAssistantBridge) createConfig(device *Device, property string) map[string]interface{} {
	config := map[string]interface{}{
		"state_topic":           bridge.mqtt.BuildTopic(device, property),
		"name":                  device.CecDevice.OSD + "_" + property,
		"unique_id":             device.Id + "_" + property + "_" + bridge.config.Mqtt.BaseTopic,
	}

	if bridge.config.Mqtt.StateTopic != "" {
		config["availability_topic"] = bridge.config.Mqtt.StateTopic
		config["payload_available"] = bridge.config.Mqtt.BirthMessage
		config["payload_not_available"] = bridge.config.Mqtt.WillMessage
	}

	deviceConfig := map[string]interface{}{
		"identifiers":  []string{"cec2mqtt_" + device.Id},
		"name":         device.CecDevice.OSD,
		"sw_version":   "Cec2Mqtt " + BuildVersion,
		"manufacturer": device.CecDevice.vendor.String(),
	}

	config["device"] = deviceConfig

	return config
}
