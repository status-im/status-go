package protocol

import (
	"context"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/keypairs"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

func (m *Messenger) dispatchSyncKeycard(ctx context.Context, chatID string, syncKeycard protobuf.SyncAllKeycards,
	rawMessageHandler RawMessageHandler) error {
	if !m.hasPairedDevices() {
		return nil
	}

	encodedMessage, err := proto.Marshal(&syncKeycard)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID:         chatID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_ALL_KEYCARDS,
		ResendAutomatically: true,
	}

	_, err = rawMessageHandler(ctx, rawMessage)
	return err
}

func (m *Messenger) prepareSyncAllKeycardsMessage(clock uint64) (message protobuf.SyncAllKeycards, err error) {
	allKeycards, err := m.settings.GetAllKnownKeycards()
	if err != nil {
		return message, err
	}

	message.Clock = clock

	for _, kc := range allKeycards {
		syncKeycard := kc.ToSyncKeycard()
		if syncKeycard.Clock == 0 {
			syncKeycard.Clock = clock
		}
		message.Keycards = append(message.Keycards, syncKeycard)
	}

	return
}

func (m *Messenger) syncAllKeycards(ctx context.Context, rawMessageHandler RawMessageHandler) (err error) {
	clock, chat := m.getLastClockWithRelatedChat()

	message, err := m.prepareSyncAllKeycardsMessage(clock)
	if err != nil {
		return err
	}

	err = m.dispatchSyncKeycard(ctx, chat.ID, message, rawMessageHandler)
	if err != nil {
		return err
	}

	chat.LastClockValue = clock
	return m.saveChat(chat)
}

func (m *Messenger) syncReceivedKeycards(syncMessage protobuf.SyncAllKeycards) ([]*keypairs.KeyPair, error) {
	var keypairsToSync []*keypairs.KeyPair
	for _, syncKc := range syncMessage.Keycards {
		var kp = &keypairs.KeyPair{}
		kp.FromSyncKeycard(syncKc)
		keypairsToSync = append(keypairsToSync, kp)
	}

	err := m.settings.SyncKeycards(syncMessage.Clock, keypairsToSync)
	if err != nil {
		return nil, err
	}

	allKeycards, err := m.settings.GetAllKnownKeycards()
	if err != nil {
		return nil, err
	}

	return allKeycards, nil
}

func (m *Messenger) handleSyncKeycards(state *ReceivedMessageState, syncMessage protobuf.SyncAllKeycards) (err error) {
	allKeycards, err := m.syncReceivedKeycards(syncMessage)
	if err != nil {
		return err
	}

	state.Response.AddAllKnownKeycards(allKeycards)

	return nil
}

func (m *Messenger) dispatchKeycardActivity(ctx context.Context, syncMessage protobuf.SyncKeycardAction) error {
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
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_KEYCARD_ACTION,
		ResendAutomatically: true,
	}

	_, err = m.dispatchMessage(ctx, rawMessage)
	if err != nil {
		return err
	}

	chat.LastClockValue = clock
	return m.saveChat(chat)
}

func (m *Messenger) handleSyncKeycardActivity(state *ReceivedMessageState, syncMessage protobuf.SyncKeycardAction) (err error) {

	var kcAction = &keypairs.KeycardAction{
		Action:        protobuf.SyncKeycardAction_Action_name[int32(syncMessage.Action)],
		OldKeycardUID: syncMessage.OldKeycardUid,
		Keycard:       &keypairs.KeyPair{},
	}
	kcAction.Keycard.FromSyncKeycard(syncMessage.Keycard)

	switch syncMessage.Action {
	case protobuf.SyncKeycardAction_KEYCARD_ADDED,
		protobuf.SyncKeycardAction_ACCOUNTS_ADDED:
		_, _, err = m.settings.AddMigratedKeyPairOrAddAccountsIfKeyPairIsAdded(*kcAction.Keycard)
	case protobuf.SyncKeycardAction_KEYCARD_DELETED:
		err = m.settings.DeleteKeycard(kcAction.Keycard.KeycardUID, kcAction.Keycard.LastUpdateClock)
	case protobuf.SyncKeycardAction_ACCOUNTS_REMOVED:
		err = m.settings.RemoveMigratedAccountsForKeycard(kcAction.Keycard.KeycardUID, kcAction.Keycard.AccountsAddresses,
			kcAction.Keycard.LastUpdateClock)
	case protobuf.SyncKeycardAction_LOCKED:
		err = m.settings.KeycardLocked(kcAction.Keycard.KeycardUID, kcAction.Keycard.LastUpdateClock)
	case protobuf.SyncKeycardAction_UNLOCKED:
		err = m.settings.KeycardUnlocked(kcAction.Keycard.KeycardUID, kcAction.Keycard.LastUpdateClock)
	case protobuf.SyncKeycardAction_UID_UPDATED:
		err = m.settings.UpdateKeycardUID(kcAction.OldKeycardUID, kcAction.Keycard.KeycardUID,
			kcAction.Keycard.LastUpdateClock)
	case protobuf.SyncKeycardAction_NAME_CHANGED:
		err = m.settings.SetKeycardName(kcAction.Keycard.KeycardUID, kcAction.Keycard.KeycardName,
			kcAction.Keycard.LastUpdateClock)
	default:
		panic("unknown action for handling keycard activity")
	}

	if err != nil {
		return err
	}

	state.Response.AddKeycardAction(kcAction)

	return nil
}

