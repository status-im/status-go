package common

import (
	"github.com/status-im/status-go/pkg/messaging/layers/encryption"
	"github.com/status-im/status-go/pkg/messaging/layers/reliability"
	"github.com/status-im/status-go/pkg/messaging/layers/segmentation"
	"github.com/status-im/status-go/pkg/messaging/layers/transport"
)

type MessagingStack struct {
	Transport    *transport.Transport
	Segmentation *segmentation.Segmenter
	Encryption   *encryption.Protocol
	Reliability  *reliability.Reliability
}
