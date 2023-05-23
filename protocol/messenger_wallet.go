package protocol

import (
	"context"
	"errors"
	"time"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/protobuf"
)

var (
	checkBalancesInterval = time.Minute * 10

	ErrCannotChangeKeypairName = errors.New("cannot change profile keypair name")
)

func (m *Messenger) retrieveWalletBalances() error {
	if m.walletAPI == nil {
		m.logger.Warn("wallet api not enabled")
	}
	accounts, err := m.settings.GetAccounts()
	if err != nil {
		return err
	}

	if len(accounts) == 0 {
		m.logger.Info("no accounts to sync wallet balance")
	}

	var ethAccounts []ethcommon.Address

	for _, acc := range accounts {
		m.logger.Info("syncing wallet address", zap.String("account", acc.Address.Hex()))
		ethAccounts = append(ethAccounts, ethcommon.BytesToAddress(acc.Address.Bytes()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	// TODO: publish tokens as a signal
	_, err = m.walletAPI.GetWalletToken(ctx, ethAccounts)
	if err != nil {
		return err
	}

	return nil
}

func (m *Messenger) watchWalletBalances() {
	m.logger.Info("watching wallet balances")

	if m.walletAPI == nil {
		m.logger.Warn("wallet service not enabled")
		return
	}
	go func() {
		for {
			select {
			case <-time.After(checkBalancesInterval):

				err := m.retrieveWalletBalances()
				if err != nil {
					m.logger.Error("failed to retrieve wallet balances", zap.Error(err))
				}
			case <-m.quit:
				return
			}
		}
	}()
}

func (m *Messenger) UpdateKeypairName(keyUID string, name string) error {
	if keyUID == m.account.KeyUID && name != m.account.Name {
		// profile keypair name must always follow profile display name
		return ErrCannotChangeKeypairName
	}
	clock, _ := m.getLastClockWithRelatedChat()
	err := m.settings.UpdateKeypairName(keyUID, name, clock)
	if err != nil {
		return err
	}

	dbKeypair, err := m.settings.GetKeypairByKeyUID(m.account.KeyUID)
	if err != nil {
		return err
	}

	return m.syncKeypair(dbKeypair, false, m.dispatchMessage)
}

func (m *Messenger) SaveOrUpdateKeypair(keypair *accounts.Keypair) error {
	if keypair.KeyUID == m.account.KeyUID && keypair.Name != m.account.Name {
		// profile keypair name must always follow profile display name
		return ErrCannotChangeKeypairName
	}
	clock, _ := m.getLastClockWithRelatedChat()
	keypair.Clock = clock

	for _, acc := range keypair.Accounts {
		acc.Clock = clock
	}

	err := m.settings.SaveOrUpdateKeypair(keypair)
	if err != nil {
		return err
	}
	return m.syncKeypair(keypair, false, m.dispatchMessage)
}

func (m *Messenger) SaveOrUpdateAccount(acc *accounts.Account) error {
	clock, _ := m.getLastClockWithRelatedChat()
	acc.Clock = clock

	err := m.settings.SaveOrUpdateAccounts([]*accounts.Account{acc})
	if err != nil {
		return err
	}
	return m.syncWalletAccount(acc, m.dispatchMessage)
}

func (m *Messenger) DeleteAccount(address types.Address) error {

	acc, err := m.settings.GetAccountByAddress(address)
	if err != nil {
		return err
	}

	err = m.settings.DeleteAccount(address)
	if err != nil {
		return err
	}

	clock, chat := m.getLastClockWithRelatedChat()
	acc.Clock = clock
	acc.Removed = true

	err = m.syncWalletAccount(acc, m.dispatchMessage)
	if err != nil {
		return err
	}

	chat.LastClockValue = clock
	return m.saveChat(chat)
}

func (m *Messenger) prepareSyncAccountMessage(acc *accounts.Account) *protobuf.SyncAccount {
	if acc.Chat {
		return nil
	}

	return &protobuf.SyncAccount{
		Clock:     acc.Clock,
		Address:   acc.Address.Bytes(),
		KeyUid:    acc.KeyUID,
		PublicKey: acc.PublicKey,
		Path:      acc.Path,
		Name:      acc.Name,
		ColorID:   acc.ColorID,
		Emoji:     acc.Emoji,
		Wallet:    acc.Wallet,
		Chat:      acc.Chat,
		Hidden:    acc.Hidden,
		Removed:   acc.Removed,
	}
}

func (m *Messenger) getMyInstallationMetadata() (*multidevice.InstallationMetadata, error) {
	installation, ok := m.allInstallations.Load(m.installationID)
	if !ok {
		return nil, errors.New("no installation found")
	}

	if installation.InstallationMetadata == nil {
		return nil, errors.New("no installation metadata")
	}

	return installation.InstallationMetadata, nil
}

func (m *Messenger) prepareSyncKeypairMessage(kp *accounts.Keypair) (*protobuf.SyncKeypair, error) {
	message := &protobuf.SyncKeypair{
		Clock:                   kp.Clock,
		KeyUid:                  kp.KeyUID,
		Name:                    kp.Name,
		Type:                    kp.Type.String(),
		DerivedFrom:             kp.DerivedFrom,
		LastUsedDerivationIndex: kp.LastUsedDerivationIndex,
		SyncedFrom:              kp.SyncedFrom,
	}

	if kp.SyncedFrom == "" {
		installationMetadata, err := m.getMyInstallationMetadata()
		if err != nil {
			return nil, err
		}
		message.SyncedFrom = installationMetadata.Name
	}

	for _, acc := range kp.Accounts {
		sAcc := m.prepareSyncAccountMessage(acc)
		if sAcc == nil {
			continue
		}

		message.Accounts = append(message.Accounts, sAcc)
	}

	return message, nil
}

func (m *Messenger) prepareSyncKeypairFullMessage(kp *accounts.Keypair) (*protobuf.SyncKeypairFull, error) {
	syncKpMsg, err := m.prepareSyncKeypairMessage(kp)
	if err != nil {
		return nil, err
	}

	syncKcMsgs, err := m.prepareSyncKeycardsMessage(kp.KeyUID)
	if err != nil {
		return nil, err
	}

	return &protobuf.SyncKeypairFull{
		Keypair:  syncKpMsg,
		Keycards: syncKcMsgs,
	}, nil
}

func (m *Messenger) syncWalletAccount(acc *accounts.Account, rawMessageHandler RawMessageHandler) error {
	if !m.hasPairedDevices() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, chat := m.getLastClockWithRelatedChat()

	message := m.prepareSyncAccountMessage(acc)

	encodedMessage, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_ACCOUNT,
		ResendAutomatically: true,
	}

	_, err = rawMessageHandler(ctx, rawMessage)
	return err
}

func (m *Messenger) syncKeypair(keypair *accounts.Keypair, fullKeypairSync bool, rawMessageHandler RawMessageHandler) (err error) {
	if !m.hasPairedDevices() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, chat := m.getLastClockWithRelatedChat()
	rawMessage := common.RawMessage{
		LocalChatID:         chat.ID,
		ResendAutomatically: true,
	}

	if fullKeypairSync {
		message, err := m.prepareSyncKeypairFullMessage(keypair)
		if err != nil {
			return err
		}

		rawMessage.MessageType = protobuf.ApplicationMetadataMessage_SYNC_FULL_KEYPAIR
		rawMessage.Payload, err = proto.Marshal(message)
		if err != nil {
			return err
		}
	} else {
		message, err := m.prepareSyncKeypairMessage(keypair)
		if err != nil {
			return err
		}

		rawMessage.MessageType = protobuf.ApplicationMetadataMessage_SYNC_KEYPAIR
		rawMessage.Payload, err = proto.Marshal(message)
		if err != nil {
			return err
		}
	}

	_, err = rawMessageHandler(ctx, rawMessage)
	return err
}
