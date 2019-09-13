package main

import (
	"github.com/RobertMe/gocec"
)

type Device struct {
	cec *Cec
	LogicalAddress gocec.LogicalAddress

	OSD string
}

func NewDevice(address gocec.LogicalAddress, cec *Cec) *Device {
	device := &Device{
		LogicalAddress: address,
		cec: cec,
	}

	return device
}
