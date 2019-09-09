package messages

import (
	"fmt"
	"strings"
)

type Address byte

type Message interface {
	MqttPath() string
	Value() string
}

func (address Address) BuildPath(name string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d/%s", address, name)
	return b.String()
}
