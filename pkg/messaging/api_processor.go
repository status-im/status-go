package messaging

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/internal/connection"
	adapters "github.com/status-im/status-go/pkg/messaging/adapters"
	types "github.com/status-im/status-go/pkg/messaging/types"
)

func (a *API) RetrieveRawAll() (map[types.ChatFilter][]*types.ReceivedMessage, error) {
	filters, err := a.core.stack.Transport.RetrieveRawAll()
	if err != nil {
		return nil, err
	}
	chatFilters := make(map[types.ChatFilter][]*types.ReceivedMessage)
	for k, v := range filters {
		chatFilters[*adapters.FromTransportFilter(&k)] = adapters.FromWakuMessages(v)
	}
	return chatFilters, nil
}

func (a *API) HandleReceivedMessages(msg *types.ReceivedMessage) (*types.HandleMessageResponse, error) {
	return a.core.controller.Processor().ProcessMessage(msg)
}

func (a *API) ConfirmMessagesProcessed(ids []string, timestamp uint64) error {
	return a.core.stack.Transport.ConfirmMessagesProcessed(ids, timestamp)
}

func (a *API) CleanMessagesProcessed(timestamp uint64) error {
	return a.core.stack.Transport.CleanMessagesProcessed(timestamp)
}

func (a *API) ClearProcessedMessageIDsCache() error {
	return a.core.stack.Transport.ClearProcessedMessageIDsCache()
}

func (a *API) SetEnvelopeEventsHandler(handler types.EnvelopeEventsHandler) error {
	return a.core.stack.Transport.SetEnvelopeEventsHandler(handler)
}

func (a *API) ReportUserOnline(publicKey *ecdsa.PublicKey, eventTime uint64) {
	a.core.stack.Reliability.ReportPeerOnline(publicKey, eventTime)
}

func (a *API) EncryptionSubscriptions() *types.EncryptionSubscriptions {
	return adapters.FromEncryptionSubscriptions(a.core.stack.Encryption.Subscriptions())
}

func (a *API) ConnectionChanged(state connection.State) {
	a.core.connectionChanged(state)
}
