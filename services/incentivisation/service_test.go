package incentivisation

import (
	"bytes"
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/stretchr/testify/suite"
)

var (
	nodeOne = []byte{0x01}
	nodeTwo = []byte{0x02}
)

type MockContract struct {
	currentSession *big.Int
	activeNodes    [][]byte
	inactiveNodes  [][]byte
	votes          []Vote
}

type Vote struct {
	joinNodes   []gethcommon.Address
	removeNodes []gethcommon.Address
}

type MockWhisper struct {
	sentMessages   []whisper.NewMessage
	filterMessages []*whisper.Message
}

func BuildMockContract() *MockContract {
	contract := &MockContract{
		currentSession: big.NewInt(0),
	}

	contract.activeNodes = append(contract.activeNodes, nodeOne)
	contract.inactiveNodes = append(contract.activeNodes, nodeTwo)
	return contract
}

func (c *MockContract) Vote(opts *bind.TransactOpts, joinNodes []gethcommon.Address, removeNodes []gethcommon.Address) (*types.Transaction, error) {

	return nil, nil
}

func (c *MockContract) VoteSync(opts *bind.TransactOpts, joinNodes []gethcommon.Address, removeNodes []gethcommon.Address) (*types.Transaction, error) {
	c.votes = append(c.votes, Vote{
		joinNodes:   joinNodes,
		removeNodes: removeNodes,
	})
	return nil, nil
}

func (c *MockContract) GetCurrentSession(opts *bind.CallOpts) (*big.Int, error) {
	return c.currentSession, nil
}
func (c *MockContract) Registered(opts *bind.CallOpts, publicKey []byte) (bool, error) {

	for _, e := range c.activeNodes {
		if bytes.Equal(publicKey, e) {
			return true, nil
		}
	}

	for _, e := range c.inactiveNodes {
		if bytes.Equal(publicKey, e) {
			return true, nil
		}
	}

	return false, nil
}

func (c *MockContract) RegisterNode(opts *bind.TransactOpts, publicKey []byte, ip uint32, port uint16) (*types.Transaction, error) {
	c.inactiveNodes = append(c.inactiveNodes, publicKey)
	return nil, nil
}
func (c *MockContract) ActiveNodeCount(opts *bind.CallOpts) (*big.Int, error) {
	return big.NewInt(int64(len(c.activeNodes))), nil
}
func (c *MockContract) InactiveNodeCount(opts *bind.CallOpts) (*big.Int, error) {
	return big.NewInt(int64(len(c.inactiveNodes))), nil
}

func (c *MockContract) GetNode(opts *bind.CallOpts, index *big.Int) ([]byte, uint32, uint16, uint32, uint32, error) {
	return c.activeNodes[index.Int64()], 0, 0, 0, 0, nil
}
func (c *MockContract) GetInactiveNode(opts *bind.CallOpts, index *big.Int) ([]byte, uint32, uint16, uint32, uint32, error) {
	return c.inactiveNodes[index.Int64()], 0, 0, 0, 0, nil
}

func (w *MockWhisper) Post(ctx context.Context, req whisper.NewMessage) (hexutil.Bytes, error) {
	w.sentMessages = append(w.sentMessages, req)
	return nil, nil
}
func (w *MockWhisper) NewMessageFilter(req whisper.Criteria) (string, error) {
	return "", nil
}
func (w *MockWhisper) AddPrivateKey(ctx context.Context, privateKey hexutil.Bytes) (string, error) {
	return "", nil
}
func (w *MockWhisper) DeleteKeyPair(ctx context.Context, key string) (bool, error) {
	return true, nil
}
func (w *MockWhisper) GenerateSymKeyFromPassword(ctx context.Context, passwd string) (string, error) {
	return "", nil
}
func (w *MockWhisper) GetFilterMessages(id string) ([]*whisper.Message, error) {
	return w.filterMessages, nil
}

func TestIncentivisationSuite(t *testing.T) {
	suite.Run(t, new(IncentivisationSuite))
}

type IncentivisationSuite struct {
	suite.Suite
	service      *Service
	mockWhisper  *MockWhisper
	mockContract *MockContract
}

func (s *IncentivisationSuite) SetupTest() {
	privateKey, err := crypto.GenerateKey()
	config := &ServiceConfig{
		IP: "192.168.1.1",
	}
	contract := BuildMockContract()
	if err != nil {
		panic(err)
	}
	w := &MockWhisper{}
	s.service = New(privateKey, w, config, contract)
	s.mockWhisper = w
	s.mockContract = contract
}

func (s *IncentivisationSuite) TestStart() {
	err := s.service.Start(nil)
	s.Require().NoError(err)
}

func (s *IncentivisationSuite) TestPerform() {
	err := s.service.Start(nil)
	s.Require().NoError(err)

	err = s.service.perform()
	s.Require().NoError(err)

	// It registers with the contract if not registered
	registered, err := s.service.registered()
	s.Require().NoError(err)
	s.Require().Equal(true, registered)

	now := time.Now().Unix()
	// Add some envelopes
	s.mockWhisper.filterMessages = []*whisper.Message{
		{
			// We strip the first byte when processing
			Sig:       append(nodeOne, nodeOne[0]),
			Timestamp: uint32(now - pingIntervalAllowance),
		},
		{
			Sig:       append(nodeOne, nodeOne[0]),
			Timestamp: uint32(now - (pingIntervalAllowance * 2)),
		},
		{
			Sig:       append(nodeTwo, nodeTwo[0]),
			Timestamp: uint32(now - (pingIntervalAllowance * 2)),
		},
	}

	// It publishes a ping on whisper
	s.Require().Equal(1, len(s.mockWhisper.sentMessages))

	// It should not vote
	s.Require().Equal(0, len(s.mockContract.votes))

	// We increase the session
	s.mockContract.currentSession = s.mockContract.currentSession.Add(s.mockContract.currentSession, big.NewInt(1))

	// We perform again
	err = s.service.perform()
	s.Require().NoError(err)

	// it should now vote
	s.Require().Equal(1, len(s.mockContract.votes))
	// Node one should have been voted up
	s.Require().Equal(1, len(s.mockContract.votes[0].joinNodes))
	s.Require().Equal(publicKeyBytesToAddress(nodeOne), s.mockContract.votes[0].joinNodes[0])
	// Node two should have been voted down
	s.Require().Equal(1, len(s.mockContract.votes[0].removeNodes))
	s.Require().Equal(publicKeyBytesToAddress(nodeTwo), s.mockContract.votes[0].removeNodes[0])

}
