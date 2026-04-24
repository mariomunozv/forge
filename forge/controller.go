package forge

import "fmt"

// Controller is the interface all Forge controllers must implement.
// Individual actions are methods on the controller struct.
type Controller interface{}

// MiddlewareFunc is a function that wraps a HandlerFunc.
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// HandlerFunc is the core handler signature in Forge.
type HandlerFunc func(*Context) error

// registry maps controller names to their instances.
type registry struct {
	controllers map[string]Controller
}

func newRegistry() *registry {
	return &registry{controllers: make(map[string]Controller)}
}

func (r *registry) set(name string, c Controller) {
	r.controllers[name] = c
}

func (r *registry) get(name string) (Controller, error) {
	c, ok := r.controllers[name]
	if !ok {
		return nil, fmt.Errorf("forge: controller '%s' not registered", name)
	}
	return c, nil
}
