package gowaku

import "github.com/waku-org/go-waku/library"

// SignalHandler defines a minimal interface
// a signal handler needs to implement.
// nolint
type SignalHandler interface {
	HandleSignal(string)
}

// SetMobileSignalHandler setup geth callback to notify about new signal
// used for gomobile builds
// nolint
func SetMobileSignalHandler(instanceID uint, handler SignalHandler) {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		panic(err.Error()) // TODO: refactor to return an error instead
	}

	library.SetMobileSignalHandler(instance, func(data []byte) {
		if len(data) > 0 {
			handler.HandleSignal(string(data))
		}
	})
}
