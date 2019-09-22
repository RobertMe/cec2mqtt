package main

import (
	"fmt"
	"os"
	"os/signal"
	"sort"
	"syscall"
)

const (
	BuildVersion = "0.0.1"
)

type Initializer func(container *Container)

var initializers = make(map[int][]Initializer, 0)

func RegisterInitializer(priority int, initializer Initializer) {
	if _, ok := initializers[priority]; ok {
		initializers[priority] = append(initializers[priority], initializer)
	} else {
		initializers[priority] = []Initializer{initializer}
	}
}

func runInitializers(container *Container) {
	priorities := make([]int, len(initializers))
	i := 0
	for priority := range initializers {
		priorities[i] = priority
		i++
	}

	sort.Ints(priorities)
	for i := len(priorities) - 1; i >= 0; i-- {
		for _, initializer := range initializers[priorities[i]] {
			initializer(container)
		}
	}
}

func main() {
	fmt.Println("Starting cec2mqtt")

	container := NewContainer()

	config, err := ParseConfig("/data/cec2mqtt/")

	if nil != err {
		panic(err)
	}

	container.Register("config", config)

	devices := NewDeviceRegistry("/data/cec2mqtt/")
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

	runInitializers(container)

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
	config.Save("/data/cec2mqtt/")
	devices.Save("/data/cec2mqtt")
}
