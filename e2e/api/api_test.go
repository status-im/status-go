package api_test

import (
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/status-im/status-go/e2e"
	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/params"
	. "github.com/status-im/status-go/testing"
	"github.com/stretchr/testify/suite"
)

const (
	testChatID = "testChat"
)

func TestAPI(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}

type APITestSuite struct {
	suite.Suite
	api *api.StatusAPI
}

func (s *APITestSuite) SetupTest() {
	s.api = api.NewStatusAPI()
	s.NotNil(s.api)
}

func (s *APITestSuite) TestCHTUpdate() {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "cht-updates")
	s.NoError(err)
	defer os.RemoveAll(tmpDir) //nolint: errcheck

	configJSON := `{
		"NetworkId": ` + strconv.Itoa(params.RopstenNetworkID) + `,
		"DataDir": "` + tmpDir + `",
		"LogLevel": "INFO",
		"RPCEnabled": true
	}`

	_, err = params.LoadNodeConfig(configJSON)
	s.NoError(err)
	// TODO(tiabc): Test that CHT is really updated.
}

func (s *APITestSuite) TestRaceConditions() {
	cnt := 25
	progress := make(chan struct{}, cnt)
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	nodeConfig1, err := e2e.MakeTestNodeConfig(GetNetworkID())
	s.NoError(err)

	nodeConfig2, err := e2e.MakeTestNodeConfig(GetNetworkID())
	s.NoError(err)

	nodeConfigs := []*params.NodeConfig{nodeConfig1, nodeConfig2}

	var funcsToTest = []func(*params.NodeConfig){
		func(config *params.NodeConfig) {
			log.Info("StartNodeAsync()")
			_, err := s.api.StartNodeAsync(config)
			s.T().Logf("StartNodeAsync() for network: %d, error: %v", config.NetworkID, err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("StopNodeAsync()")
			_, err := s.api.StopNodeAsync()
			s.T().Logf("StopNodeAsync(), error: %v", err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("RestartNodeAsync()")
			_, err := s.api.RestartNodeAsync()
			s.T().Logf("RestartNodeAsync(), error: %v", err)
			progress <- struct{}{}
		},
		// TODO(adam): quarantined until it uses a different datadir
		// as otherwise it wipes out cached blockchain data.
		// func(config *params.NodeConfig) {
		// 	log.Info("ResetChainDataAsync()")
		// 	_, err := s.api.ResetChainDataAsync()
		// 	s.T().Logf("ResetChainDataAsync(), error: %v", err)
		// 	progress <- struct{}{}
		// },
	}

	// increase StartNode()/StopNode() population
	for i := 0; i < 5; i++ {
		funcsToTest = append(funcsToTest, funcsToTest[0], funcsToTest[1])
	}

	for i := 0; i < cnt; i++ {
		randConfig := nodeConfigs[rnd.Intn(len(nodeConfigs))]
		randFunc := funcsToTest[rnd.Intn(len(funcsToTest))]

		if rnd.Intn(100) > 75 { // introduce random delays
			time.Sleep(500 * time.Millisecond)
		}
		go randFunc(randConfig)
	}

	for range progress {
		cnt -= 1
		if cnt <= 0 {
			break
		}
	}

	time.Sleep(2 * time.Second) // so that we see some logs
	// just in case we have a node running
	s.api.StopNode() //nolint: errcheck
}

func (s *APITestSuite) TestCellsRemovedAfterSwitchAccount() {
	const itersCount = 5
	var (
		require   = s.Require()
		getChatId = func(id int) string {
			return testChatID + strconv.Itoa(id)
		}
	)

	config, err := e2e.MakeTestNodeConfig(GetNetworkID())
	require.NoError(err)
	err = s.api.StartNode(config)
	require.NoError(err)
	defer s.api.StopNode() //nolint: errcheck

	address1, _, _, err := s.api.AccountManager().CreateAccount(TestConfig.Account1.Password)
	require.NoError(err)

	address2, _, _, err := s.api.AccountManager().CreateAccount(TestConfig.Account2.Password)
	require.NoError(err)

	err = s.api.SelectAccount(address1, TestConfig.Account1.Password)
	require.NoError(err)

	for i := 0; i < itersCount; i++ {
		_, e := s.api.JailManager().CreateCell(getChatId(i))
		require.NoError(e)
	}

	err = s.api.SelectAccount(address2, TestConfig.Account2.Password)
	require.NoError(err)

	for i := 0; i < itersCount; i++ {
		_, e := s.api.JailManager().Cell(getChatId(i))
		require.Error(e)
	}
}

// TestLogoutRemovesCells we want be sure that
// cells will be removed after the API call "Logout"
func (s *APITestSuite) TestLogoutRemovesCells() {
	var (
		err     error
		require = s.Require()
	)

	config, err := e2e.MakeTestNodeConfig(GetNetworkID())
	require.NoError(err)
	err = s.api.StartNode(config)
	require.NoError(err)
	defer s.api.StopNode() //nolint: errcheck

	address1, _, _, err := s.api.AccountManager().CreateAccount(TestConfig.Account1.Password)
	require.NoError(err)

	err = s.api.SelectAccount(address1, TestConfig.Account1.Password)
	require.NoError(err)

	s.api.JailManager().CreateAndInitCell(testChatID)

	err = s.api.Logout()
	require.NoError(err)

	_, err = s.api.JailManager().Cell(testChatID)
	require.Error(err, "Expected that cells was removed")
}

func (s *APITestSuite) TestCreateChildAccount() bool {
	require := s.Require()

	config, err := e2e.MakeTestNodeConfig(GetNetworkID())
	require.NoError(err)
	err = s.api.StartNode(config)
	require.NoError(err)
	defer s.api.StopNode() //nolint: errcheck

	// to make sure that we start with empty account (which might get populated during previous tests)
	require.NoError(s.api.Logout())

	keyStore, err := s.api.NodeManager().AccountKeyStore()
	require.NoError(err)

	// create an account
	createAccountResponse, err := s.api.CreateAccount(TestConfig.Account1.Password)
	require.Empty(err, "could not create account: %s", err)

	address, pubKey, mnemonic := createAccountResponse.Address, createAccountResponse.PubKey, createAccountResponse.Mnemonic
	s.T().Logf("Account created: {address: %s, key: %s, mnemonic:%s}", address, pubKey, mnemonic)

	acct, err := common.ParseAccountString(address)
	require.NoError(err, "can not get account from address: %v", err)

	// obtain decrypted key, and make sure that extended key (which will be used as root for sub-accounts) is present
	_, key, err := keyStore.AccountDecryptedKey(acct, TestConfig.Account1.Password)
	require.NoError(err, "can not obtain decrypted account key: %v", err)
	require.NotNil(key.ExtendedKey, "CKD#2 has not been generated for new account")

	// try creating sub-account, w/o selecting main account i.e. w/o login to main account
	_, err = s.api.CreateChildAccount("", TestConfig.Account1.Password)
	require.EqualValues(account.ErrNoAccountSelected, err, "expected error is not returned (tried to create sub-account w/o login): %v", err)

	err = s.api.SelectAccount(address, TestConfig.Account1.Password)
	require.NoError(err, "Test failed: could not select account: %v", err)

	// try to create sub-account with wrong password
	_, err = s.api.CreateChildAccount("", "wrong password")
	require.EqualError(err, "cannot retrieve a valid key for a given account: could not decrypt key with given passphrase", "expected error is not returned (tried to create sub-account with wrong password): %v", err)

	// create sub-account (from implicit parent)
	createSubAccountResponse1, err := s.api.CreateChildAccount("", TestConfig.Account1.Password)
	require.NoError(err, "Test failed: could not select account: %v", err)

	// make sure that sub-account index automatically progresses
	createSubAccountResponse2, err := s.api.CreateChildAccount("", TestConfig.Account1.Password)
	require.NoError(err, "cannot create sub-account: %v", err)
	require.NotEqual(createSubAccountResponse1.Address, createSubAccountResponse2.Address, "sub-account index auto-increament failed")
	require.NotEqual(createSubAccountResponse1.PubKey, createSubAccountResponse2.PubKey, "sub-account index auto-increament failed")

	// create sub-account (from explicit parent)
	createSubAccountResponse3, err := s.api.CreateChildAccount(createSubAccountResponse2.Address, TestConfig.Account1.Password)
	require.NoError(err, "cannot create sub-account: %v", err)

	subAccount1, subAccount2, subAccount3 := createSubAccountResponse1.Address, createSubAccountResponse2.Address, createSubAccountResponse3.Address
	subPubKey1, subPubKey2, subPubKey3 := createSubAccountResponse1.PubKey, createSubAccountResponse2.PubKey, createSubAccountResponse3.PubKey

	require.NotEqual(subAccount1, subAccount3, "sub-account index auto-increament failed: subAccount1 == subAccount3")
	require.NotEqual(subPubKey1, subPubKey3, "sub-account index auto-increament failed: subPubKey1 == subPubKey3")
	require.NotEqual(subAccount2, subAccount3, "sub-account index auto-increament failed: subAccount2 == subAccount3")
	require.NotEqual(subPubKey2, subPubKey3, "sub-account index auto-increament failed: subPubKey2 == subPubKey3")

	return true
}

func (s *APITestSuite) TestRecoverAccount() bool {
	require := s.Require()

	config, err := e2e.MakeTestNodeConfig(GetNetworkID())
	require.NoError(err)
	err = s.api.StartNode(config)
	require.NoError(err)
	defer s.api.StopNode() //nolint: errcheck

	keyStore, _ := s.api.NodeManager().AccountKeyStore()

	// create an account
	accountInfo, err := s.api.CreateAccount(TestConfig.Account1.Password)
	require.NoError(err, "could not create account: %v", err)
	address := accountInfo.Address
	pubKey := accountInfo.PubKey
	mnemonic := accountInfo.Mnemonic
	s.T().Logf("Account created: {address: %s, key: %s, mnemonic:%s}", address, pubKey, mnemonic)

	// try recovering using password + mnemonic
	recoverAccountResponse, err := s.api.RecoverAccount(TestConfig.Account1.Password, mnemonic)
	require.NoError(err, "recover account failed: %v", err)

	addressCheck, pubKeyCheck := recoverAccountResponse.Address, recoverAccountResponse.PubKey
	require.Equal(address, addressCheck, "recover account details failed to pull the correct details for address")
	require.Equal(pubKey, pubKeyCheck, "recover account details failed to pull the correct details for pubKey")

	// now test recovering, but make sure that account/key file is removed i.e. simulate recovering on a new device
	account, err := common.ParseAccountString(address)
	require.NoError(err, "can not get account from address: %v", err)

	account, key, err := keyStore.AccountDecryptedKey(account, TestConfig.Account1.Password)
	require.NoError(err, "can not obtain decrypted account key: %v", err)
	extChild2String := key.ExtendedKey.String()

	err = keyStore.Delete(account, TestConfig.Account1.Password)
	require.NoError(err, "cannot remove accoun: %v", err)

	recoverAccountResponse, err = s.api.RecoverAccount(TestConfig.Account1.Password, mnemonic)
	require.NoError(err, "recover account failed (for non-cached account): %s", err)

	addressCheck, pubKeyCheck = recoverAccountResponse.Address, recoverAccountResponse.PubKey

	require.Equal(address, addressCheck, "recover account details failed to pull the correct details (for non-cached account) for address")
	require.Equal(pubKey, pubKeyCheck, "recover account details failed to pull the correct details (for non-cached account) for pubKey")

	// make sure that extended key exists and is imported ok too
	_, key, err = keyStore.AccountDecryptedKey(account, TestConfig.Account1.Password)
	require.NoError(err, "can not obtain decrypted account key: %v", err)
	require.Equal(extChild2String, key.ExtendedKey.String(), "CKD#2 key mismatch, expected: %s, got: %s", extChild2String, key.ExtendedKey.String())

	// make sure that calling import several times, just returns from cache (no error is expected)
	recoverAccountResponse, err = s.api.RecoverAccount(TestConfig.Account1.Password, mnemonic)
	require.NoError(err, "recover account failed (for non-cached account): %v", err)

	addressCheck, pubKeyCheck = recoverAccountResponse.Address, recoverAccountResponse.PubKey
	require.Equal(address, addressCheck, "recover account details failed to pull the correct details (for non-cached account) for address")
	require.Equal(pubKey, pubKeyCheck, "recover account details failed to pull the correct details (for non-cached account) for pubKey")

	// time to login with recovered data
	whisperService, err := s.api.NodeManager().WhisperService()
	require.NoError(err, "whisper service not running: %v", err)

	hasKeyPair := whisperService.HasKeyPair(pubKeyCheck)
	require.False(hasKeyPair, "identity already present in whisper")

	err = s.api.SelectAccount(addressCheck, TestConfig.Account1.Password)
	require.NoError(err, "Test failed: could not select account: %v", err)

	hasKeyPair = whisperService.HasKeyPair(pubKeyCheck)
	require.True(hasKeyPair, "identity not injected into whisper: %v", err)

	return true
}
