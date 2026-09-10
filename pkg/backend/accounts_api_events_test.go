package backend

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gethcommon "github.com/ethereum/go-ethereum/common"

	accsmanagementcommon "github.com/status-im/status-go/internal/accounts-management/common"
	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	types "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/protocol/requests"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/services/accounts"
	"github.com/status-im/status-go/pkg/services/accounts/accountsevent"
)

func setupLoggedInBackendForAccountsEvents(t *testing.T) *setupContext {
	testContext := setupTestContext(t, testPassword, false, false, true)

	request := &requests.CreateAccount{
		RootDataDir:   testContext.config.RootDataDir,
		Password:      testPassword,
		KdfIterations: 1,
	}

	_, err := testContext.backend.StartNodeWithChatKeyOrMnemonic(request, testContext.mnemonic, nil, false)
	require.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, testContext.backend.Logout())
		assert.NoError(t, testContext.backend.StopNode())
	})

	return testContext
}

func addLedgerKeypairViaAPI(t *testing.T, accountsAPI *accounts.API, ledgerAddress gethcommon.Address) *accsmanagementtypes.Keypair {
	walletAccounts := []*accsmanagementtypes.Account{
		{
			KeyUID:   "ledger-kp-uid",
			Path:     accsmanagementcommon.PathWalletRoot + "/0",
			Address:  types.Address(ledgerAddress),
			Operable: accsmanagementtypes.AccountNonOperable,
			Name:     "Ledger account",
			Emoji:    "emoji",
			ColorID:  "blue",
		},
	}

	addedKeypair, err := accountsAPI.AddKeypairStoredToColdWallet(context.Background(), "ledger-kp-uid", "0xmaster-address",
		"Ledger keypair", "ledger-wallet-xpub", accsmanagementtypes.ColdWalletTypeLedger, walletAccounts)
	require.NoError(t, err)
	require.NotNil(t, addedKeypair)

	return addedKeypair
}

func TestAddKeypairStoredToColdWalletPublishesAccountsAddedEvent(t *testing.T) {
	testContext := setupLoggedInBackendForAccountsEvents(t)
	statusNode := testContext.backend.statusNode
	accountsAPI := statusNode.AccountService().AccountsAPI()

	addedCh, unsub := pubsub.Subscribe[accountsevent.AccountsAddedEvent](statusNode.AccountsPublisher(), 4)
	defer unsub()

	ledgerAddress := gethcommon.HexToAddress("0x1000000000000000000000000000000000000001")
	addLedgerKeypairViaAPI(t, accountsAPI, ledgerAddress)

	select {
	case event := <-addedCh:
		require.Equal(t, []gethcommon.Address{ledgerAddress}, event.Accounts,
			"Expected the added event to carry the new cold-wallet address because wallet trackers start watching from it")
	case <-time.After(5 * time.Second):
		t.Fatal("Expected an AccountsAddedEvent because adding a cold-wallet keypair must trigger wallet trackers for its addresses")
	}
}

func TestDeleteColdWalletKeypairPublishesAccountsRemovedEvent(t *testing.T) {
	testContext := setupLoggedInBackendForAccountsEvents(t)
	statusNode := testContext.backend.statusNode
	accountsAPI := statusNode.AccountService().AccountsAPI()

	ledgerAddress := gethcommon.HexToAddress("0x1000000000000000000000000000000000000001")
	addLedgerKeypairViaAPI(t, accountsAPI, ledgerAddress)

	removedCh, unsub := pubsub.Subscribe[accountsevent.AccountsRemovedEvent](statusNode.AccountsPublisher(), 4)
	defer unsub()

	err := accountsAPI.DeleteKeypair(context.Background(), "ledger-kp-uid", "")
	require.NoError(t, err,
		"Expected the delete to succeed without a password because a cold-wallet keypair has no keystore files to verify")

	select {
	case event := <-removedCh:
		require.Equal(t, []gethcommon.Address{ledgerAddress}, event.Accounts,
			"Expected the removed event to carry the deleted cold-wallet address because wallet trackers must stop watching it")
	case <-time.After(5 * time.Second):
		t.Fatal("Expected an AccountsRemovedEvent because deleting a cold-wallet keypair must stop wallet trackers for its addresses")
	}
}
