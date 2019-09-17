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

	monitors map[string]*Monitor
	monitorsMutex sync.Mutex

	states map[string]string
	statesMutex sync.Mutex
}

func InitPowerBridge(devices *DeviceRegistry, cec *Cec, mqtt *Mqtt) {
	bridge := PowerBridge{
		cec:  cec,
		mqtt: mqtt,

		monitors: make(map[string]*Monitor),
		states: make(map[string]string),
	}

	devices.RegisterDeviceAddedHandler(func(device *Device) {
		bridge.statesMutex.Lock()
		bridge.monitorsMutex.Lock()
		defer bridge.statesMutex.Unlock()
		defer bridge.monitorsMutex.Unlock()
		bridge.states[device.Id] = "unknown"
		bridge.monitors[device.Id] = CreateMonitor(
			bridge.createStarter(device),
			bridge.createRunner(device),
			5 * time.Minute,
			5 * time.Second,
			time.Minute,
		)
	})

	getDevice := func(address gocec.LogicalAddress) *Device {
		return devices.FindByLogicalAddress(address)
	}

	cec.RegisterMessageHandler(func (message gocec.Message) {
		bridge.setPowerStatus(getDevice(message.Source()), bridge.cec.connection.GetPowerStatus(message.Source()))
	}, gocec.OpcodeReportPowerStatus)

	cec.RegisterMessageHandler(func (message gocec.Message) {
		bridge.MonitorPower(getDevice(message.Source()).Id)
	}, gocec.OpcodeSetSystemAudioMode)

	cec.RegisterMessageHandler(func (message gocec.Message) {
		for deviceId, _ := range bridge.states {
			bridge.MonitorPower(deviceId)
		}
	}, gocec.OpcodeStandby, gocec.OpcodeActiveSource)
}

func (bridge *PowerBridge) MonitorPower(deviceId string) {
	bridge.monitorsMutex.Lock()
	defer bridge.monitorsMutex.Unlock()

	if monitor, ok := bridge.monitors[deviceId]; ok {
		monitor.Reset()
	}
}

func (bridge *PowerBridge) setPowerStatus(device *Device, status gocec.PowerStatus) {
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
	if bridge.states[device.Id] == value {
		return
	}

	bridge.states[device.Id]  = value
	go bridge.mqtt.Publish(bridge.mqtt.BuildTopic(device, "power"), 0, false, value)
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
		bridge.setPowerStatus(device, bridge.cec.connection.GetPowerStatus(device.LogicalAddress))
	}
}
