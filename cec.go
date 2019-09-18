package main

import (
	"errors"
	"github.com/RobertMe/gocec"
	"strings"
)

type MessageReceivedHandler func(message gocec.Message)

type Cec struct {
	connection *gocec.Connection
	adapter gocec.Adapter

	devices *DeviceRegistry
	messageReceivedHandlers map[gocec.Opcode][]MessageReceivedHandler
}

type CecDeviceDescription struct {
	logicalAddress gocec.LogicalAddress
	physicalAddress gocec.PhysicalAddress
	vendor gocec.Vendor
	OSD string
}


func InitialiseCec(devices *DeviceRegistry, path string) (*Cec, error) {
	config := gocec.NewConfiguration("cec2mqtt", false)

	config.SetMonitorOnly(false)

	cec := &Cec{
		devices: devices,
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

	adapterAddress, _ := cec.connection.GetAdapterAddress()
	addresses := cec.connection.ActiveDevices()

	for _, address := range addresses {
		// Don't register a device for the CEC adapter
		if address != adapterAddress {
			_ = cec.GetDevice(address)
		}
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

	creator := func() *CecDeviceDescription {
		return &CecDeviceDescription{
			logicalAddress:  address,
			physicalAddress: cec.connection.GetPhysicalAddress(address),
			vendor:          cec.connection.GetVendor(address),
			OSD:			 cec.connection.GetOSDName(address),
		}
	}

	return cec.devices.GetByCecDevice(address, creator)
}

func (cec *Cec) Transmit(message gocec.Message) {
	cec.connection.Transmit(message)
}
