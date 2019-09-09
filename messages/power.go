package messages

type PowerState byte

const (
	PowerStateOn PowerState = 1
	PowerStateStandBy PowerState = 2
	PowerStateUnknown PowerState = 99
)

type PowerMessage struct {
	Address Address
	State PowerState
}

func (message *PowerMessage) MqttPath() string {
	return message.Address.BuildPath("power")
}

func (message *PowerMessage) Value() string {
	switch message.State {
	case PowerStateOn:
		return "on"
	case PowerStateStandBy:
		return "off"
	default:
		return "unknown"
	}
}
