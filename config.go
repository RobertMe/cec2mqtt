package main

import (
	log "github.com/sirupsen/logrus"
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
	logContext := log.WithFields(log.Fields{
		"config_file": configPath + "config.yaml",
	})
	logContext.Info("Reading configuration")
	data, err := ioutil.ReadFile(configPath + "config.yaml")
	if nil != err {
		logContext.WithFields(log.Fields{
			"error": err,
		}).Error("Configuration could not be read")
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)

	if err != nil {
		logContext.WithFields(log.Fields{
			"error": err,
		}).Error("Configuration file could not be parsed as valid YAML")
		return nil, err
	}

	if config.HomeAssistant.Enable && config.HomeAssistant.DiscoveryPrefix == "" {
		log.Debug("Home assistant integration is enabled but discovery prefix is not set. Setting default.")
		config.HomeAssistant.DiscoveryPrefix = "homeassistant"
	} else {
		strings.Trim(config.HomeAssistant.DiscoveryPrefix, "/")
	}

	return &config, nil
}

func (config *Config) Save(configPath string) error {
	data, err := yaml.Marshal(config)

	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to convert configuration into YAML")

		return err
	}

	err = ioutil.WriteFile(configPath + "config.yaml", data, 0644)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"config_file": configPath + "config.yaml",
		}).Error("Failed to save configuration to file")

		return err
	}

	return nil
}
