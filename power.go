package main

import (
	"github.com/RobertMe/gocec"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

func init() {
	RegisterInitializer(0, InitPowerBridge)
}

type PowerState struct {
	state string
	published bool
}

type PowerBridge struct {
	cec  *Cec
	mqtt *Mqtt

	monitors      map[string]*Monitor
	monitorsMutex sync.Mutex

	states      map[string]*PowerState
	statesMutex sync.Mutex
}

func InitPowerBridge(container *Container) {
	cec := container.Get("cec").(*Cec)
	mqtt := container.Get("mqtt").(*Mqtt)
	bridge := PowerBridge{
		cec:  cec,
		mqtt: mqtt,

		monitors: make(map[string]*Monitor),
		states:   make(map[string]*PowerState),
	}

	container.Register("bridge.power", bridge)

	devices := container.Get("devices").(*DeviceRegistry)
	devices.RegisterDeviceAddedHandler(func(device *Device) {
		bridge.statesMutex.Lock()
		bridge.monitorsMutex.Lock()
		defer bridge.statesMutex.Unlock()
		defer bridge.monitorsMutex.Unlock()
		bridge.states[device.Id] = &PowerState{state: "unknown", published: false}
		bridge.monitors[device.Id] = CreateMonitor(
			bridge.createStarter(device),
			bridge.createRunner(device),
			5*time.Minute,
			5*time.Second,
			time.Minute,
		)

		log.WithFields(log.Fields{
			"device.id": device.Id,
		}).Debug("Subscribing to power change requests")

		mqtt.Subscribe(mqtt.BuildTopic(device, "power/set"), 0, func(payload []byte) {
			log.WithFields(log.Fields{
				"device.id": device.Id,
				"payload":   payload,
			})
			switch string(payload) {
			case "on":
				log.WithFields(log.Fields{
					"device.id": device.Id,
				}).Info("Powering device on as requested on MQTT")
				cec.connection.PowerOnDevice(device.LogicalAddress)
			case "off":
				log.WithFields(log.Fields{
					"device.id": device.Id,
				}).Info("Turning device into standby as requested on MQTT")
				cec.connection.StandByDevice(device.LogicalAddress)
			}
		})
	})

	if haBridge, ok := container.Get("home-assistant").(*HomeAssistantBridge); ok {
		log.Info("Enabling Home Assistant configuration for power")
		devices.RegisterDeviceAddedHandler(func(device *Device) {
			haBridge.RegisterSwitch(device, "power")
		})
	}

	getDevice := func(address gocec.LogicalAddress) *Device {
		return devices.FindByLogicalAddress(address)
	}

	cec.RegisterMessageHandler(func(message gocec.Message) {
		device := getDevice(message.Source())
		status := gocec.PowerStatus(message[2])

		log.WithFields(log.Fields{
			"device.id": device.Id,
			"status":    status,
		}).Debug("New power status received")

		bridge.setPowerStatus(device, status)
	}, gocec.OpcodeReportPowerStatus)

	cec.RegisterMessageHandler(func(message gocec.Message) {
		device := getDevice(message.Source())

		log.WithFields(log.Fields{
			"device.id": device.Id,
		}).Debug("Restarting power monitor because of system audio mode change")

		bridge.MonitorPower(device.Id)
	}, gocec.OpcodeSetSystemAudioMode)

	cec.RegisterMessageHandler(func(message gocec.Message) {
		log.WithFields(log.Fields{
			"message.source":      message.Source(),
			"message.destination": message.Destination(),
			"message.opcode":      message.Opcode(),
			"message.raw":         []byte(message),
		}).Debug("Restarting power monitor on all devices")

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
	state := bridge.states[device.Id]
	defer bridge.statesMutex.Unlock()
	if state.state == value && state.published {
		return
	}

	log.WithFields(log.Fields{
		"device.logical_address": device.LogicalAddress,
		"device.id":              device.Id,
		"state.cec":              status,
		"state.converted":        value,
	}).Info("Updating power state")

	state.state = value

	if value != "unknown" || !state.published {
		go bridge.mqtt.Publish(bridge.mqtt.BuildTopic(device, "power"), 0, false, value)
		state.published = true
	} else {
		state.published = false
	}
}

func (bridge *PowerBridge) createStarter(device *Device) Starter {
	source := gocec.DeviceTV

	if device.LogicalAddress == gocec.DeviceTV {
		source = gocec.DeviceBroadcast
	}

	message := gocec.NewMessage(source, device.LogicalAddress, gocec.OpcodeGiveDevicePowerStatus, []byte{})

	context := log.WithFields(log.Fields{
		"device.logical_address": device.LogicalAddress,
		"device.id":              device.Id,
		"source":                 source,
		"message":                []byte(message),
	})

	return func() {
		context.Trace("Requesting power state from monitor")
		bridge.cec.Transmit(message)
	}
}

func (bridge *PowerBridge) createRunner(device *Device) Runner {
	return func() {
		status := bridge.cec.connection.GetPowerStatus(device.LogicalAddress)

		log.WithFields(log.Fields{
			"device.logical_address": device.LogicalAddress,
			"device.id":              device.Id,
			"status":                 status,
		}).Trace("Updating power from monitor")

		bridge.setPowerStatus(device, status)
	}
}
