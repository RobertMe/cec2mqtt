package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

type Initializer func(deviceRegistry *DeviceRegistry, cec *Cec, mqtt *Mqtt)

var initializers = make([]Initializer, 0)

func main() {
	fmt.Println("Starting cec2mqtt")

	config, err := ParseConfig("/etc/cec2mqtt.yaml")

	if nil != err {
		panic(err)
	}

	devices := NewDeviceRegistry(config)

	mqtt, err := ConnectMqtt(config)

	if nil != err {
		panic(err)
	}

	cec, _ := InitialiseCec(devices, "")

	for _, initializer := range initializers {
		initializer(devices, cec, mqtt)
	}

	cec.Start()

	signals := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<- signals
		done <- true
	}()

	fmt.Println("Cec2mqtt started")
	<- done
	fmt.Println("Exiting")
}
