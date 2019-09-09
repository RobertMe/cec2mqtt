package main

import (
	"github.com/RobertMe/cec2mqtt/messages"
	"github.com/RobertMe/gocec"
)

func convertPowerStatus(status gocec.PowerStatus) messages.PowerState {
	var state messages.PowerState
	switch status {
	case gocec.PowerStatusOn, gocec.PowerStatusTransitionToStandby:
		state = messages.PowerStateOn
	case gocec.PowerStatusStandBy, gocec.PowerStatusTransitionToOn:
		state = messages.PowerStateStandBy
	default:
		state = messages.PowerStateUnknown
	}
	return state
}
