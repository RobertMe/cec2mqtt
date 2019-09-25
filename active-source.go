package main

import (
	"github.com/RobertMe/gocec"
	log "github.com/sirupsen/logrus"
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

	if haBridge, ok := container.Get("home-assistant").(*HomeAssistantBridge); ok {
		log.Info("Enabling Home Assistant configuration for active source")
		devices.RegisterDeviceAddedHandler(func(device *Device) {
			haBridge.RegisterBinarySensor(device, "is_active_source")
		})
	}

	cec.RegisterMessageHandler(func(message gocec.Message) {
		log.WithFields(log.Fields{
			"message.source": message.Source(),
			"message.destination": message.Destination(),
			"message.opcode": message.Opcode(),
			"message.raw": []byte(message),
		}).Debug("Restarting active source monitor")

		bridge.monitor.Reset()
	}, gocec.OpcodeActiveSource, gocec.OpcodeSetStreamPath)

	cec.RegisterMessageHandler(func(message gocec.Message) {
		log.WithFields(log.Fields{
			"message.source": message.Source(),
			"message.destination": message.Destination(),
			"message.opcode": message.Opcode(),
			"message.raw": []byte(message),
		}).Debug("Updating active source based on power status")

		bridge.monitor.Reset()

		if bridge.activeSource == nil ||  message.Source() != bridge.activeSource.LogicalAddress {
			return
		}

		if gocec.PowerStatus(message.Parameters()[0]) == gocec.PowerStatusStandBy {
			bridge.updateActiveSource(nil)
		}
	}, gocec.OpcodeReportPowerStatus)

	cec.RegisterMessageHandler(func(message gocec.Message) {
		if message.Source() == gocec.DeviceTV {
			log.Debug("Setting active source to nil because TV is in standby")
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

	if newSource != nil {
		if bridge.cec.connection.GetPowerStatus(newSource.LogicalAddress) == gocec.PowerStatusStandBy {
			newSource = nil
		}
	}

	from, to := "", ""
	if bridge.activeSource != nil {
		from = bridge.activeSource.Id
	}
	if newSource != nil {
		to = newSource.Id
	}

	log.WithFields(log.Fields{
		"from": from,
		"to": to,
	}).Info("Updating active source")

	mqtt := bridge.mqtt
	if bridge.activeSource != nil {
		log.WithFields(log.Fields{
			"device.id": bridge.activeSource.Id,
		}).Debug("Setting device as inactive source")

		mqtt.Publish(mqtt.BuildTopic(bridge.activeSource, "is_active_source"), 0, false, "off")
	}

	if newSource != nil {
		log.WithFields(log.Fields{
			"device.id": newSource.Id,
		}).Debug("Setting device as active source")

		mqtt.Publish(mqtt.BuildTopic(newSource, "is_active_source"), 0, false, "on")
	}

	bridge.activeSource = newSource
}

func (bridge *ActiveSourceBridge) checkActiveSource() {
	address := bridge.cec.connection.GetActiveSource()

	var newSource *Device = nil
	newSourceId := ""
	if address != gocec.DeviceUnknown {
		var ok bool
		if newSource, ok = bridge.devices[address]; ok {
			newSourceId = newSource.Id
		}
	}

	log.WithFields(log.Fields{
		"active_source.logical_address": address,
		"active_source.id":              newSourceId,
	}).Trace("Updating active source from monitor")

	bridge.updateActiveSource(newSource)
}
