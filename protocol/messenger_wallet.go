package protocol

import (
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

var checkBalancesInterval = time.Minute * 10

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

func (m *Messenger) SaveAccount(acc *accounts.Account) error {
	clock, _ := m.getLastClockWithRelatedChat()
	acc.Clock = clock

	err := m.settings.SaveAccounts([]*accounts.Account{acc})
	if err != nil {
		return err
	}
	return m.syncWallets([]*accounts.Account{acc}, m.dispatchMessage)
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

	accs := []*accounts.Account{acc}
	err = m.syncWallets(accs, m.dispatchMessage)
	if err != nil {
		return err
	}

	chat.LastClockValue = clock
	return m.saveChat(chat)
}

func (m *Messenger) prepareSyncWalletAccountsMessage(accs []*accounts.Account) *protobuf.SyncWalletAccounts {
	accountMessages := make([]*protobuf.SyncWalletAccount, 0)
	for _, acc := range accs {
		if acc.Chat {
			continue
		}

		syncMessage := &protobuf.SyncWalletAccount{
			Clock:                   acc.Clock,
			Address:                 acc.Address.Bytes(),
			Wallet:                  acc.Wallet,
			Chat:                    acc.Chat,
			Type:                    acc.Type.String(),
			Storage:                 acc.Storage,
			Path:                    acc.Path,
			PublicKey:               acc.PublicKey,
			Name:                    acc.Name,
			Color:                   acc.Color,
			Hidden:                  acc.Hidden,
			Removed:                 acc.Removed,
			Emoji:                   acc.Emoji,
			DerivedFrom:             acc.DerivedFrom,
			KeyUid:                  acc.KeyUID,
			KeypairName:             acc.KeypairName,
			LastUsedDerivationIndex: acc.LastUsedDerivationIndex,
		}

		accountMessages = append(accountMessages, syncMessage)
	}

	return &protobuf.SyncWalletAccounts{
		Accounts: accountMessages,
	}
}

// syncWallets syncs all wallets with paired devices
func (m *Messenger) syncWallets(accs []*accounts.Account, rawMessageHandler RawMessageHandler) error {
	if !m.hasPairedDevices() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, chat := m.getLastClockWithRelatedChat()

	message := m.prepareSyncWalletAccountsMessage(accs)

	encodedMessage, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_WALLET_ACCOUNT,
		ResendAutomatically: true,
	}

	_, err = rawMessageHandler(ctx, rawMessage)
	return err
}
