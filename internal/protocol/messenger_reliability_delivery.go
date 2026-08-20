package protocol

import (
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/pkg/messaging/events"
	"github.com/status-im/status-go/pkg/pubsub"
)

// watchReliabilityDeliveryEvents consumes delivery confirmations produced by
// reliability layers that are not carried in MVDS acknowledgements (currently
// SDS for community messages).
func (m *Messenger) watchReliabilityDeliveryEvents() {
	deliveredSub, unsubscribe := pubsub.Subscribe[events.DeliveredMessage](m.messaging.Publisher(), 100)

	m.shutdownWaitGroup.Add(1)
	go func() {
		defer panics.LogOnPanic()
		defer m.shutdownWaitGroup.Done()
		defer unsubscribe()

		for {
			select {
			case <-m.quit:
				return
			case delivered, ok := <-deliveredSub:
				if !ok {
					return
				}

				messageIDs := make([]string, 0, len(delivered.MessageIDs))
				for _, messageID := range delivered.MessageIDs {
					messageIDs = append(messageIDs, cryptotypes.EncodeHex(messageID))
				}
				m.markDeliveredMessageIDs(messageIDs)
			}
		}
	}()
}
