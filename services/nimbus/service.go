// +build nimbus

package nimbus

import (
	"errors"
	"fmt"
	"reflect"

	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/params"
)

// errors
var (
	ErrNodeStopped    = errors.New("node not started")
	ErrServiceUnknown = errors.New("service unknown")
)

// DuplicateServiceError is returned during Node startup if a registered service
// constructor returns a service of the same type that was already started.
type DuplicateServiceError struct {
	Kind reflect.Type
}

// Error generates a textual representation of the duplicate service error.
func (e *DuplicateServiceError) Error() string {
	return fmt.Sprintf("duplicate service: %v", e.Kind)
}

// ServiceContext is a collection of service independent options inherited from
// the protocol stack, that is passed to all constructors to be optionally used;
// as well as utility methods to operate on the service environment.
type ServiceContext struct {
	config   *params.NodeConfig
	services map[reflect.Type]Service // Index of the already constructed services
	// EventMux *event.TypeMux           // Event multiplexer used for decoupled notifications
	// AccountManager *accounts.Manager        // Account manager created by the node.
}

func NewServiceContext(config *params.NodeConfig, services map[reflect.Type]Service) *ServiceContext {
	return &ServiceContext{
		config:   config,
		services: services,
	}
}

// Service retrieves a currently running service registered of a specific type.
func (ctx *ServiceContext) Service(service interface{}) error {
	element := reflect.ValueOf(service).Elem()
	if running, ok := ctx.services[element.Type()]; ok {
		element.Set(reflect.ValueOf(running))
		return nil
	}
	return ErrServiceUnknown
}

// ServiceConstructor is the function signature of the constructors needed to be
// registered for service instantiation.
type ServiceConstructor func(ctx *ServiceContext) (Service, error)

// Service is an individual protocol that can be registered into a node.
//
// Notes:
//
// • Service life-cycle management is delegated to the node. The service is allowed to
// initialize itself upon creation, but no goroutines should be spun up outside of the
// Start method.
//
// • Restart logic is not required as the node will create a fresh instance
// every time a service is started.
type Service interface {
	// APIs retrieves the list of RPC descriptors the service provides
	APIs() []gethrpc.API

	// StartService is called after all services have been constructed and the networking
	// layer was also initialized to spawn any goroutines required by the service.
	StartService() error

	// Stop terminates all goroutines belonging to the service, blocking until they
	// are all terminated.
	Stop() error
}
