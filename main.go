package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
)

const (
	BuildVersion = "0.0.1"
)

type Initializer func(container *Container)

var initializers = make(map[int][]Initializer, 0)

func RegisterInitializer(priority int, initializer Initializer) {
	if _, ok := initializers[priority]; ok {
		initializers[priority] = append(initializers[priority], initializer)
	} else {
		initializers[priority] = []Initializer{initializer}
	}
}

func runInitializers(container *Container) {
	priorities := make([]int, len(initializers))
	i := 0
	for priority := range initializers {
		priorities[i] = priority
		i++
	}

	sort.Ints(priorities)
	for i := len(priorities) - 1; i >= 0; i-- {
		for _, initializer := range initializers[priorities[i]] {
			initializer(container)
		}
	}
}

func main() {

	var dataDir string
	flag.StringVar(&dataDir, "data-dir", "/data/cec2mqtt/", "Sets the directory where the data, including config, files are stored")

	var logLevel string
	flag.StringVar(&logLevel, "log-level", "info", "Sets the log level. Options are panic, fatal, error, warning, info, debug, trace")

	var logCecMessages bool
	flag.BoolVar(&logCecMessages, "log-cec-messages", false, "Enables logging of the libcec log")

	flag.Parse()

	switch logLevel {
	case "panic":
		log.SetLevel(log.PanicLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "warning":
		log.SetLevel(log.WarnLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "trace":
		log.SetLevel(log.TraceLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	log.WithField("version", BuildVersion).Info("Starting Cec2Mqtt")

	dataDir = strings.TrimRight(dataDir, "/") + "/"

	container := NewContainer()

	config, err := ParseConfig(dataDir)

	if nil != err {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Error reading configuration")
	}

	container.Register("config", config)

	devices := NewDeviceRegistry(dataDir)
	container.Register("devices", devices)

	mqtt, err := ConnectMqtt(config)

	if nil != err {
		log.WithFields(log.Fields{
			"config": config.Mqtt,
			"error": err,
		}).Fatal("Failed to connect to MQTT broker")
	}

	container.Register("mqtt", mqtt)

	cec, err := InitialiseCec(devices, "")
	cec.LibCecLoggingEnabled = logCecMessages

	if nil != err {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Failed to setup CEC connection")
	}

	container.Register("cec", cec)

	runInitializers(container)

	cec.Start()

	signals := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<- signals
		done <- true
	}()

	log.Info("Cec2Mqtt started")
	<- done
	log.Info("Exiting")
	config.Save(dataDir)
	devices.Save(dataDir)
}
