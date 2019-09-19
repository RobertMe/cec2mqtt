package main

import (
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"strings"
)

type MqttConfig struct {
	Host         string `yaml:"host"`
	Username     string
	Password     string
	StateTopic   string `yaml:"state_topic"`
	BirthMessage string `yaml:"birth_message"`
	WillMessage  string `yaml:"will_message"`
	BasePath     string `yaml:"base_path"`
}

type DeviceConfig struct {
	Id              string `yaml:"id"`
	PhysicalAddress string `yaml:"physical_address"`
	VendorId        int    `yaml:"vendor_id"`
	OSD             string `yaml:"osd"`
	MqttTopic       string `yaml:"mqtt_topic"`
}

type HomeAssistantConfig struct {
	Enable          bool   `yaml:"enable"`
	DiscoveryPrefix string `yaml:"discovery_prefix"`
}

type Config struct {
	Mqtt          MqttConfig
	Devices       map[string]*DeviceConfig
	HomeAssistant HomeAssistantConfig `yaml:"home_assistant"`
}

func ParseConfig(configPath string) (*Config, error) {
	data, err := ioutil.ReadFile(configPath)
	if nil != err {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)

	if err != nil {
		return nil, err
	}

	if config.Devices == nil {
		config.Devices = make(map[string]*DeviceConfig)
	}

	if config.HomeAssistant.Enable && config.HomeAssistant.DiscoveryPrefix == "" {
		config.HomeAssistant.DiscoveryPrefix = "homeassistant"
	} else {
		strings.Trim(config.HomeAssistant.DiscoveryPrefix, "/")
	}

	return &config, nil
}

func (config *Config) Save(configPath string) error {
	data, err := yaml.Marshal(config)

	if err != nil {
		return err
	}

	return ioutil.WriteFile(configPath, data, 0644)
}

func (config *Config) FindDevice(physicalAddress string, vendorId int, name string) *DeviceConfig {
	var option *DeviceConfig

	for _, device := range config.Devices {
		if device.PhysicalAddress == physicalAddress && device.VendorId == vendorId {
			if device.OSD == name {
				// Exact match so must be it
				return device
			}

			if option != nil {
				option = nil
				break
			}

			option = device
		} else if device.VendorId == vendorId && device.OSD == name {
			if option != nil {
				option = nil
				break
			}

			option = device
		}
	}

	if option != nil {
		option.PhysicalAddress = physicalAddress
		option.VendorId = vendorId
		option.OSD = name
		return option
	}

	var id string
	for {
		id = uuid.New().String()

		if _, ok := config.Devices[id]; !ok {
			break
		}
	}

	device := &DeviceConfig{
		Id:              id,
		PhysicalAddress: physicalAddress,
		VendorId:        vendorId,
		OSD:             name,
		MqttTopic:       name,
	}

	config.Devices[id] = device

	return device
}
