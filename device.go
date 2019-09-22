package main

import (
	"github.com/RobertMe/gocec"
	"github.com/google/uuid"
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
	devices = make(map[string]*DeviceConfig)
	data, err := ioutil.ReadFile(dataDirectory + "devices.yaml")
	if err != nil {
		// Log
		return
	}

	err = yaml.Unmarshal(data, &devices)
	if err != nil {
		// Log
		return
	}

	return
}

func (registry *DeviceRegistry) RegisterDeviceAddedHandler(handler DeviceAddedHandler) {
	registry.deviceAddedHandlers = append(registry.deviceAddedHandlers, handler)
}

func (registry *DeviceRegistry) FindByLogicalAddress(address gocec.LogicalAddress) *Device {
	registry.devicesMutex.Lock()
	defer registry.devicesMutex.Unlock()
	device, ok := registry.devices[address]
	if !ok {
		return nil
	}

	return device
}

func (registry *DeviceRegistry) GetByCecDevice(address gocec.LogicalAddress, creator CreateCecDeviceDescription) *Device {
	registry.devicesMutex.Lock()

	if device, ok := registry.devices[address]; ok {
		registry.devicesMutex.Unlock()
		return device
	}

	description := creator()

	device, ok := registry.physicalAddressMap[description.physicalAddress]
	if ok {
		if device.CecDevice.physicalAddress == description.physicalAddress &&
			device.CecDevice.vendor == description.vendor {
			registry.devicesMutex.Unlock()
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

	for _, handler := range registry.deviceAddedHandlers {
		handler(device)
	}

	return device
}

func (registry *DeviceRegistry) FindDevice(physicalAddress string, vendorId int, name string) *DeviceConfig {
	var option *DeviceConfig

	for _, device := range registry.configDevices {
		if device.PhysicalAddress == physicalAddress && device.VendorId == vendorId {
			if device.OSD == name {
				// Exact match so must be it
				return device
			}

			if option != nil {
				option = nil
				break
			}

			option = device
		} else if device.VendorId == vendorId && device.OSD == name {
			if option != nil {
				option = nil
				break
			}

			option = device
		}
	}

	if option != nil {
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

	registry.configDevices[id] = device

	return device
}

func (registry *DeviceRegistry) Save(configPath string) error {
	data, err := yaml.Marshal(registry.configDevices)

	if err != nil {
		return err
	}

	return ioutil.WriteFile(configPath + "devices.yaml", data, 0644)
}
