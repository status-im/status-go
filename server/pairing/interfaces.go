package pairing

import (
	"go.uber.org/zap"
)

// PayloadMounterReceiver represents a struct that can:
//   - mount payload data from a PayloadRepository or a PayloadLoader into memory (PayloadMounter.Mount)
//   - prepare data to be sent encrypted (PayloadMounter.ToSend) via some transport
//   - receive and prepare encrypted transport data (PayloadReceiver.Receive) to be stored
//   - prepare the received (PayloadReceiver.Received) data to be stored to a PayloadRepository or a PayloadStorer
type PayloadMounterReceiver interface {
	PayloadMounter
	PayloadReceiver
}

// PayloadRepository represents a struct that can both load and store data to an internally managed data store
type PayloadRepository interface {
	PayloadLoader
	PayloadStorer
}

type PayloadLocker interface {
	// LockPayload prevents future excess to outbound safe and received data
	LockPayload()
}

type HandlerServer interface {
	GetLogger() *zap.Logger
}

type ProtobufMarshaller interface {
	MarshalProtobuf() ([]byte, error)
}

type ProtobufUnmarshaller interface {
	UnmarshalProtobuf([]byte) error
}
