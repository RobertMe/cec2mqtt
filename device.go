package main

import (
	"github.com/RobertMe/gocec"
	"sync"
)

type DeviceAddedHandler func(device *Device)

type DeviceRegistry struct {
	confg *Config

	deviceAddedHandlers []DeviceAddedHandler

	devicesMutex sync.Mutex
	devices map[gocec.LogicalAddress]*Device
	physicalAddressMap map[gocec.PhysicalAddress]*Device
}

type Device struct {
	CecDevice *CecDeviceDescription

	LogicalAddress gocec.LogicalAddress
}

type CreateCecDeviceDescription func() *CecDeviceDescription

func NewDeviceRegistry(config *Config) *DeviceRegistry {
	return &DeviceRegistry{
		confg:               config,
		deviceAddedHandlers: make([]DeviceAddedHandler, 0),
		devices:             make(map[gocec.LogicalAddress]*Device),
		physicalAddressMap:  make(map[gocec.PhysicalAddress]*Device),
	}
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

	device = &Device{
		CecDevice:      description,
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
