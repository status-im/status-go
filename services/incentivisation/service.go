package incentivisation

import (
	"bytes"
	"context"

	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
	"net"
	"sort"

	whisper "github.com/status-im/whisper/whisperv6"
	"time"
)

const (
	gasLimit              = 1001000
	pingIntervalAllowance = 240
	tickerInterval        = 30
	defaultTopic          = "status-incentivisation-topic"
)

type Enode struct {
	PublicKey      []byte
	IP             net.IP
	Port           uint16
	JoiningSession uint32
	ActiveSession  uint32
	Active         bool
}

func formatEnodeURL(publicKey string, ip string, port uint16) string {
	return fmt.Sprintf("enode://%s:%s:%d", publicKey, ip, port)
}

func (n *Enode) toEnodeURL() string {
	return formatEnodeURL(n.PublicKeyString(), n.IP.String(), n.Port)
}

func (n *Enode) PublicKeyString() string {
	return hex.EncodeToString(n.PublicKey)
}

type Whisper interface {
	Post(ctx context.Context, req whisper.NewMessage) (hexutil.Bytes, error)
	NewMessageFilter(req whisper.Criteria) (string, error)
	AddPrivateKey(ctx context.Context, privateKey hexutil.Bytes) (string, error)
	DeleteKeyPair(ctx context.Context, key string) (bool, error)
	GenerateSymKeyFromPassword(ctx context.Context, passwd string) (string, error)
	GetFilterMessages(id string) ([]*whisper.Message, error)
}

type ServiceConfig struct {
	RPCEndpoint     string
	ContractAddress string
	IP              string
	Port            uint16
}

type Service struct {
	w               Whisper
	whisperKeyID    string
	whisperSymKeyID string
	whisperFilterID string
	nodes           map[string]*Enode
	ticker          *time.Ticker
	quit            chan struct{}
	config          *ServiceConfig
	contract        Contract
	privateKey      *ecdsa.PrivateKey
	log             log.Logger
	// The first round we will not be voting, as we might have incomplete data
	initialSession uint64
	// The current session
	currentSession uint64
	whisperPings   map[string][]uint32
}

// New returns a new incentivization Service
func New(prv *ecdsa.PrivateKey, w Whisper, config *ServiceConfig, contract Contract) *Service {
	logger := log.New("package", "status-go/incentivisation/service")
	return &Service{
		w:            w,
		config:       config,
		privateKey:   prv,
		log:          logger,
		contract:     contract,
		nodes:        make(map[string]*Enode),
		whisperPings: make(map[string][]uint32),
	}
}

// Protocols returns a new protocols list. In this case, there are none.
func (s *Service) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

// APIs returns a list of new APIs.
func (s *Service) APIs() []rpc.API {
	apis := []rpc.API{
		{
			Namespace: "incentivisation",
			Version:   "1.0",
			Service:   NewAPI(s),
			Public:    true,
		},
	}
	return apis
}

// checkRegistered checks that a node is registered with the contract
func (s *Service) checkRegistered() error {
	registered, err := s.registered()
	if err != nil {
		s.log.Error("error querying contract", "registered", err)
		return err
	}

	if registered {
		s.log.Debug("Already registered")
		return nil
	}
	_, err = s.register()
	if err != nil {
		s.log.Error("error querying contract", "registered", err)
		return err
	}
	return nil
}

// ensureSession checks if we are in a new session and updates the session if so
func (s *Service) ensureSession() (bool, error) {
	session, err := s.GetCurrentSession()
	if err != nil {
		s.log.Error("failed to get current session", "err", err)
		return false, err
	}

	if session != s.currentSession {
		s.currentSession = session
		return true, nil
	}
	return false, nil
}

// checkPings checks we have received the expected pings since it was last called
func (s *Service) checkPings() map[string]bool {
	result := make(map[string]bool)
	now := time.Now().Unix()
	s.log.Debug("checking votes", "votes", s.whisperPings)
	for enodeID, timestamps := range s.whisperPings {
		result[enodeID] = true

		if len(timestamps) < 2 {
			s.log.Debug("Node failed check", "enodeID", enodeID)
			result[enodeID] = false
			continue
		}

		sort.Slice(timestamps, func(i, j int) bool { return timestamps[i] < timestamps[j] })
		timestamps = append(timestamps, uint32(now))
		for i := 1; i < len(timestamps); i++ {

			if timestamps[i]-timestamps[i-1] > pingIntervalAllowance {
				result[enodeID] = false
			}
		}
		if result[enodeID] {
			s.log.Debug("Node passed check", "enodeID", enodeID)
		} else {
			s.log.Debug("Node failed check", "enodeID", enodeID)
		}

	}
	s.log.Debug("voting result", "result", result)
	return result
}

