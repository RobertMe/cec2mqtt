package main

import (
	"fmt"
	"github.com/RobertMe/cec2mqtt/messages"
	"time"
)

func main() {
	fmt.Println("Started")
	cec, _ := InitialiseCec("")

	go listener(cec.Messages)
	cec.Start()

	for {
		time.Sleep(20 * time.Second)
	}
}

func listener(messages chan messages.Message) {
	for {
		select {
		case message := <- messages:
			fmt.Printf("%s => %s\n", message.MqttPath(), message.Value())
		}
	}
}
