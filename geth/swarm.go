package geth

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/swarm"
	bzzapi "github.com/ethereum/go-ethereum/swarm/api"
)

var (
	ErrInvalidSwarmService           = errors.New("swarm service is unavailable")
	ErrSwarmIdentityInjectionFailure = errors.New("failed to inject identity into Swarm")
)

// Service for managing swarm instance
type SwarmService struct {
	instanceDir  string
	httpEndpoint string
	swarm        *swarm.Swarm
	privateKey   *ecdsa.PrivateKey
}

func newSwarmService(stack *node.Node) (*SwarmService, error) {
	service := &SwarmService{
		instanceDir:  filepath.Join(stack.InstanceDir(), "swarmdata"),
		httpEndpoint: fmt.Sprintf("http://%s", stack.HTTPEndpoint()),
	}

	var err error
	service.swarm, err = service.newSwarmNode(nil)

	return service, err
}

// activateSwarmService configures and registers the SwarmService with a given node.
func (s *SwarmService) newSwarmNode(prvkey *ecdsa.PrivateKey) (*swarm.Swarm, error) {
	var err error
	if prvkey == nil {
		// TODO rely on truly random key (via crypto.GenerateKey())
		// atm, Swarm has issues with dynamically switching underlying account
		prvkey, err = ecdsa.GenerateKey(secp256k1.S256(), strings.NewReader("fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"))
		if err != nil {
			return nil, err
		}
	}

	chbookaddr := common.Address{}

	networkId := uint64(1)
	if UseTestnet {
		networkId = 3
	}

	bzzconfig, err := bzzapi.NewConfig(s.instanceDir, chbookaddr, prvkey, networkId)
	if err != nil {
		return nil, fmt.Errorf("%v: %v", err, err)
	}

	var client *ethclient.Client
	client, err = ethclient.Dial(s.httpEndpoint)
	if err != nil {
		return nil, fmt.Errorf("Can't connect: %v", err)
	}

	return swarm.NewSwarm(nil, client, bzzconfig, false, true, "")
}

// Run swarm node on with specified private key
// TODO implement dynamic account selection (when Swarm is stable enough for this)
func (s *SwarmService) RunSwarmNode(prvkey *ecdsa.PrivateKey) error {
	return nil

	nodeManager := NodeManagerInstance()
	p2p := nodeManager.node.geth.Server()

	// stop old swarm instance
	if err := s.StopSwarmNode(); err != nil {
		return err
	}

	sw, err := s.newSwarmNode(prvkey)
	if err != nil {
		return err
	}

	*s.swarm = *sw
	return s.swarm.Start(p2p)
}

func (s *SwarmService) RestartSwarmNode() error {
	nodeManager := NodeManagerInstance()

	if err := s.StopSwarmNode(); err != nil {
		return err
	}

	return s.swarm.Start(nodeManager.node.geth.Server())
}

// Stop current running swarm node
func (s *SwarmService) StopSwarmNode() error {
	if err := s.swarm.Stop(); err != nil {
		return err
	}

	return nil
}
