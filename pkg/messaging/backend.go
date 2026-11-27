package messaging

import "fmt"

// Backend is a messaging transport implementation behind the transport seam
// (types.Waku and transport.MessagingAPI). Every build compiles both; the
// logos_delivery build tag only selects DefaultBackend.
type Backend int

const (
	// BackendDefault is DefaultBackend.
	BackendDefault Backend = iota
	// BackendGoWaku is go-waku, running in-process.
	BackendGoWaku
	// BackendLogosDelivery is logos-delivery, through liblogosdelivery.
	BackendLogosDelivery
)

// Backend names, as WakuV2Config.Backend and the API take them.
const (
	BackendNameGoWaku        = "go-waku"
	BackendNameLogosDelivery = "logos-delivery"
)

// ParseBackend returns the backend named name. An empty name is
// BackendDefault.
func ParseBackend(name string) (Backend, error) {
	switch name {
	case "":
		return BackendDefault, nil
	case BackendNameGoWaku:
		return BackendGoWaku, nil
	case BackendNameLogosDelivery:
		return BackendLogosDelivery, nil
	}
	return BackendDefault, fmt.Errorf("unknown messaging backend %q, want %q or %q", name, BackendNameGoWaku, BackendNameLogosDelivery)
}

// Resolve returns the backend b stands for: DefaultBackend for
// BackendDefault, b otherwise.
func (b Backend) Resolve() Backend {
	if b == BackendDefault {
		return DefaultBackend
	}
	return b
}

func (b Backend) String() string {
	switch b.Resolve() {
	case BackendLogosDelivery:
		return BackendNameLogosDelivery
	default:
		return BackendNameGoWaku
	}
}
