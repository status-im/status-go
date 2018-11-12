package registry

import (
	"context"
	"crypto/ecdsa"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

type VerifierTestSuite struct {
	suite.Suite
	backend         *backends.SimulatedBackend
	privKey         *ecdsa.PrivateKey
	from            common.Address
	contractAddress common.Address
	registry        *Registry
	verifier        *Verifier
}

func TestVerifierTestSuite(t *testing.T) {
	suite.Run(t, &VerifierTestSuite{})
}

func (s *VerifierTestSuite) SetupTest() {
	s.setupAccount()
	s.setupBackendAndContract()

	var err error
	s.verifier, err = NewVerifier(s.backend, s.contractAddress)
	s.Require().NoError(err)
}

func (s *VerifierTestSuite) setupBackendAndContract() {
	var err error

	auth := bind.NewKeyedTransactor(s.privKey)
	alloc := make(core.GenesisAlloc)
	alloc[auth.From] = core.GenesisAccount{Balance: big.NewInt(133700000)}
	s.backend = backends.NewSimulatedBackend(alloc, math.MaxInt64)

	s.contractAddress, _, s.registry, err = DeployRegistry(auth, s.backend)
	s.Require().NoError(err)
	s.backend.Commit()
}

func (s *VerifierTestSuite) setupAccount() {
	var err error

	s.privKey, err = crypto.GenerateKey()
	s.Require().NoError(err)

	s.from = crypto.PubkeyToAddress(s.privKey.PublicKey)
}

func (s *VerifierTestSuite) add(nodeID enode.ID) {
	auth := bind.NewKeyedTransactor(s.privKey)
	_, err := s.registry.Add(auth, nodeID[:])
	s.Require().NoError(err)
	s.backend.Commit()
}

func (s *VerifierTestSuite) generateNodeID() enode.ID {
	k, err := crypto.GenerateKey()
	s.Require().NoError(err)

	return enode.PubkeyToIDV4(&k.PublicKey)
}

func (s *VerifierTestSuite) TestVerifyNode() {
	id := s.generateNodeID()
	res := s.verifier.VerifyNode(context.Background(), id)
	s.Require().False(res)

	s.add(id)

	res = s.verifier.VerifyNode(context.Background(), id)
	s.Require().True(res)
}
