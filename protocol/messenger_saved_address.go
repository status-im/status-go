package protocol

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/server"
	"github.com/status-im/status-go/services/wallet"
)

func (m *Messenger) UpsertSavedAddress(ctx context.Context, sa wallet.SavedAddress) error {
	updatedClock, err := m.savedAddressesManager.UpdateMetadataAndUpsertSavedAddress(sa)
	if err != nil {
		return err
	}
	return m.syncNewSavedAddress(ctx, &sa, updatedClock)
}

func (m *Messenger) DeleteSavedAddress(ctx context.Context, chainID uint64, address gethcommon.Address) error {
	updatedClock, err := m.savedAddressesManager.DeleteSavedAddress(chainID, address)
	if err != nil {
		return err
	}
	return m.syncDeletedSavedAddress(ctx, chainID, address, updatedClock)
}

func (m *Messenger) garbageCollectRemovedSavedAddresses() error {
	return m.savedAddressesManager.DeleteSoftRemovedSavedAddresses(uint64(time.Now().AddDate(0, 0, -30).Unix()))
}

func (m *Messenger) setInstallationHostname() error {
	randomDeviceIDLen := 5

	ourInstallation, ok := m.allInstallations.Load(m.installationID)
	if !ok {
		m.logger.Error("Messenger's installationID is not set or not loadable")
		return nil
	}

	var imd *multidevice.InstallationMetadata
	if ourInstallation.InstallationMetadata == nil {
		imd = new(multidevice.InstallationMetadata)
	} else {
		imd = ourInstallation.InstallationMetadata
	}

	// If the name is already set, don't do anything
	// TODO check the full working mechanics of this
	if len(imd.Name) > randomDeviceIDLen {
		return nil
	}

	if len(imd.Name) == 0 {
		n, err := common.RandomAlphabeticalString(randomDeviceIDLen)
		if err != nil {
			return err
		}

		imd.Name = n
	}

	hn, err := server.GetDeviceName()
	if err != nil {
		return err
	}
	imd.Name = fmt.Sprintf("%s %s", hn, imd.Name)
	return m.setInstallationMetadata(m.installationID, imd)
}

func (m *Messenger) dispatchSyncSavedAddress(ctx context.Context, syncMessage protobuf.SyncSavedAddress) error {
	if !m.hasPairedDevices() {
		return nil
	}

	clock, chat := m.getLastClockWithRelatedChat()

	encodedMessage, err := proto.Marshal(&syncMessage)
	if err != nil {
		return err
	}

	_, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_SAVED_ADDRESS,
		ResendAutomatically: true,
	})
	if err != nil {
		return err
	}

	chat.LastClockValue = clock
	return m.saveChat(chat)
}

func (m *Messenger) syncNewSavedAddress(ctx context.Context, savedAddress *wallet.SavedAddress, updateClock uint64) error {
	return m.dispatchSyncSavedAddress(ctx, protobuf.SyncSavedAddress{
		Address:     savedAddress.Address.Bytes(),
		Name:        savedAddress.Name,
		Favourite:   savedAddress.Favourite,
		ChainId:     savedAddress.ChainID,
		UpdateClock: updateClock,
	})
}

func (m *Messenger) syncDeletedSavedAddress(ctx context.Context, chainID uint64, address gethcommon.Address, updateClock uint64) error {
	return m.dispatchSyncSavedAddress(ctx, protobuf.SyncSavedAddress{
		Address:     address.Bytes(),
		ChainId:     chainID,
		UpdateClock: updateClock,
		Removed:     true,
	})
}

func (m *Messenger) syncSavedAddress(ctx context.Context, savedAddress wallet.SavedAddress) (err error) {
	if savedAddress.Removed {
		if err = m.syncDeletedSavedAddress(ctx, savedAddress.ChainID, savedAddress.Address, savedAddress.UpdateClock); err != nil {
			return err
		}
	} else {
		if err = m.syncNewSavedAddress(ctx, &savedAddress, savedAddress.UpdateClock); err != nil {
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
		state.Response.SavedAddresses = append(state.Response.SavedAddresses, &wallet.SavedAddress{ChainID: syncMessage.ChainId, Address: address})
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
		state.Response.SavedAddresses = append(state.Response.SavedAddresses, &sa)
	}
	return
}
