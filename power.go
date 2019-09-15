package main

import (
	"github.com/RobertMe/gocec"
	"sync"
	"time"
)

func init() {
	initializers = append(initializers, InitPowerBridge)
}

type PowerBridge struct {
	cec *Cec
	mqtt *Mqtt

	monitors map[gocec.LogicalAddress]*Monitor
	monitorsMutex sync.Mutex

	states map[gocec.LogicalAddress]string
	statesMutex sync.Mutex
}

func InitPowerBridge(devices *DeviceRegistry, cec *Cec, mqtt *Mqtt) {
	bridge := PowerBridge{
		cec:  cec,
		mqtt: mqtt,

		monitors: make(map[gocec.LogicalAddress]*Monitor),
		states: make(map[gocec.LogicalAddress]string),
	}

	devices.RegisterDeviceAddedHandler(func(device *Device) {
		bridge.statesMutex.Lock()
		bridge.monitorsMutex.Lock()
		defer bridge.statesMutex.Unlock()
		defer bridge.monitorsMutex.Unlock()
		bridge.states[device.LogicalAddress] = "unknown"
		bridge.monitors[device.LogicalAddress] = CreateMonitor(
			bridge.createStarter(device),
			bridge.createRunner(device),
			5 * time.Minute,
			5 * time.Second,
			time.Minute,
		)
	})

	cec.RegisterMessageHandler(func (message gocec.Message) {
		bridge.setPowerStatus(message.Source(), bridge.cec.connection.GetPowerStatus(message.Source()))
	}, gocec.OpcodeReportPowerStatus)

	cec.RegisterMessageHandler(func (message gocec.Message) {
		bridge.MonitorPower(message.Source())
	}, gocec.OpcodeSetSystemAudioMode)

	cec.RegisterMessageHandler(func (message gocec.Message) {
		for address, _ := range bridge.states {
			bridge.MonitorPower(address)
		}
	}, gocec.OpcodeStandby, gocec.OpcodeActiveSource)
}

func (bridge *PowerBridge) MonitorPower(address gocec.LogicalAddress) {
	bridge.monitorsMutex.Lock()
	defer bridge.monitorsMutex.Unlock()

	if monitor, ok := bridge.monitors[address]; ok {
		monitor.Reset()
	}
}

func (bridge *PowerBridge) setPowerStatus(address gocec.LogicalAddress, status gocec.PowerStatus) {
	var value string
	switch status {
	case gocec.PowerStatusOn, gocec.PowerStatusTransitionToStandby:
		value = "on"
	case gocec.PowerStatusStandBy, gocec.PowerStatusTransitionToOn:
		value = "off"
	default:
		value = "unknown"
	}

	bridge.statesMutex.Lock()
	defer bridge.statesMutex.Unlock()
	if bridge.states[address] == value {
		return
	}

	bridge.states[address]  = value
	go bridge.mqtt.Publish(bridge.mqtt.BuildTopic(address, "power"), 0, false, value)
}

func (bridge *PowerBridge) createStarter(device *Device) Starter {
	source := gocec.DeviceTV

	if device.LogicalAddress == gocec.DeviceTV {
		source = gocec.DeviceBroadcast
	}

	message := gocec.Message{byte(source) + byte(device.LogicalAddress), byte(gocec.OpcodeGiveDevicePowerStatus)}

	return func() {
		bridge.cec.Transmit(message)
	}
}

func (bridge *PowerBridge) createRunner(device *Device) Runner {
	return func() {
		bridge.setPowerStatus(device.LogicalAddress, bridge.cec.connection.GetPowerStatus(device.LogicalAddress))
	}
}