func (m *Messenger) AddMigratedKeyPairOrAddAccountsIfKeyPairIsAdded(ctx context.Context, kp *keypairs.KeyPair) (added bool, err error) {
	addedKc, addedAccs, err := m.settings.AddMigratedKeyPairOrAddAccountsIfKeyPairIsAdded(*kp)
	if err != nil {
		return addedKc || addedAccs, err
	}

	activityMessage := protobuf.SyncKeycardAction{
		Keycard: kp.ToSyncKeycard(),
	}
	if addedKc {
		activityMessage.Action = protobuf.SyncKeycardAction_KEYCARD_ADDED
	} else if addedAccs {
		activityMessage.Action = protobuf.SyncKeycardAction_ACCOUNTS_ADDED
	}

	return addedKc || addedAccs, m.dispatchKeycardActivity(ctx, activityMessage)
}

func (m *Messenger) RemoveMigratedAccountsForKeycard(ctx context.Context, kcUID string, accountAddresses []string, clock uint64) error {
	var addresses []types.Address
	for _, addr := range accountAddresses {
		addresses = append(addresses, types.HexToAddress(addr))
	}

	err := m.settings.RemoveMigratedAccountsForKeycard(kcUID, addresses, clock)
	if err != nil {
		return err
	}

	activityMessage := protobuf.SyncKeycardAction{
		Action: protobuf.SyncKeycardAction_ACCOUNTS_REMOVED,
		Keycard: &protobuf.SyncKeycard{
			Uid:   kcUID,
			Clock: clock,
		},
	}

	for _, addr := range addresses {
		activityMessage.Keycard.Addresses = append(activityMessage.Keycard.Addresses, addr.Bytes())
	}

	return m.dispatchKeycardActivity(ctx, activityMessage)
}

func (m *Messenger) SetKeycardName(ctx context.Context, kcUID string, kpName string, clock uint64) error {
	err := m.settings.SetKeycardName(kcUID, kpName, clock)
	if err != nil {
		return err
	}

	activityMessage := protobuf.SyncKeycardAction{
		Action: protobuf.SyncKeycardAction_NAME_CHANGED,
		Keycard: &protobuf.SyncKeycard{
			Uid:   kcUID,
			Name:  kpName,
			Clock: clock,
		},
	}

	return m.dispatchKeycardActivity(ctx, activityMessage)
}

func (m *Messenger) KeycardLocked(ctx context.Context, kcUID string, clock uint64) error {
	err := m.settings.KeycardLocked(kcUID, clock)
	if err != nil {
		return err
	}

	activityMessage := protobuf.SyncKeycardAction{
		Action: protobuf.SyncKeycardAction_LOCKED,
		Keycard: &protobuf.SyncKeycard{
			Uid:   kcUID,
			Clock: clock,
		},
	}

	return m.dispatchKeycardActivity(ctx, activityMessage)
}

func (m *Messenger) KeycardUnlocked(ctx context.Context, kcUID string, clock uint64) error {
	err := m.settings.KeycardUnlocked(kcUID, clock)
	if err != nil {
		return err
	}

	activityMessage := protobuf.SyncKeycardAction{
		Action: protobuf.SyncKeycardAction_UNLOCKED,
		Keycard: &protobuf.SyncKeycard{
			Uid:   kcUID,
			Clock: clock,
		},
	}

	return m.dispatchKeycardActivity(ctx, activityMessage)
}

func (m *Messenger) DeleteKeycard(ctx context.Context, kcUID string, clock uint64) error {
	err := m.settings.DeleteKeycard(kcUID, clock)
	if err != nil {
		return err
	}

	activityMessage := protobuf.SyncKeycardAction{
		Action: protobuf.SyncKeycardAction_KEYCARD_DELETED,
		Keycard: &protobuf.SyncKeycard{
			Uid:   kcUID,
			Clock: clock,
		},
	}

	return m.dispatchKeycardActivity(ctx, activityMessage)
}

func (m *Messenger) UpdateKeycardUID(ctx context.Context, oldKcUID string, newKcUID string, clock uint64) error {
	err := m.settings.UpdateKeycardUID(oldKcUID, newKcUID, clock)
	if err != nil {
		return err
	}

	activityMessage := protobuf.SyncKeycardAction{
		Action:        protobuf.SyncKeycardAction_UID_UPDATED,
		OldKeycardUid: oldKcUID,
		Keycard: &protobuf.SyncKeycard{
			Uid:   newKcUID,
			Clock: clock,
		},
	}

	return m.dispatchKeycardActivity(ctx, activityMessage)
}
