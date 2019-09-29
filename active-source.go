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
	cec            *Cec
	mqtt           *Mqtt
	activeSource   *Device
	devices        map[gocec.LogicalAddress]*Device
	monitor        *Monitor
	allowedSources map[gocec.LogicalAddress]bool
}

func InitAcitveSourceBridge(container *Container) {
	cec := container.Get("cec").(*Cec)
	mqtt := container.Get("mqtt").(*Mqtt)

	bridge := &ActiveSourceBridge{
		cec:            cec,
		mqtt:           mqtt,
		activeSource:   nil,
		devices:        make(map[gocec.LogicalAddress]*Device),
		allowedSources: make(map[gocec.LogicalAddress]bool),
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
		bridge.allowedSources[device.LogicalAddress] = true
	})

	if haBridge, ok := container.Get("home-assistant").(*HomeAssistantBridge); ok {
		log.Info("Enabling Home Assistant configuration for active source")
		devices.RegisterDeviceAddedHandler(func(device *Device) {
			haBridge.RegisterBinarySensor(device, "is_active_source")
		})
	}

	cec.RegisterMessageHandler(func(message gocec.Message) {
		log.WithFields(log.Fields{
			"message.source":      message.Source(),
			"message.destination": message.Destination(),
			"message.opcode":      message.Opcode(),
			"message.raw":         []byte(message),
		}).Debug("Restarting active source monitor")

		bridge.monitor.Reset()
	}, gocec.OpcodeActiveSource, gocec.OpcodeSetStreamPath)

	cec.RegisterMessageHandler(func(message gocec.Message) {
		log.WithFields(log.Fields{
			"message.source":      message.Source(),
			"message.destination": message.Destination(),
			"message.opcode":      message.Opcode(),
			"message.raw":         []byte(message),
		}).Debug("Updating active source based on power status")

		defer bridge.monitor.Reset()

		powerStatus := gocec.PowerStatus(message.Parameters()[0])
		bridge.allowedSources[message.Source()] = powerStatus != gocec.PowerStatusStandBy

		if bridge.activeSource == nil || message.Source() != bridge.activeSource.LogicalAddress {
			return
		}

		if powerStatus == gocec.PowerStatusStandBy {
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
		if allowed, ok := bridge.allowedSources[newSource.LogicalAddress]; ok && !allowed {
			if bridge.activeSource == nil {
				log.WithFields(log.Fields{
					"device.id":              newSource.Id,
					"device.logical_address": newSource.LogicalAddress,
				}).Trace("Skipping active source update because active device still is in standby")

				return
			}

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
		"to":   to,
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
