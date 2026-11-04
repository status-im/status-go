//go:build include_onchain_tests
// +build include_onchain_tests

package rln

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/waku-org/go-zerokit-rln/rln"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/suite"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/contracts"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager/dynamic"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/keystore"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/web3"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

const keystorePassword = "test"

func TestWakuRLNRelayDynamicSuite(t *testing.T) {
	suite.Run(t, new(WakuRLNRelayDynamicSuite))
}

type WakuRLNRelayDynamicSuite struct {
	suite.Suite
	web3Config *web3.Config
	u1PrivKey  *ecdsa.PrivateKey
	u2PrivKey  *ecdsa.PrivateKey
}

func TempFileName(prefix, suffix string) string {
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	return filepath.Join(os.TempDir(), prefix+hex.EncodeToString(randBytes)+suffix)
}

func (s *WakuRLNRelayDynamicSuite) SetupTest() {

	clientAddr := os.Getenv("GANACHE_NETWORK_RPC_URL")
	if clientAddr == "" {
		clientAddr = "ws://localhost:8545"
	}

	backend, err := ethclient.Dial(clientAddr)
	s.Require().NoError(err)
	defer backend.Close()

	chainID, err := backend.ChainID(context.TODO())
	s.Require().NoError(err)

	// TODO: obtain account list from ganache mnemonic or from eth_accounts

	s.u1PrivKey, err = crypto.ToECDSA(common.FromHex("0x156ec84a451d8a2d0062993242b6c4e863647f5544ff8030f23578d4142f43f8"))
	s.Require().NoError(err)
	s.u2PrivKey, err = crypto.ToECDSA(common.FromHex("0xa00da43843ad6b5161ddbace48f293ac3f82f8a8257af34de4c32900bb6e9a97"))
	s.Require().NoError(err)

	// Deploying contracts
	auth, err := bind.NewKeyedTransactorWithChainID(s.u1PrivKey, chainID)
	s.Require().NoError(err)

	poseidonHasherAddr, _, _, err := contracts.DeployPoseidonHasher(auth, backend)
	s.Require().NoError(err)

	registryAddress, tx, rlnRegistry, err := contracts.DeployRLNRegistry(auth, backend, poseidonHasherAddr)
	s.Require().NoError(err)
	txReceipt, err := bind.WaitMined(context.TODO(), backend, tx)
	s.Require().NoError(err)
	s.Require().Equal(txReceipt.Status, types.ReceiptStatusSuccessful)

	tx, err = rlnRegistry.NewStorage(auth)
	s.Require().NoError(err)
	txReceipt, err = bind.WaitMined(context.TODO(), backend, tx)
	s.Require().NoError(err)
	s.Require().Equal(txReceipt.Status, types.ReceiptStatusSuccessful)

	s.web3Config = web3.NewConfig(clientAddr, registryAddress)
	err = s.web3Config.Build(context.TODO())
	s.Require().NoError(err)
}

func (s *WakuRLNRelayDynamicSuite) generateCredentials(rlnInstance *rln.RLN) *rln.IdentityCredential {
	identityCredential, err := rlnInstance.MembershipKeyGen()
	s.Require().NoError(err)
	return identityCredential
}

