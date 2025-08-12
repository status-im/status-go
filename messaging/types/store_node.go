package types

import (
	"encoding/json"
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/ethereum/go-ethereum/p2p/enode"

	wakutypes "github.com/status-im/status-go/waku/types"
)

type StoreNode struct {
	ID     string               `json:"id"`
	Name   string               `json:"name"`
	Custom bool                 `json:"custom"`
	ENR    *enode.Node          `json:"enr"`
	Addr   *multiaddr.Multiaddr `json:"addr"`
}

func (m StoreNode) PeerID() (peer.ID, error) {
	return wakutypes.Mailserver{
		ID:     m.ID,
		Name:   m.Name,
		Custom: m.Custom,
		ENR:    m.ENR,
		Addr:   m.Addr,
	}.PeerID()
}

func (m StoreNode) PeerInfo() (peer.AddrInfo, error) {
	return wakutypes.Mailserver{
		ID:     m.ID,
		Name:   m.Name,
		Custom: m.Custom,
		ENR:    m.ENR,
		Addr:   m.Addr,
	}.PeerInfo()
}

// UnmarshalJSON implements the custom JSON unmarshaling logic for Mailserver.
// It supports ENR and Addr being saved as strings.
func (m *StoreNode) UnmarshalJSON(data []byte) error {
	type Alias StoreNode // Create an alias type to avoid infinite recursion
	aux := struct {
		Alias
		ENR  string `json:"enr"`  // Temporary field to handle ENR as a string
		Addr string `json:"addr"` // Temporary field to handle Addr as a string
	}{}

	// Unmarshal the data into the temporary struct
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Set the basic fields
	*m = StoreNode(aux.Alias)

	// Decode the ENR if present
	if aux.ENR != "" {
		decodedENR, err := enode.Parse(enode.ValidSchemes, aux.ENR)
		if err != nil {
			return fmt.Errorf("invalid ENR: %w", err)
		}
		m.ENR = decodedENR
	}

	// Decode the Multiaddr if present
	if aux.Addr != "" {
		decodedAddr, err := multiaddr.NewMultiaddr(aux.Addr)
		if err != nil {
			return fmt.Errorf("invalid Addr: %w", err)
		}
		m.Addr = &decodedAddr
	}

	return nil
}
