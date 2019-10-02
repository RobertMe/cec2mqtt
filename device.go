package main

import (
	"github.com/RobertMe/gocec"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"sync"
)

type DeviceAddedHandler func(device *Device)

type DeviceRegistry struct {
	configDevices map[string]*DeviceConfig

	deviceAddedHandlers []DeviceAddedHandler

	devicesMutex       sync.Mutex
	devices            map[gocec.LogicalAddress]*Device
	physicalAddressMap map[gocec.PhysicalAddress]*Device
}

type DeviceConfig struct {
	Id              string `yaml:"id"`
	PhysicalAddress string `yaml:"physical_address"`
	VendorId        int    `yaml:"vendor_id"`
	OSD             string `yaml:"osd"`
	MqttTopic       string `yaml:"mqtt_topic"`
	Ignore			bool `yaml:"ignore"`
}

type Device struct {
	Id string

	CecDevice *CecDeviceDescription
	Config    *DeviceConfig

	LogicalAddress gocec.LogicalAddress
}

type CreateCecDeviceDescription func() *CecDeviceDescription

func NewDeviceRegistry(dataDirectory string) *DeviceRegistry {
	return &DeviceRegistry{
		configDevices:       loadDevicesFromConfig(dataDirectory),
		deviceAddedHandlers: make([]DeviceAddedHandler, 0),
		devices:             make(map[gocec.LogicalAddress]*Device),
		physicalAddressMap:  make(map[gocec.PhysicalAddress]*Device),
	}
}

func loadDevicesFromConfig(dataDirectory string) (devices map[string]*DeviceConfig) {
	logContext := log.WithFields(log.Fields{
		"config_file": dataDirectory + "devices.yaml",
	})
	logContext.Info("Reading devices from config file")

	devices = make(map[string]*DeviceConfig)
	data, err := ioutil.ReadFile(dataDirectory + "devices.yaml")
	if err != nil {
		logContext.Info("Devices configuration file does not exist yet")
		return
	}

	err = yaml.Unmarshal(data, &devices)
	if err != nil {
		logContext.WithFields(log.Fields{
			"error": err,
		}).Error("Devices configuration file could not be parsed as valid YAML")
		return
	}

	return
}

func (registry *DeviceRegistry) RegisterDeviceAddedHandler(handler DeviceAddedHandler) {
	log.Trace("Registering device added handler")
	registry.deviceAddedHandlers = append(registry.deviceAddedHandlers, handler)
}

func (registry *DeviceRegistry) FindByLogicalAddress(address gocec.LogicalAddress) *Device {
	logContext := log.WithFields(log.Fields{
		"logical_address": address,
	})
	registry.devicesMutex.Lock()
	defer registry.devicesMutex.Unlock()
	device, ok := registry.devices[address]
	if !ok {
		logContext.Info("Could not find device by logical address")
		return nil
	}

	if device.Config.Ignore {
		logContext.Debug("Found device by logical address, but it's ignored")

		return nil
	}

	logContext.WithFields(log.Fields{
		"device.id": device.Id,
	}).Debug("Found device by logical address")

	return device
}

