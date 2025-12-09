package messaging

import (
	"crypto/ecdsa"

	ethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/messaging/adapters"
	"github.com/status-im/status-go/messaging/types"
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

func (a *API) MarkP2PMessageAsProcessed(hash ethcommon.Hash) {
	a.core.stack.Transport.MarkP2PMessageAsProcessed(hash)
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
