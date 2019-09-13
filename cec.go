package main

import (
	"errors"
	"github.com/RobertMe/gocec"
	"strings"
)

type DeviceAddedHandler func(device *Device)

type MessageReceivedHandler func(message gocec.Message)

type Cec struct {
	connection *gocec.Connection
	adapter gocec.Adapter

	devices map[gocec.LogicalAddress]*Device

	deviceAddedHandlers []DeviceAddedHandler
	messageReceivedHandlers map[gocec.Opcode][]MessageReceivedHandler
}

func InitialiseCec(path string) (*Cec, error) {
	config := gocec.NewConfiguration("cec2mqtt", false)

	config.SetMonitorOnly(false)

	cec := &Cec{
		devices: make(map[gocec.LogicalAddress]*Device),
		deviceAddedHandlers: make([]DeviceAddedHandler, 0),
		messageReceivedHandlers: make(map[gocec.Opcode][]MessageReceivedHandler),
	}
	config.SetLogCallback(cec.handleLogMessage)

	connection, err := gocec.NewConnection(config)
	if err != nil {
		return nil, err
	}

	var adapter gocec.Adapter
	adapters := connection.FindAdapters()
	if len(path) == 0 {
		adapter = adapters[0]
	} else {
		var found bool
		for _, adapter = range adapters {
			if adapter.Path == path {
				found = true
			}
		}

		if !found {
			return nil, errors.New("")
		}
	}

	cec.connection = connection
	cec.adapter = adapter

	return cec, nil
}

func (cec *Cec) RegisterDeviceAddedHandler(handler DeviceAddedHandler) {
	cec.deviceAddedHandlers = append(cec.deviceAddedHandlers, handler)
}

func (cec *Cec) RegisterMessageHandler(handler MessageReceivedHandler, opcodes ...gocec.Opcode) {
	for _, opcode := range opcodes {
		handlers, ok := cec.messageReceivedHandlers[opcode]
		if !ok {
			cec.messageReceivedHandlers[opcode] = []MessageReceivedHandler{handler}
		} else {
			cec.messageReceivedHandlers[opcode] = append(handlers, handler)
		}
	}
}

func (cec *Cec) Start() {
	cec.connection.Open(cec.adapter)

	addresses := cec.connection.ActiveDevices()

	for _, address := range addresses {
		_ = cec.GetDevice(address)
	}
}

func (cec *Cec) handleLogMessage(logMessage *gocec.LogMessage) {
	if logMessage.Level != gocec.LogLevelTraffic {
		return
	}

	if !strings.HasPrefix(logMessage.Message, ">> ") {
		return
	}

	message, _ := gocec.ParseMessage(logMessage.Message[3:])

	device := cec.GetDevice(message.Source())

	_ = device

	if handlers, ok := cec.messageReceivedHandlers[message.Opcode()]; ok {
		for _, handler := range handlers {
			handler(message)
		}
	}
}

func (cec *Cec) GetDevice(address gocec.LogicalAddress) *Device {
	if address == gocec.DeviceBroadcast {
		return nil
	}

	device, ok := cec.devices[address]
	if !ok {
		device = NewDevice(address, cec)
		cec.devices[address] = device

		for _, handler := range cec.deviceAddedHandlers {
			handler(device)
		}
	}
	return device
}

func (cec *Cec) Transmit(message gocec.Message) {
	cec.connection.Transmit(message)
}
