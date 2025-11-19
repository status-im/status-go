package messaging

import (
	mvdsnode "github.com/status-im/mvds/node"

	"github.com/status-im/status-go/messaging/common"
	"github.com/status-im/status-go/messaging/layers/encryption"
	"github.com/status-im/status-go/messaging/layers/segmentation"
	"github.com/status-im/status-go/messaging/layers/transport"
)

type Persistence interface {
	TransportStorage() transport.Persistence
	SegmentationStorage() segmentation.Persistence
	MVDSStorage() mvdsnode.Persistence
	EncryptionStorage() encryption.Persistence

	MessageConfirmationStorage() common.MessageConfirmationPersistence
	HashRatchetStorage() common.HashRatchetPersistence
}
