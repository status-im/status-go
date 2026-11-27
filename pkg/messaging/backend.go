package messaging

// Backend is a messaging transport implementation behind the transport seam
// (types.Waku and transport.MessagingAPI). Every build compiles both; the
// logos_delivery build tag only selects DefaultBackend.
type Backend int

const (
	// BackendGoWaku is go-waku, running in-process.
	BackendGoWaku Backend = iota
	// BackendLogosDelivery is logos-delivery, through liblogosdelivery.
	BackendLogosDelivery
)
