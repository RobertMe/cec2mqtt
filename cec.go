package main

import (
	"errors"
	"github.com/RobertMe/cec2mqtt/messages"
	"github.com/RobertMe/gocec"
	"strings"
)

type Cec struct {
	connection *gocec.Connection
	devices map[gocec.LogicalAddress]*Device

	Messages chan messages.Message
}

func InitialiseCec(path string) (*Cec, error) {
	config := gocec.NewConfiguration("cec2mqtt", false)

	config.SetMonitorOnly(false)

	cec := &Cec{
		devices: make(map[gocec.LogicalAddress]*Device),
		Messages: make(chan messages.Message, 10),
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

	connection.Open(adapter)

	return cec, nil
}

func (cec *Cec) Start() {
	addresses := cec.connection.ActiveDevices()

	for _, address := range addresses {
		_ = cec.getDevice(address)
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

	device := cec.getDevice(message.Source())

	switch message.Opcode() {
	case gocec.OpcodeReportPowerStatus:
		cec.handlePowerMessage(device, &message)
	case gocec.OpcodeStandby:
		cec.handleStandbyMessage(&message)
	case gocec.OpcodeActiveSource:
		cec.handleActiveSourceMessage(&message)
	case gocec.OpcodeSetSystemAudioMode:
		cec.handleSetAudioMode(device, &message)
	}
}

func (cec *Cec) getDevice(address gocec.LogicalAddress) *Device {
	if address == gocec.DeviceBroadcast {
		return nil
	}

	device, ok := cec.devices[address]
	if !ok {
		device = NewDevice(address, cec)
		cec.devices[address] = device
	}
	return device
}

func (cec *Cec) handlePowerMessage(device *Device, message *gocec.Message) {
	device.SetPowerStatus(convertPowerStatus(gocec.PowerStatus(message.Parameters()[0])))
}

func (cec *Cec) handleStandbyMessage(message *gocec.Message) {
	for _, device := range cec.devices {
		go device.MonitorPowerStatus()
	}
}

func (cec *Cec) handleActiveSourceMessage(message *gocec.Message) {
	for _, device := range cec.devices {
		go device.MonitorPowerStatus()
	}
}

func (cec *Cec) handleSetAudioMode(device *Device, message *gocec.Message) {
	go device.MonitorPowerStatus()
}
