package main

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
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

type Config struct {
	Mqtt MqttConfig
}

func ParseConfig(configPath string) (*Config, error) {
	data, err := ioutil.ReadFile(configPath)
	if nil != err {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)

	return &config, nil
}
