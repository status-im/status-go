package messaging

import (
	"encoding/hex"
	"fmt"
	"net"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"

	bindings "github.com/logos-messaging/logos-delivery-go-bindings/pkg/messaging"

	"github.com/status-im/status-go/pkg/messaging/delivery"
	"github.com/status-im/status-go/pkg/messaging/waku"
)

// deliveryConfig selects the logos-delivery preset named after the fleet: a
// built-in one, or one registered from a fleets file by
// delivery.RegisterFleetPresets.
func deliveryConfig(params CoreParams) (bindings.Config, error) {
	if params.Fleet == "" {
		return bindings.Config{}, fmt.Errorf("no fleet to select a logos-delivery preset by")
	}
	preset := params.Fleet
	mode := bindings.ModeCore
	if params.Mode == waku.ModeEdge {
		mode = bindings.ModeEdge
	}
	overrides := bindings.Overrides{
		"reliability":                 false,
		"backfill-enabled":            false,
		"configured-store-nodes-only": true,
	}
	if params.NodeKey != nil {
		overrides["nodekey"] = hex.EncodeToString(crypto.FromECDSA(params.NodeKey))
	}
	if params.WakuConfig.Port != 0 {
		overrides["tcp-port"] = params.WakuConfig.Port
	}
	if params.WakuConfig.UDPPort != 0 {
		overrides["discv5-udp-port"] = params.WakuConfig.UDPPort
		// QUIC listens on the UDP port of the TCP one.
		if params.WakuConfig.UDPPort == params.WakuConfig.Port {
			overrides["quic-support"] = false
		}
	}
	if params.DataDir != "" {
		overrides["local-storage-path"] = params.DataDir
	}
	if os.Getenv(advertiseInterfaceIPEnv) == "1" {
		ip, err := interfaceIPv4()
		if err != nil {
			return bindings.Config{}, err
		}
		overrides["nat"] = "extip:" + ip.String()
	}
	return bindings.Config{Mode: mode, Preset: preset, MessagingOverrides: overrides}, nil
}

// advertiseInterfaceIPEnv, set to 1, makes the node advertise its interface
// address, as go-waku does. logos-delivery advertises only an address that is
// configured or confirmed from outside, which a node on a private network,
// such as the functional tests, never gets.
const advertiseInterfaceIPEnv = "STATUS_DELIVERY_ADVERTISE_INTERFACE_IP"

// interfaceIPv4 returns the first non-loopback IPv4 address of the host.
func interfaceIPv4() (net.IP, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("list the interface addresses: %w", err)
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		if ip := ipNet.IP.To4(); ip != nil {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("no non-loopback IPv4 interface address to advertise")
}

func newDeliveryBackend(params CoreParams, logger *zap.Logger) (*delivery.Adapter, error) {
	cfg, err := deliveryConfig(params)
	if err != nil {
		return nil, err
	}
	return delivery.NewAdapter(cfg, logger)
}