func (s *WakuRLNRelayDynamicSuite) register(appKeystore *keystore.AppKeystore, identityCredential *rln.IdentityCredential, privKey *ecdsa.PrivateKey) rln.MembershipIndex {
	membershipFee, err := s.web3Config.RLNContract.MEMBERSHIPDEPOSIT(&bind.CallOpts{Context: context.TODO()})
	s.Require().NoError(err)

	auth, err := bind.NewKeyedTransactorWithChainID(privKey, s.web3Config.ChainID)
	s.Require().NoError(err)

	auth.Value = membershipFee
	auth.Context = context.TODO()
	tx, err := s.web3Config.RegistryContract.Register(auth, s.web3Config.RLNContract.StorageIndex, rln.Bytes32ToBigInt(identityCredential.IDCommitment))
	s.Require().NoError(err)

	txReceipt, err := bind.WaitMined(context.TODO(), s.web3Config.ETHClient, tx)
	s.Require().NoError(err)
	s.Require().Equal(txReceipt.Status, types.ReceiptStatusSuccessful)

	evt, err := s.web3Config.RLNContract.ParseMemberRegistered(*txReceipt.Logs[0])
	s.Require().NoError(err)

	membershipIndex := rln.MembershipIndex(uint(evt.Index.Int64()))

	membershipCredential := keystore.MembershipCredentials{
		IdentityCredential:     identityCredential,
		TreeIndex:              membershipIndex,
		MembershipContractInfo: keystore.NewMembershipContractInfo(s.web3Config.ChainID, s.web3Config.RegistryContract.Address),
	}

	err = appKeystore.AddMembershipCredentials(membershipCredential, keystorePassword)
	s.Require().NoError(err)

	return membershipIndex
}

func (s *WakuRLNRelayDynamicSuite) TestDynamicGroupManagement() {
	// Create a RLN instance
	rlnInstance, err := rln.NewRLN()
	s.Require().NoError(err)

	rt := group_manager.NewMerkleRootTracker(5, rlnInstance)

	u1Credentials := s.generateCredentials(rlnInstance)
	appKeystore, err := keystore.New(s.tmpKeystorePath(), dynamic.RLNAppInfo, utils.Logger())
	s.Require().NoError(err)

	membershipIndex := s.register(appKeystore, u1Credentials, s.u1PrivKey)

	gm, err := dynamic.NewDynamicGroupManager(s.web3Config.ETHClientAddress, s.web3Config.RegistryContract.Address, &membershipIndex, appKeystore, keystorePassword, prometheus.DefaultRegisterer, rlnInstance, rt, utils.Logger())
	s.Require().NoError(err)

	// initialize the WakuRLNRelay
	rlnRelay := &WakuRLNRelay{
		Details: group_manager.Details{
			RootTracker:  rt,
			GroupManager: gm,
			RLN:          rlnInstance,
		},
		log:          utils.Logger(),
		nullifierLog: NewNullifierLog(context.TODO(), utils.Logger()),
	}

	err = rlnRelay.Start(context.TODO())
	s.Require().NoError(err)

	u2Credentials := s.generateCredentials(rlnInstance)
	appKeystore2, err := keystore.New(s.tmpKeystorePath(), dynamic.RLNAppInfo, utils.Logger())
	s.Require().NoError(err)

	membershipIndex = s.register(appKeystore2, u2Credentials, s.u2PrivKey)

	time.Sleep(1 * time.Second)

	treeCommitment, err := rlnInstance.GetLeaf(membershipIndex)
	s.Require().NoError(err)
	s.Require().Equal(u2Credentials.IDCommitment, treeCommitment)
}

func (s *WakuRLNRelayDynamicSuite) TestInsertKeyMembershipContract() {
	// Create a RLN instance
	rlnInstance, err := rln.NewRLN()
	s.Require().NoError(err)

	credentials1 := s.generateCredentials(rlnInstance)
	credentials2 := s.generateCredentials(rlnInstance)
	credentials3 := s.generateCredentials(rlnInstance)

	appKeystore, err := keystore.New(s.tmpKeystorePath(), dynamic.RLNAppInfo, utils.Logger())
	s.Require().NoError(err)

	s.register(appKeystore, credentials1, s.u1PrivKey)

	// Batch Register
	auth, err := bind.NewKeyedTransactorWithChainID(s.u2PrivKey, s.web3Config.ChainID)
	s.Require().NoError(err)

	membershipFee, err := s.web3Config.RLNContract.MEMBERSHIPDEPOSIT(&bind.CallOpts{Context: context.TODO()})
	s.Require().NoError(err)

	auth.Value = membershipFee.Mul(big.NewInt(2), membershipFee)
	auth.Context = context.TODO()

	tx, err := s.web3Config.RegistryContract.Register1(auth, s.web3Config.RLNContract.StorageIndex, []*big.Int{rln.Bytes32ToBigInt(credentials2.IDCommitment), rln.Bytes32ToBigInt(credentials3.IDCommitment)})
	s.Require().NoError(err)

	txReceipt, err := bind.WaitMined(context.TODO(), s.web3Config.ETHClient, tx)
	s.Require().NoError(err)
	s.Require().Equal(txReceipt.Status, types.ReceiptStatusSuccessful)
}

