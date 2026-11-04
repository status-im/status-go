package library

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
)

type dnsDiscoveryItem struct {
	PeerID    string   `json:"peerID"`
	Addresses []string `json:"multiaddrs"`
	ENR       string   `json:"enr,omitempty"`
}

// DNSDiscovery executes dns discovery on an url and returns a list of nodes
func DNSDiscovery(url string, nameserver string, ms int) (string, error) {
	var ctx context.Context
	var cancel context.CancelFunc

	if ms > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	var dnsDiscOpt []dnsdisc.DNSDiscoveryOption
	if nameserver != "" {
		dnsDiscOpt = append(dnsDiscOpt, dnsdisc.WithNameserver(nameserver))
	}

	nodes, err := dnsdisc.RetrieveNodes(ctx, url, dnsDiscOpt...)
	if err != nil {
		return "", err
	}

	var response []dnsDiscoveryItem
	for _, n := range nodes {
		item := dnsDiscoveryItem{
			PeerID: n.PeerID.String(),
		}
		for _, addr := range n.PeerInfo.Addrs {
			item.Addresses = append(item.Addresses, addr.String())
		}

		if n.ENR != nil {
			item.ENR = n.ENR.String()
		}

		response = append(response, item)
	}

	return marshalJSON(response)
}

// StartDiscoveryV5 starts discv5 discovery
func StartDiscoveryV5(instance *WakuInstance) error {
	if err := validateInstance(instance, MustBeStarted); err != nil {
		return err
	}

	if instance.node.DiscV5() == nil {
		return errors.New("DiscV5 is not mounted")
	}
	return instance.node.DiscV5().Start(instance.ctx)
}

// StopDiscoveryV5 stops discv5 discovery
func StopDiscoveryV5(instance *WakuInstance) error {
	if err := validateInstance(instance, MustBeStarted); err != nil {
		return err
	}

	if instance.node.DiscV5() == nil {
		return errors.New("DiscV5 is not mounted")
	}
	instance.node.DiscV5().Stop()
	return nil
}

// SetBootnodes is used to update the bootnodes receiving a JSON array of ENRs
func SetBootnodes(instance *WakuInstance, bootnodes string) error {
	if err := validateInstance(instance, MustBeStarted); err != nil {
		return err
	}

	if instance.node.DiscV5() == nil {
		return errors.New("DiscV5 is not mounted")
	}

	var tmp []json.RawMessage
	if err := json.Unmarshal([]byte(bootnodes), &tmp); err != nil {
		return err
	}

	var enrList []string
	for _, el := range tmp {
		var enr string
		if err := json.Unmarshal(el, &enr); err != nil {
			return err
		}
		enrList = append(enrList, enr)
	}

	var nodes []*enode.Node
	for _, addr := range enrList {
		node, err := enode.Parse(enode.ValidSchemes, addr)
		if err != nil {
			return err
		}
		nodes = append(nodes, node)
	}

	return instance.node.DiscV5().SetBootnodes(nodes)
}
