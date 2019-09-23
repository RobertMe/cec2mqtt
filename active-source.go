package main

import (
	"github.com/RobertMe/gocec"
	"time"
)

func init() {
	RegisterInitializer(0, InitAcitveSourceBridge)
}

type ActiveSourceBridge struct {
	cec          *Cec
	mqtt         *Mqtt
	activeSource *Device
	devices      map[gocec.LogicalAddress]*Device
	monitor      *Monitor
}

func InitAcitveSourceBridge(container *Container) {
	cec := container.Get("cec").(*Cec)
	mqtt := container.Get("mqtt").(*Mqtt)

	bridge := &ActiveSourceBridge{
		cec:          cec,
		mqtt:         mqtt,
		activeSource: nil,
		devices:      make(map[gocec.LogicalAddress]*Device),
	}

	bridge.monitor = CreateMonitor(
		func() {},
		bridge.checkActiveSource,
		10*time.Minute,
		10*time.Second,
		1*time.Minute,
	)

	devices := container.Get("devices").(*DeviceRegistry)

	devices.RegisterDeviceAddedHandler(func(device *Device) {
		bridge.devices[device.LogicalAddress] = device
	})

	cec.RegisterMessageHandler(func(message gocec.Message) {
		bridge.monitor.Reset()
	}, gocec.OpcodeActiveSource, gocec.OpcodeSetStreamPath, gocec.OpcodeReportPowerStatus)

	cec.RegisterMessageHandler(func(message gocec.Message) {
		if message.Source() == gocec.DeviceTV {
			bridge.updateActiveSource(nil)
		}
	}, gocec.OpcodeStandby)

	container.Register("active-source", bridge)
}

func (bridge *ActiveSourceBridge) updateActiveSource(newSource *Device) {
	if bridge.activeSource != nil && newSource == bridge.activeSource {
		return
	}

	if newSource == nil && bridge.activeSource == nil {
		return
	}

	mqtt := bridge.mqtt
	if bridge.activeSource != nil {
		mqtt.Publish(mqtt.BuildTopic(bridge.activeSource, "is_active_source"), 0, false, "off")
	}

	if newSource != nil {
		mqtt.Publish(mqtt.BuildTopic(newSource, "is_active_source"), 0, false, "on")
	}

	bridge.activeSource = newSource
}

func (bridge *ActiveSourceBridge) checkActiveSource() {
	address := bridge.cec.connection.GetActiveSource()

	var newSource *Device = nil
	if address != gocec.DeviceUnknown {
		newSource = bridge.devices[address]
	}

	bridge.updateActiveSource(newSource)
}
