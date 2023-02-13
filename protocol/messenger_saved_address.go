package protocol

import (
	"context"
	"time"

	"github.com/golang/protobuf/proto"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/services/wallet"
)

func (m *Messenger) UpsertSavedAddress(ctx context.Context, sa wallet.SavedAddress) error {
	updatedClock, err := m.savedAddressesManager.UpdateMetadataAndUpsertSavedAddress(sa)
	if err != nil {
		return err
	}
	return m.syncNewSavedAddress(ctx, &sa, updatedClock, m.dispatchMessage)
}

func (m *Messenger) DeleteSavedAddress(ctx context.Context, chainID uint64, address gethcommon.Address) error {
	updatedClock, err := m.savedAddressesManager.DeleteSavedAddress(chainID, address)
	if err != nil {
		return err
	}
	return m.syncDeletedSavedAddress(ctx, chainID, address, updatedClock, m.dispatchMessage)
}

func (m *Messenger) garbageCollectRemovedSavedAddresses() error {
	return m.savedAddressesManager.DeleteSoftRemovedSavedAddresses(uint64(time.Now().AddDate(0, 0, -30).Unix()))
}

func (m *Messenger) dispatchSyncSavedAddress(ctx context.Context, syncMessage protobuf.SyncSavedAddress, rawMessageHandler RawMessageHandler) error {
	if !m.hasPairedDevices() {
		return nil
	}

	clock, chat := m.getLastClockWithRelatedChat()

	encodedMessage, err := proto.Marshal(&syncMessage)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_SAVED_ADDRESS,
		ResendAutomatically: true,
	}

	_, err = rawMessageHandler(ctx, rawMessage)
	if err != nil {
		return err
	}

	chat.LastClockValue = clock
	return m.saveChat(chat)
}

func (m *Messenger) syncNewSavedAddress(ctx context.Context, savedAddress *wallet.SavedAddress, updateClock uint64, rawMessageHandler RawMessageHandler) error {
	return m.dispatchSyncSavedAddress(ctx, protobuf.SyncSavedAddress{
		Address:     savedAddress.Address.Bytes(),
		Name:        savedAddress.Name,
		Favourite:   savedAddress.Favourite,
		ChainId:     savedAddress.ChainID,
		UpdateClock: updateClock,
	}, rawMessageHandler)
}

func (m *Messenger) syncDeletedSavedAddress(ctx context.Context, chainID uint64, address gethcommon.Address, updateClock uint64, rawMessageHandler RawMessageHandler) error {
	return m.dispatchSyncSavedAddress(ctx, protobuf.SyncSavedAddress{
		Address:     address.Bytes(),
		ChainId:     chainID,
		UpdateClock: updateClock,
		Removed:     true,
	}, rawMessageHandler)
}

func (m *Messenger) syncSavedAddress(ctx context.Context, savedAddress wallet.SavedAddress, rawMessageHandler RawMessageHandler) (err error) {
	if savedAddress.Removed {
		if err = m.syncDeletedSavedAddress(ctx, savedAddress.ChainID, savedAddress.Address, savedAddress.UpdateClock, rawMessageHandler); err != nil {
			return err
		}
	} else {
		if err = m.syncNewSavedAddress(ctx, &savedAddress, savedAddress.UpdateClock, rawMessageHandler); err != nil {
			return err
		}
	}
	return
}

func (m *Messenger) handleSyncSavedAddress(state *ReceivedMessageState, syncMessage protobuf.SyncSavedAddress) (err error) {
	address := gethcommon.BytesToAddress(syncMessage.Address)
	if syncMessage.Removed {
		_, err = m.savedAddressesManager.DeleteSavedAddressIfNewerUpdate(syncMessage.ChainId,
			address, syncMessage.UpdateClock)
		if err != nil {
			return err
		}
		state.Response.AddSavedAddress(&wallet.SavedAddress{ChainID: syncMessage.ChainId, Address: address})
	} else {
		sa := wallet.SavedAddress{
			Address:   address,
			Name:      syncMessage.Name,
			Favourite: syncMessage.Favourite,
			ChainID:   syncMessage.ChainId,
		}

		_, err = m.savedAddressesManager.AddSavedAddressIfNewerUpdate(sa, syncMessage.UpdateClock)
		if err != nil {
			return err
		}
		state.Response.AddSavedAddress(&sa)
	}
	return
}
