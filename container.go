package main

import (
	log "github.com/sirupsen/logrus"
	"sync"
)

type Container struct {
	mux      sync.RWMutex
	services map[string]interface{}
}

func NewContainer() *Container {
	return &Container{
		services: make(map[string]interface{}),
	}
}

func (container *Container) Register(name string, service interface{}) {
	log.WithFields(log.Fields{
		"name": name,
	}).Trace("Registering new service into container")

	container.mux.Lock()
	defer container.mux.Unlock()
	container.services[name] = service
}

func (container *Container) Unregister(name string) {
	log.WithFields(log.Fields{
		"name": name,
	}).Trace("Unregistering new service from container")
	container.mux.Lock()
	defer container.mux.Unlock()
	delete(container.services, name)
}

func (container *Container) Get(name string) interface{} {
	log.WithFields(log.Fields{
		"name": name,
	}).Trace("Getting service from container")
	container.mux.RLock()
	defer container.mux.RUnlock()
	service, ok := container.services[name]
	if !ok {
		log.WithFields(log.Fields{
			"name": name,
		}).Debug("Service not available in container")

		return nil
	}

	return service
}
