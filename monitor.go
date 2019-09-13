package main

import "time"

type Starter func()
type Runner func()

type Monitor struct {
	reset chan int

	longInterval time.Duration
	shortInterval time.Duration
	shortTimer *time.Timer
	shortDuration time.Duration

	starter Starter
	runner Runner
}

func CreateMonitor(starter Starter, runner Runner, longInterval time.Duration, shortInterval time.Duration, shortDuration time.Duration) *Monitor {
	monitor := &Monitor{
		reset: make(chan int),
		longInterval: longInterval,
		shortInterval: shortInterval,
		shortDuration: shortDuration,
		starter: starter,
		runner: runner,
	}

	go monitor.run()

	return monitor
}

func (monitor *Monitor) Reset() {
	monitor.reset <- 0
}

func (monitor *Monitor) run() {
	monitor.starter()
	monitor.runner()

	ticker := time.NewTicker(monitor.shortInterval)
	monitor.shortTimer = time.NewTimer(monitor.shortDuration)

	for {
		select {
		case <- ticker.C:
			monitor.runner()
		case <- monitor.reset:
			ticker.Stop()
			monitor.shortTimer.Stop()

			monitor.starter()
			monitor.runner()

			ticker = time.NewTicker(monitor.shortInterval)
			monitor.shortTimer = time.NewTimer(monitor.shortDuration)

		case <- monitor.shortTimer.C:
			ticker.Stop()
			monitor.shortTimer.Stop()

			ticker = time.NewTicker(monitor.longInterval)
		}
	}
}