func (s *WakuRLNRelayDynamicSuite) TestMerkleTreeConstruction() {
	// Create a RLN instance
	rlnInstance, err := rln.NewRLN()
	s.Require().NoError(err)

	credentials1 := s.generateCredentials(rlnInstance)
	credentials2 := s.generateCredentials(rlnInstance)

	err = rlnInstance.InsertMembers(0, []rln.IDCommitment{credentials1.IDCommitment, credentials2.IDCommitment})
	s.Require().NoError(err)

	//  get the Merkle root
	expectedRoot, err := rlnInstance.GetMerkleRoot()
	s.Require().NoError(err)

	// register the members to the contract
	appKeystore, err := keystore.New(s.tmpKeystorePath(), dynamic.RLNAppInfo, utils.Logger())
	s.Require().NoError(err)

	membershipIndex := s.register(appKeystore, credentials1, s.u1PrivKey)
	membershipIndex = s.register(appKeystore, credentials2, s.u1PrivKey)

	rlnInstance, rootTracker, err := GetRLNInstanceAndRootTracker(s.tmpRLNDBPath())
	s.Require().NoError(err)
	// mount the rln relay protocol in the on-chain/dynamic mode
	gm, err := dynamic.NewDynamicGroupManager(s.web3Config.ETHClientAddress, s.web3Config.RegistryContract.Address, &membershipIndex, appKeystore, keystorePassword, prometheus.DefaultRegisterer, rlnInstance, rootTracker, utils.Logger())
	s.Require().NoError(err)

	rlnRelay := New(group_manager.Details{
		RLN:          rlnInstance,
		RootTracker:  rootTracker,
		GroupManager: gm,
	}, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s.Require().NoError(err)

	err = rlnRelay.Start(context.TODO())
	s.Require().NoError(err)

	// wait for the event to reach the group handler
	time.Sleep(2 * time.Second)

	// rln pks are inserted into the rln peer's Merkle tree and the resulting root
	// is expected to be the same as the calculatedRoot i.e., the one calculated outside of the mountRlnRelayDynamic proc
	calculatedRoot, err := rlnRelay.RLN.GetMerkleRoot()
	s.Require().NoError(err)
	s.Require().Equal(expectedRoot, calculatedRoot)
}

func (s *WakuRLNRelayDynamicSuite) TestCorrectRegistrationOfPeers() {
	// Creating an RLN instance (just for generating membership keys)
	rlnInstance, err := rln.NewRLN()
	s.Require().NoError(err)

	// Node 1 ============================================================

	// Register credentials1 in contract and keystore1
	credentials1 := s.generateCredentials(rlnInstance)
	appKeystore, err := keystore.New(s.tmpKeystorePath(), dynamic.RLNAppInfo, utils.Logger())
	s.Require().NoError(err)
	membershipGroupIndex := s.register(appKeystore, credentials1, s.u1PrivKey)

	// mount the rln relay protocol in the on-chain/dynamic mode
	rootInstance, rootTracker, err := GetRLNInstanceAndRootTracker(s.tmpRLNDBPath())
	s.Require().NoError(err)
	gm1, err := dynamic.NewDynamicGroupManager(s.web3Config.ETHClientAddress, s.web3Config.RegistryContract.Address, &membershipGroupIndex, appKeystore, keystorePassword, prometheus.DefaultRegisterer, rootInstance, rootTracker, utils.Logger())
	s.Require().NoError(err)

	rlnRelay1 := New(group_manager.Details{
		GroupManager: gm1,
		RootTracker:  rootTracker,
		RLN:          rootInstance,
	}, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())

	err = rlnRelay1.Start(context.TODO())
	s.Require().NoError(err)

	// Node 2 ============================================================

	// Register credentials2 in contract and keystore2
	credentials2 := s.generateCredentials(rlnInstance)
	appKeystore2, err := keystore.New(s.tmpKeystorePath(), dynamic.RLNAppInfo, utils.Logger())
	s.Require().NoError(err)

	membershipGroupIndex = s.register(appKeystore2, credentials2, s.u2PrivKey)

	// mount the rln relay protocol in the on-chain/dynamic mode
	rootInstance, rootTracker, err = GetRLNInstanceAndRootTracker(s.tmpRLNDBPath())
	s.Require().NoError(err)
	gm2, err := dynamic.NewDynamicGroupManager(s.web3Config.ETHClientAddress, s.web3Config.RegistryContract.Address, &membershipGroupIndex, appKeystore2, keystorePassword, prometheus.DefaultRegisterer, rootInstance, rootTracker, utils.Logger())
	s.Require().NoError(err)

	rlnRelay2 := New(group_manager.Details{
		GroupManager: gm2,
		RootTracker:  rootTracker,
		RLN:          rootInstance,
	}, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	err = rlnRelay2.Start(context.TODO())
	s.Require().NoError(err)

	// ==================================
	// the two nodes should be registered into the contract
	// since nodes are spun up sequentially
	// the first node has index 0 whereas the second node gets index 1
	idx1 := rlnRelay1.GroupManager.MembershipIndex()
	idx2 := rlnRelay2.GroupManager.MembershipIndex()
	s.Require().NoError(err)
	s.Require().Equal(rln.MembershipIndex(0), idx1)
	s.Require().Equal(rln.MembershipIndex(1), idx2)
}

func (s *WakuRLNRelayDynamicSuite) tmpKeystorePath() string {
	keystoreDir, err := os.MkdirTemp("", "keystore_dir")
	s.Require().NoError(err)
	return filepath.Join(keystoreDir, "keystore.json")
}

func (s *WakuRLNRelayDynamicSuite) tmpRLNDBPath() string {
	dbPath, err := os.MkdirTemp("", "rln_db")
	s.Require().NoError(err)
	return dbPath
}

func (s *WakuRLNRelayDynamicSuite) TestDynamicGroupManagerGetters() {
	// Create a RLN instance
	rlnInstance, err := rln.NewRLN()
	s.Require().NoError(err)

	ctx := context.Background()

	rt := group_manager.NewMerkleRootTracker(5, rlnInstance)

	u1Credentials := s.generateCredentials(rlnInstance)
	appKeystore, err := keystore.New(s.tmpKeystorePath(), dynamic.RLNAppInfo, utils.Logger())
	s.Require().NoError(err)

	membershipIndex := s.register(appKeystore, u1Credentials, s.u1PrivKey)

	gm, err := dynamic.NewDynamicGroupManager(s.web3Config.ETHClientAddress, s.web3Config.RegistryContract.Address, &membershipIndex, appKeystore, keystorePassword, prometheus.DefaultRegisterer, rlnInstance, rt, utils.Logger())
	s.Require().NoError(err)

	// initialize the WakuRLNRelay
	rlnRelay := &WakuRLNRelay{
		Details: group_manager.Details{
			RootTracker:  rt,
			GroupManager: gm,
			RLN:          rlnInstance,
		},
		log:          utils.Logger(),
		nullifierLog: NewNullifierLog(ctx, utils.Logger()),
	}

	err = rlnRelay.Start(ctx)
	s.Require().NoError(err)

	// Test IdentityCredentials
	_, err = gm.IdentityCredentials()
	s.Require().NoError(err)

	// Test MembershipIndex
	mIndex := gm.MembershipIndex()
	s.Require().Equal(mIndex, uint(membershipIndex))

	// Test IsReady
	isReady, err := gm.IsReady(ctx)
	s.Require().NoError(err)
	s.Require().True(isReady)

	// Test Stop / gm.Start happened as a part of rlnRelay.Start
	err = gm.Stop()
	s.Require().NoError(err)

}