// perform is the main loop, it posts a ping, registers with the contract, check the pings and votes
func (s *Service) perform() error {
	hash, err := s.postPing()
	if err != nil {
		s.log.Error("Could not post ping", "err", err)
		return err
	}
	s.log.Debug("Posted ping", "hash", hash)

	err = s.FetchEnodes()
	if err != nil {
		return err
	}

	err = s.fetchMessages()
	if err != nil {
		return err
	}

	err = s.checkRegistered()
	if err != nil {
		s.log.Error("Could not check if node is registered with the contract", "err", err)
		return err
	}

	// This actually updates the session
	newSession, err := s.ensureSession()
	if err != nil {
		s.log.Error("Could not check session", "err", err)
		return err
	}

	if !newSession {
		s.log.Debug("Not a new session idling")
		return nil
	}

	result := s.checkPings()
	err = s.vote(result)
	if err != nil {
		s.log.Error("Could not vote", "err", err)
		return err
	}

	// Reset whisper pings
	s.whisperPings = make(map[string][]uint32)

	return nil
}

// vote reports to the contract the decisions of the votes
func (s *Service) vote(result map[string]bool) error {
	var behavingNodes []gethcommon.Address
	var misbehavingNodes []gethcommon.Address
	auth := s.auth()

	for enodeIDString, passedCheck := range result {
		enodeID, err := hex.DecodeString(enodeIDString)
		if err != nil {
			return err
		}
		if passedCheck {
			behavingNodes = append(behavingNodes, publicKeyBytesToAddress(enodeID))
		} else {
			misbehavingNodes = append(misbehavingNodes, publicKeyBytesToAddress(enodeID))
		}
	}

	_, err := s.contract.VoteSync(&bind.TransactOpts{
		GasLimit: gasLimit,
		From:     auth.From,
		Signer:   auth.Signer,
	}, behavingNodes, misbehavingNodes)

	return err
}

func (s *Service) startTicker() {
	s.ticker = time.NewTicker(tickerInterval * time.Second)
	s.quit = make(chan struct{})
	go func() {
		for {
			select {
			case <-s.ticker.C:
				err := s.perform()
				if err != nil {
					s.log.Error("could not execute tick", "err", err)
				}
			case <-s.quit:
				s.ticker.Stop()
				return
			}
		}
	}()
}

func (s *Service) Start(server *p2p.Server) error {
	s.log.Info("Incentivisation service started", "address", s.addressString(), "publickey", s.publicKeyString())
	s.startTicker()

	session, err := s.GetCurrentSession()
	if err != nil {
		return err
	}
	s.initialSession = session
	s.currentSession = session

	whisperKeyID, err := s.w.AddPrivateKey(context.TODO(), crypto.FromECDSA(s.privateKey))
	if err != nil {
		return err
	}

	s.whisperKeyID = whisperKeyID

	whisperSymKeyID, err := s.w.GenerateSymKeyFromPassword(context.TODO(), defaultTopic)

	if err != nil {
		return err
	}
	s.whisperSymKeyID = whisperSymKeyID

	criteria := whisper.Criteria{
		SymKeyID: whisperSymKeyID,
		Topics:   []whisper.TopicType{toWhisperTopic(defaultTopic)},
	}
	filterID, err := s.w.NewMessageFilter(criteria)
	if err != nil {
		s.log.Error("could not create filter", "err", err)
		return err
	}
	s.whisperFilterID = filterID

	return nil
}

// Stop is run when a service is stopped.
func (s *Service) Stop() error {
	s.log.Info("Incentivisation service stopped")
	_, err := s.w.DeleteKeyPair(context.TODO(), s.whisperKeyID)
	return err
}

func (s *Service) publicKeyBytes() []byte {
	return crypto.FromECDSAPub(&s.privateKey.PublicKey)[1:]
}

func (s *Service) GetCurrentSession() (uint64, error) {
	response, err := s.contract.GetCurrentSession(nil)
	if err != nil {
		s.log.Error("failed to get current session", "err", err)
		return 0, err
	}
	return response.Uint64(), nil
}

func (s *Service) registered() (bool, error) {
	response, err := s.contract.Registered(nil, s.publicKeyBytes())
	if err != nil {
		return false, err
	}
	return response, nil
}

