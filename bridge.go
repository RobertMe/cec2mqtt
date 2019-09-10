package main

type Bridge struct {
	config *Config
	cec *Cec
	mqtt *Mqtt
}

func NewBridge(config *Config, cec *Cec, mqtt *Mqtt) *Bridge {
	bridge := &Bridge{
		config:config,
		cec: cec,
		mqtt: mqtt,
	}

	go bridge.handleMessages()

	return bridge
}

func (bridge *Bridge) handleMessages() {
	for {
		message := <- bridge.cec.Messages

		switch message.(type) {
		default:
			bridge.mqtt.Publish(message.MqttPath(), 0, false, message.Value())
		}
	}
}
