package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

type Initializer func(container *Container)

var initializers = make([]Initializer, 0)

func main() {
	fmt.Println("Starting cec2mqtt")

	container := NewContainer()

	config, err := ParseConfig("/etc/cec2mqtt.yaml")

	if nil != err {
		panic(err)
	}

	container.Register("config", config)

	devices := NewDeviceRegistry(config)
	container.Register("devices", devices)

	mqtt, err := ConnectMqtt(config)

	if nil != err {
		panic(err)
	}

	container.Register("mqtt", mqtt)

	cec, err := InitialiseCec(devices, "")

	if nil != err {
		panic(err)
	}

	container.Register("cec", cec)

	for _, initializer := range initializers {
		initializer(container)
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
	config.Save("/etc/cec2mqtt.yaml")
}
