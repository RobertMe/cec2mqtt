package main

import (
	"github.com/RobertMe/cec2mqtt/messages"
	"github.com/RobertMe/gocec"
	"time"
)

type Device struct {
	cec *Cec
	LogicalAddress gocec.LogicalAddress

	OSD string

	powerTicker *time.Ticker
	PowerState messages.PowerState
}

func NewDevice(address gocec.LogicalAddress, cec *Cec) *Device {
	device := &Device{
		LogicalAddress: address,
		cec: cec,
	}

	go device.MonitorPowerStatus()

	return device
}

func (device *Device) MonitorPowerStatus() {
	if device.powerTicker != nil {
		device.powerTicker.Stop()
	}

	source := gocec.DeviceTV

	if device.LogicalAddress == gocec.DeviceTV {
		source = gocec.DeviceBroadcast
	}

	message := gocec.Message{byte(source) + byte(device.LogicalAddress), byte(gocec.OpcodeGiveDevicePowerStatus)}
	device.cec.connection.Transmit(message)
	device.powerTicker = time.NewTicker(5 * time.Second)
	defer func() {
		device.powerTicker.Stop()
		device.powerTicker = nil
	}()

	updatePower := func() {
		device.SetPowerStatus(convertPowerStatus(device.cec.connection.GetPowerStatus(device.LogicalAddress)))
	}

	updatePower()
	for i := 0; i < 12; i++ {
		<- device.powerTicker.C
		updatePower()
	}
}

func (device *Device) SetPowerStatus(state messages.PowerState) {
	if device.PowerState == state {
		return
	}

	device.PowerState = state
	device.cec.Messages <- &messages.PowerMessage{
		Address: messages.Address(device.LogicalAddress),
		State:   state,
	}
}