func (registry *DeviceRegistry) GetByCecDevice(address gocec.LogicalAddress, creator CreateCecDeviceDescription) *Device {
	registry.devicesMutex.Lock()

	logContext := log.WithFields(log.Fields{
		"logical_address": address,
	})

	if device, ok := registry.devices[address]; ok {
		registry.devicesMutex.Unlock()

		logContext = logContext.WithFields(log.Fields{
			"device.id":       device.Config.Id,
		})

		if device.Config.Ignore {
			logContext.Trace("Found device by logical address, but it's ignored")

			return nil
		}

		logContext.Trace("Found device by logical address")

		return device
	}

	description := creator()

	if description.physicalAddress == [2]byte{0xFF, 0xFF} ||
		description.vendor == gocec.VendorUnknown ||
		description.OSD == "cec2mqtt" {
		registry.devicesMutex.Unlock()
		logContext.WithFields(log.Fields{
			"description.physical_address": description.physicalAddress,
			"description.vendor": uint(description.vendor),
			"description.osd": description.OSD,
		}).Debug("Ignoring device because it is incomplete or cec2mqtt")

		return nil
	}

	device, ok := registry.physicalAddressMap[description.physicalAddress]
	if ok {
		if device.CecDevice.physicalAddress == description.physicalAddress &&
			device.CecDevice.vendor == description.vendor {
			registry.devices[address] = device
			registry.devicesMutex.Unlock()

			logContext = logContext.WithFields(log.Fields{
				"physical_address": description.physicalAddress,
				"vendor_id":        description.vendor,
				"description":      description,
				"device.id":        device.Config.Id,
			})

			if device.Config.Ignore {
				logContext.Debug("Found device by physical address and vendor, but it's ignored")

				return nil
			}

			logContext.Debug("Found device by physical address and vendor")

			return device
		}
	}

	deviceConfig := registry.FindDevice(description.physicalAddress.String(), int(description.vendor), description.OSD)

	device = &Device{
		Id:             deviceConfig.Id,
		CecDevice:      description,
		Config:         deviceConfig,
		LogicalAddress: description.logicalAddress,
	}

	registry.devices[device.LogicalAddress] = device
	registry.physicalAddressMap[description.physicalAddress] = device

	registry.devicesMutex.Unlock()

	logContext = logContext.WithFields(log.Fields{
		"physical_address": description.physicalAddress,
		"vendor":           description.vendor,
		"description":      description,
		"device.id":        device.Config.Id,
	})

	if deviceConfig.Ignore {
		logContext.Debug("Added new device, but not registering it as it's ignored")

		return nil
	}

	logContext.Debug("Adding new device")

	for _, handler := range registry.deviceAddedHandlers {
		handler(device)
	}

	return device
}

func (registry *DeviceRegistry) FindDevice(physicalAddress string, vendorId int, name string) *DeviceConfig {
	logContext := log.WithFields(log.Fields{
		"physical_address": physicalAddress,
		"vendor_id":        vendorId,
		"name":             name,
	})
	logContext.Trace("Searching device in config")
	var option *DeviceConfig

	for _, device := range registry.configDevices {
		deviceLogContext := logContext.WithFields(log.Fields{
			"device.id":               device.Id,
			"device.physical_address": device.PhysicalAddress,
			"device.vendor_id":        device.VendorId,
			"device.osd":              device.OSD,
		})

		if device.PhysicalAddress == physicalAddress && device.VendorId == vendorId {
			if device.OSD == name {
				// Exact match so must be it

				deviceLogContext.Info("Matched exact device from config")

				return device
			}

			if option != nil {
				deviceLogContext.Debug("Multiple possible matches are found. Skipping.")
				option = nil
				break
			}

			deviceLogContext.Debug("Found possible match")
			option = device
		} else if device.VendorId == vendorId && device.OSD == name {
			if option != nil {
				deviceLogContext.Debug("Multiple possible matches are found. Skipping.")
				option = nil
				break
			}

			deviceLogContext.Debug("Found possible match")
			option = device
		}
	}

	if option != nil {
		logContext.WithFields(log.Fields{
			"device.id":               option.Id,
			"device.physical_address": option.PhysicalAddress,
			"device.vendor_id":        option.VendorId,
			"device.osd":              option.OSD,
		}).Info("Found device in config")

		option.PhysicalAddress = physicalAddress
		option.VendorId = vendorId
		option.OSD = name
		return option
	}

	var id string
	for {
		id = uuid.New().String()

		if _, ok := registry.configDevices[id]; !ok {
			break
		}
	}

	device := &DeviceConfig{
		Id:              id,
		PhysicalAddress: physicalAddress,
		VendorId:        vendorId,
		OSD:             name,
		MqttTopic:       name,
	}

	log.WithFields(log.Fields{
		"device.id":               device.Id,
		"device.physical_address": device.PhysicalAddress,
		"device.vendor_id":        device.VendorId,
		"device.osd":              device.OSD,
		"device.mqtt_topic":       device.MqttTopic,
	}).Info("Created new device")

	registry.configDevices[id] = device

	return device
}

func (registry *DeviceRegistry) Save(configPath string) error {
	data, err := yaml.Marshal(registry.configDevices)

	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to convert devices configuration into YAML")
		return err
	}

	err = ioutil.WriteFile(configPath+"devices.yaml", data, 0644)
	if err != nil {
		log.WithFields(log.Fields{
			"error":       err,
			"config_file": configPath + "devices.yaml",
		}).Error("Failed to save devices configuration to file")

		return err
	}

	return nil
}
