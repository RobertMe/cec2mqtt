package main

import (
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
	BaseTopic    string `yaml:"base_topic"`
}

type HomeAssistantConfig struct {
	Enable          bool   `yaml:"enable"`
	DiscoveryPrefix string `yaml:"discovery_prefix"`
}

type Config struct {
	Mqtt          MqttConfig
	HomeAssistant HomeAssistantConfig `yaml:"home_assistant"`
}

func ParseConfig(configPath string) (*Config, error) {
	data, err := ioutil.ReadFile(configPath + "config.yaml")
	if nil != err {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)

	if err != nil {
		return nil, err
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

	return ioutil.WriteFile(configPath + "config.yaml", data, 0644)
}
