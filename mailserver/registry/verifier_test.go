package registry

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/stretchr/testify/suite"
)

type VerifierTestSuite struct {
	suite.Suite
	backend         bind.ContractBackend
	privKey         *ecdsa.PrivateKey
	from            common.Address
	contractAddress common.Address
	registry        *Registry
	verifier        *RegistryVerifier
}

func (s *VerifierTestSuite) SetupTest() {
	s.setupAccount()
	s.setupBackendAndContract()

	var err error
	s.verifier, err = NewVerifier(s.backend, s.contractAddress)
	s.Require().NoError(err)
	s.verifier.pendingOpt = true
}

func (s *VerifierTestSuite) setupBackendAndContract() {
	var err error

	auth := bind.NewKeyedTransactor(s.privKey)
	alloc := make(core.GenesisAlloc)
	alloc[auth.From] = core.GenesisAccount{Balance: big.NewInt(133700000)}
	s.backend = backends.NewSimulatedBackend(alloc)

	auth = s.newKeyedTransactor()

	s.contractAddress, _, s.registry, err = DeployRegistry(auth, s.backend)
	s.Require().NoError(err)
}

func (s *VerifierTestSuite) setupAccount() {
	var err error

	s.privKey, err = crypto.GenerateKey()
	s.Require().NoError(err)

	pubKey := s.privKey.Public()
	publicKeyECDSA, ok := pubKey.(*ecdsa.PublicKey)
	s.Require().True(ok)

	s.from = crypto.PubkeyToAddress(*publicKeyECDSA)
}

func (s *VerifierTestSuite) add(nodeID discover.NodeID) {
	auth := s.newKeyedTransactor()
	_, err := s.registry.Add(auth, nodeID[:])
	s.Require().NoError(err)
}

func (s *VerifierTestSuite) newKeyedTransactor() *bind.TransactOpts {
	nonce, err := s.backend.PendingNonceAt(context.Background(), s.from)
	s.Require().NoError(err)

	gasPrice, err := s.backend.SuggestGasPrice(context.Background())
	s.Require().NoError(err)

	auth := bind.NewKeyedTransactor(s.privKey)
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)
	auth.GasLimit = uint64(600000)
	auth.GasPrice = gasPrice

	return auth
}

func (s *VerifierTestSuite) generateNodeID() discover.NodeID {
	k, err := crypto.GenerateKey()
	s.Require().NoError(err)

	pub := k.PublicKey

	var nodeID discover.NodeID
	pbytes := elliptic.Marshal(pub.Curve, pub.X, pub.Y)
	s.Require().Equal(len(nodeID), len(pbytes)-1)
	copy(nodeID[:], pbytes[1:])

	return nodeID
}

func (s *VerifierTestSuite) TestVerifyNode() {
	id := s.generateNodeID()
	res := s.verifier.VerifyNode(context.Background(), id)
	s.Require().False(res)

	s.add(id)

	res = s.verifier.VerifyNode(context.Background(), id)
	s.Require().True(res)
}

func TestVerifierTestSuite(t *testing.T) {
	suite.Run(t, &VerifierTestSuite{})
}