func (s *Service) register() (bool, error) {
	auth := s.auth()
	ip, err := ip2Long(s.config.IP)
	if err != nil {
		return false, err
	}

	_, err = s.contract.RegisterNode(&bind.TransactOpts{
		GasLimit: gasLimit,
		From:     auth.From,
		Signer:   auth.Signer,
	}, s.publicKeyBytes(), ip, s.config.Port)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) FetchEnodes() error {
	one := big.NewInt(1)

	activeNodeCount, err := s.contract.ActiveNodeCount(nil)
	if err != nil {
		return err
	}
	s.log.Debug("fetched active node count", "count", activeNodeCount)
	for i := big.NewInt(0); i.Cmp(activeNodeCount) < 0; i.Add(i, one) {
		publicKey, ip, port, joiningSession, activeSession, err := s.contract.GetNode(nil, i)
		if err != nil {
			return err
		}

		node := &Enode{
			PublicKey:      publicKey,
			IP:             int2ip(ip),
			Port:           port,
			JoiningSession: joiningSession,
			ActiveSession:  activeSession,
		}

		s.log.Debug("adding node", "node", node.toEnodeURL())
		if node.PublicKeyString() != s.publicKeyString() {
			s.nodes[node.PublicKeyString()] = node
		}
	}

	inactiveNodeCount, err := s.contract.InactiveNodeCount(nil)
	if err != nil {
		return err
	}
	s.log.Debug("fetched inactive node count", "count", inactiveNodeCount)
	for i := big.NewInt(0); i.Cmp(inactiveNodeCount) < 0; i.Add(i, one) {
		publicKey, ip, port, joiningSession, activeSession, err := s.contract.GetInactiveNode(nil, i)
		if err != nil {
			return err
		}

		node := &Enode{
			PublicKey:      publicKey,
			IP:             int2ip(ip),
			Port:           port,
			JoiningSession: joiningSession,
			ActiveSession:  activeSession,
		}

		s.log.Debug("adding node", "node", node.toEnodeURL())
		if node.PublicKeyString() != s.publicKeyString() {
			s.nodes[node.PublicKeyString()] = node
		}
	}

	return nil

}

func (s *Service) publicKeyString() string {
	return hex.EncodeToString(s.publicKeyBytes())
}

func (s *Service) addressString() string {
	buf := crypto.Keccak256Hash(s.publicKeyBytes())
	address := buf[12:]

	return hex.EncodeToString(address)
}

// postPing publishes a whisper message
func (s *Service) postPing() (hexutil.Bytes, error) {
	msg := defaultWhisperMessage()

	msg.Topic = toWhisperTopic(defaultTopic)

	enodeURL := formatEnodeURL(s.publicKeyString(), s.config.IP, s.config.Port)
	payload, err := EncodeMessage(enodeURL, defaultTopic)
	if err != nil {
		return nil, err
	}

	msg.Payload = payload
	msg.Sig = s.whisperKeyID
	msg.SymKeyID = s.whisperSymKeyID

	return s.w.Post(context.TODO(), msg)
}

// fetchMessages checks for whisper messages
func (s *Service) fetchMessages() error {
	messages, err := s.w.GetFilterMessages(s.whisperFilterID)
	if err != nil {
		return err
	}

	for i := 0; i < len(messages); i++ {
		signature := hex.EncodeToString(messages[i].Sig[1:])
		timestamp := messages[i].Timestamp
		if s.nodes[signature] != nil {
			s.whisperPings[signature] = append(s.whisperPings[signature], timestamp)
		}
	}
	return nil
}

func (s *Service) auth() *bind.TransactOpts {
	return bind.NewKeyedTransactor(s.privateKey)
}

func ip2Long(ip string) (uint32, error) {
	var long uint32
	err := binary.Read(bytes.NewBuffer(net.ParseIP(ip).To4()), binary.BigEndian, &long)
	if err != nil {
		return 0, err
	}
	return long, nil
}

func toWhisperTopic(s string) whisper.TopicType {
	return whisper.BytesToTopic(crypto.Keccak256([]byte(s)))
}

func defaultWhisperMessage() whisper.NewMessage {
	msg := whisper.NewMessage{}

	msg.TTL = 10
	msg.PowTarget = 0.002
	msg.PowTime = 1

	return msg
}

func int2ip(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}

func publicKeyBytesToAddress(publicKey []byte) gethcommon.Address {
	buf := crypto.Keccak256Hash(publicKey)
	address := buf[12:]

	return gethcommon.HexToAddress(hex.EncodeToString(address))
}
