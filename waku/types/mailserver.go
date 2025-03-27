package types

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/multiformats/go-multiaddr"

	"github.com/waku-org/go-waku/waku/v2/protocol/enr"
	"github.com/waku-org/go-waku/waku/v2/utils"

	"github.com/status-im/status-go/eth-node/types"
)

// MailServerResponse is the response payload sent by the mailserver.
type MailServerResponse struct {
	LastEnvelopeHash types.Hash
	Cursor           []byte
	Error            error
}

// SyncMailRequest contains details which envelopes should be synced
// between Mail Servers.
type SyncMailRequest struct {
	// Lower is a lower bound of time range for which messages are requested.
	Lower uint32
	// Upper is a lower bound of time range for which messages are requested.
	Upper uint32
	// Bloom is a bloom filter to filter envelopes.
	Bloom []byte
	// Limit is the max number of envelopes to return.
	Limit uint32
	// Cursor is used for pagination of the results.
	Cursor []byte
}

// SyncEventResponse is a response from the Mail Server
// form which the peer received envelopes.
type SyncEventResponse struct {
	Cursor []byte
	Error  string
}

func MustDecodeENR(enrStr string) *enode.Node {
	node, err := enode.Parse(enode.ValidSchemes, enrStr)
	if err != nil || node == nil {
		panic("could not decode enr: " + enrStr)
	}
	return node
}

func MustDecodeMultiaddress(multiaddrsStr string) *multiaddr.Multiaddr {
	maddr, err := multiaddr.NewMultiaddr(multiaddrsStr)
	if err != nil || maddr == nil {
		panic("could not decode multiaddr: " + multiaddrsStr)
	}
	return &maddr
}

type Mailserver struct {
	ID     string               `json:"id"`
	Name   string               `json:"name"`
	Custom bool                 `json:"custom"`
	ENR    *enode.Node          `json:"enr"`
	Addr   *multiaddr.Multiaddr `json:"addr"`

	// Deprecated: only used with WakuV1
	Password       string `json:"password,omitempty"`
	Fleet          string `json:"fleet"`
	FailedRequests uint   `json:"-"`
}

func (m Mailserver) PeerInfo() (peer.AddrInfo, error) {
	var maddrs []multiaddr.Multiaddr

	if m.ENR != nil {
		addrInfo, err := enr.EnodeToPeerInfo(m.ENR)
		if err != nil {
			return peer.AddrInfo{}, err
		}
		addrInfo.Addrs = utils.EncapsulatePeerID(addrInfo.ID, addrInfo.Addrs...)
		maddrs = append(maddrs, addrInfo.Addrs...)
	}

	if m.Addr != nil {
		maddrs = append(maddrs, *m.Addr)
	}

	p, err := peer.AddrInfosFromP2pAddrs(maddrs...)
	if err != nil {
		return peer.AddrInfo{}, err
	}

	if len(p) != 1 {
		return peer.AddrInfo{}, errors.New("invalid mailserver setup")
	}

	return p[0], nil
}

func (m Mailserver) PeerID() (peer.ID, error) {
	p, err := m.PeerInfo()
	if err != nil {
		return "", err
	}
	return p.ID, nil
}

func (m Mailserver) NullablePassword() (val sql.NullString) {
	if m.Password != "" {
		val.String = m.Password
		val.Valid = true
	}
	return
}

// UnmarshalJSON implements the custom JSON unmarshaling logic for Mailserver.
// It supports ENR and Addr being saved as strings.
func (m *Mailserver) UnmarshalJSON(data []byte) error {
	type Alias Mailserver // Create an alias type to avoid infinite recursion
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
	*m = Mailserver(aux.Alias)

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
