package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

func init() {
	RegisterInitializer(100, InitHomeAssistantBridge)
}

type HomeAssistantBridge struct {
	discoveryPrefix string
	mqtt            *Mqtt
	config          *Config
}

func InitHomeAssistantBridge(container *Container) {
	config := container.Get("config").(*Config)
	if !config.HomeAssistant.Enable {
		return
	}

	bridge := &HomeAssistantBridge{
		discoveryPrefix: config.HomeAssistant.DiscoveryPrefix,
		mqtt:            container.Get("mqtt").(*Mqtt),
		config:          config,
	}

	container.Register("home-assistant", bridge)
}

func (bridge *HomeAssistantBridge) RegisterSwitch(device *Device, property string) {
	topic := strings.Builder{}
	fmt.Fprintf(&topic, "%s/switch/%s/%s/config", bridge.discoveryPrefix, device.Id, property)

	config := bridge.createConfig(device, property)
	config["command_topic"] = bridge.mqtt.BuildTopic(device, property + "/set")

	if encoded, err := json.Marshal(config); err == nil {
		bridge.mqtt.Publish(topic.String(), 0, true, encoded)
	}
}

func (bridge *HomeAssistantBridge) RegisterBinarySensor(device *Device, property string) {
	topic := strings.Builder{}
	fmt.Fprintf(&topic, "%s/binary_sensor/%s/%s/config", bridge.discoveryPrefix, device.Id, property)

	config := bridge.createConfig(device, property)

	if encoded, err := json.Marshal(config); err == nil {
		bridge.mqtt.Publish(topic.String(), 0, true, encoded)
	}
}

func (bridge *HomeAssistantBridge) createConfig(device *Device, property string) map[string]interface{} {
	config := map[string]interface{}{
		"state_topic": bridge.mqtt.BuildTopic(device, property),
		"name":        device.CecDevice.OSD + "_" + property,
		"unique_id":   device.Id + "_" + property + "_" + bridge.config.Mqtt.BaseTopic,
		"availability_topic": bridge.config.Mqtt.StateTopic,
	}

	deviceConfig := map[string]interface{}{
		"identifiers": []string{"cec2mqtt_" + device.Id},
		"name": device.CecDevice.OSD,
		"sw_version": "Cec2Mqtt " + BuildVersion,
		"manufacturer": device.CecDevice.vendor.String(),
	}

	config["device"] = deviceConfig

	return config
}
