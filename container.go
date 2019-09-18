package main

import "sync"

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
	container.mux.Lock()
	defer container.mux.Unlock()
	container.services[name] = service
}

func (container *Container) Unregister(name string) {
	container.mux.Lock()
	defer container.mux.Unlock()
	delete(container.services, name)
}

func (container *Container) Get(name string) interface{} {
	container.mux.RLock()
	defer container.mux.RUnlock()
	service, ok := container.services[name]
	if !ok {
		return nil
	}

	return service
}
