package messaging

import (
	mvdsnode "github.com/status-im/mvds/node"

	"github.com/status-im/status-go/messaging/layers/encryption"
	"github.com/status-im/status-go/messaging/layers/segmentation"
	"github.com/status-im/status-go/messaging/layers/transport"
	"github.com/status-im/status-go/messaging/types"
	wakuv2 "github.com/status-im/status-go/messaging/waku"
)

type Persistence interface {
	WakuStorage() wakuv2.ProtectedTopicsPersistence
	TransportStorage() TransportPersistence
	SegmentationStorage() segmentation.Persistence
	MVDSStorage() mvdsnode.Persistence
	EncryptionStorage() encryption.Persistence
	MessageSenderStorage() types.MessageSenderPersistence
}

type TransportPersistence interface {
	KeysStorage() transport.KeysPersistence
	ProcessedMessageIDsCacheStorage() transport.ProcessedMessageIDsCachePersistence
}
