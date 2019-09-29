package main

import (
	"errors"
	"github.com/RobertMe/gocec"
	log "github.com/sirupsen/logrus"
	"strings"
)

type MessageReceivedHandler func(message gocec.Message)

type Cec struct {
	connection *gocec.Connection
	adapter    gocec.Adapter

	devices                 *DeviceRegistry
	messageReceivedHandlers map[gocec.Opcode][]MessageReceivedHandler
	LibCecLoggingEnabled    bool
}

type CecDeviceDescription struct {
	logicalAddress  gocec.LogicalAddress
	physicalAddress gocec.PhysicalAddress
	vendor          gocec.Vendor
	OSD             string
}

func InitialiseCec(devices *DeviceRegistry, path string) (*Cec, error) {
	config := gocec.NewConfiguration("cec2mqtt", false)

	config.SetMonitorOnly(false)

	cec := &Cec{
		devices:                 devices,
		messageReceivedHandlers: make(map[gocec.Opcode][]MessageReceivedHandler),
	}
	config.SetLogCallback(cec.handleLogMessage)

	connection, err := gocec.NewConnection(config)
	if err != nil {
		return nil, err
	}

	var adapter gocec.Adapter
	adapters := connection.FindAdapters()

	log.WithFields(log.Fields{
		"adapters": adapters,
	}).Debug("Adapters found")

	if len(path) == 0 {
		adapter = adapters[0]

		log.WithFields(log.Fields{
			"adapter": adapter,
		}).Debug("Using the first available CEC adapter")

	} else {
		var found bool
		for _, adapter = range adapters {
			if adapter.Path == path {
				found = true
			}
		}

		if !found {
			return nil, errors.New("Adapter " + path + " has not been found")
		}

		log.WithFields(log.Fields{
			"adapter": adapter,
		}).Debug("Matched adapter")
	}

	cec.connection = connection
	cec.adapter = adapter

	return cec, nil
}

func (cec *Cec) RegisterMessageHandler(handler MessageReceivedHandler, opcodes ...gocec.Opcode) {
	log.WithFields(log.Fields{
		"opcodes": opcodes,
	}).Trace("Registering message handler")

	for _, opcode := range opcodes {
		handlers, ok := cec.messageReceivedHandlers[opcode]
		if !ok {
			cec.messageReceivedHandlers[opcode] = []MessageReceivedHandler{handler}
		} else {
			cec.messageReceivedHandlers[opcode] = append(handlers, handler)
		}
	}
}

func (cec *Cec) Start() error {
	if err := cec.connection.Open(cec.adapter); err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"adapter": cec.adapter,
	}).Info("Opened CEC connection")

	adapterAddress, _ := cec.connection.GetAdapterAddress()
	addresses := cec.connection.ActiveDevices()

	for _, address := range addresses {
		// Don't register a device for the CEC adapter
		if address != adapterAddress {
			log.WithFields(log.Fields{
				"logical_address": address,
			}).Trace("Found active device")
			_ = cec.GetDevice(address)
		} else {
			log.WithFields(log.Fields{
				"logical_address": address,
			}).Trace("Skipping active device as it's the adapter")
		}
	}

	return nil
}

func (cec *Cec) handleLogMessage(logMessage *gocec.LogMessage) {
	if cec.LibCecLoggingEnabled {
		log.WithFields(log.Fields{
			"message": logMessage.Message,
			"time":    logMessage.Time,
			"level":   logMessage.Level,
		}).Debug("Incoming message from libcec")
	}

	if logMessage.Level != gocec.LogLevelTraffic {
		return
	}

	if !strings.HasPrefix(logMessage.Message, ">> ") {
		return
	}

	message, _ := gocec.ParseMessage(logMessage.Message[3:])

	device := cec.GetDevice(message.Source())

	context := log.WithFields(log.Fields{
		"raw_message":            logMessage.Message[3:],
		"parsed_message":         message,
		"source.logical_address": message.Source(),
		"source.device.id":       device.Id,
		"opcode":                 message.Opcode(),
	})

	context.Debug("Message received")

	if handlers, ok := cec.messageReceivedHandlers[message.Opcode()]; ok {
		for _, handler := range handlers {
			context.Debug("Invoking handler")

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
			OSD:             cec.connection.GetOSDName(address),
		}
	}

	return cec.devices.GetByCecDevice(address, creator)
}

func (cec *Cec) Transmit(message gocec.Message) {
	log.WithFields(log.Fields{
		"message.text": message.String(),
		"message.raw":  []byte(message),
	}).Trace("Transmitting CEC message")
	cec.connection.Transmit(message)
}
