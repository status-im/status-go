package enr

import (
	"fmt"
	"net"
	"testing"

	gcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestEnodeToMultiAddr(t *testing.T) {
	enr := "enr:-QESuEC1p_s3xJzAC_XlOuuNrhVUETmfhbm1wxRGis0f7DlqGSw2FM-p2Ugl_r25UHQJ3f1rIRrpzxJXSMaJe4yk1XFSAYJpZIJ2NIJpcISygI2rim11bHRpYWRkcnO4XAArNiZub2RlLTAxLmRvLWFtczMud2FrdS50ZXN0LnN0YXR1c2ltLm5ldAZ2XwAtNiZub2RlLTAxLmRvLWFtczMud2FrdS50ZXN0LnN0YXR1c2ltLm5ldAYfQN4DgnJzkwABCAAAAAEAAgADAAQABQAGAAeJc2VjcDI1NmsxoQJATXRSRSUyTw_QLB6H_U3oziVQgNRgrXpK7wp2AMyNxYN0Y3CCdl-DdWRwgiMohXdha3UyDw"

	parsedNode := enode.MustParse(enr)
	expectedMultiAddr := "/ip4/178.128.141.171/tcp/30303/p2p/16Uiu2HAkykgaECHswi3YKJ5dMLbq2kPVCo89fcyTd38UcQD6ej5W"
	actualMultiAddr, err := enodeToMultiAddr(parsedNode)
	require.NoError(t, err)
	require.Equal(t, expectedMultiAddr, actualMultiAddr.String())
}

func TestMultiaddr(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	key, _ := gcrypto.GenerateKey()
	wakuFlag := NewWakuEnrBitfield(true, true, true, true)

	wss, _ := ma.NewMultiaddr("/dns4/www.somedomainname.com/tcp/443/wss")
	circuit1, _ := ma.NewMultiaddr("/dns4/node-02.gc-us-central1-a.status.prod.status.im/tcp/30303/p2p/16Uiu2HAmDQugwDHM3YeUp86iGjrUvbdw3JPRgikC7YoGBsT2ymMg/p2p-circuit")
	circuit2, _ := ma.NewMultiaddr("/dns4/node-01.gc-us-central1-a.status.prod.status.im/tcp/30303/p2p/16Uiu2HAmDQugwDHM3YeUp86iGjrUvbdw3JPRgikC7YoGBsT2ymMg/p2p-circuit")
	circuit3, _ := ma.NewMultiaddr("/dns4/node-03.gc-us-central1-a.status.prod.status.im/tcp/30303/p2p/16Uiu2HAmDQugwDHM3YeUp86iGjrUvbdw3JPRgikC7YoGBsT2ymMg/p2p-circuit")
	circuit4, _ := ma.NewMultiaddr("/dns4/node-03.gc-us-central1-a.status.prod.status.im/tcp/30303/p2p/16Uiu2HAmDQugwDHM3YeUp86iGjrUvbdw3JPRgikC7YoGBsT2ymMg/p2p-circuit")
	circuit5, _ := ma.NewMultiaddr("/dns4/node-03.gc-us-central1-a.status.prod.status.im/tcp/30303/p2p/16Uiu2HAmDQugwDHM3YeUp86iGjrUvbdw3JPRgikC7YoGBsT2ymMg/p2p-circuit")
	circuit6, _ := ma.NewMultiaddr("/dns4/node-03.gc-us-central1-a.status.prod.status.im/tcp/30303/p2p/16Uiu2HAmDQugwDHM3YeUp86iGjrUvbdw3JPRgikC7YoGBsT2ymMg/p2p-circuit")

	multiaddrValues := []ma.Multiaddr{
		wss,
		circuit1,
		circuit2,
		circuit3,
		circuit4,
		circuit5,
		circuit6,
	}

	db, _ := enode.OpenDB("")
	localNode := enode.NewLocalNode(db, key)
	err = UpdateLocalNode(logger, localNode, &LocalNodeParams{
		Multiaddrs:       multiaddrValues,
		IPAddr:           &net.TCPAddr{IP: net.IPv4(192, 168, 1, 241), Port: 60000},
		UDPPort:          50000,
		WakuFlags:        wakuFlag,
		AdvertiseAddr:    nil,
		ShouldAutoUpdate: false,
	})
	require.NoError(t, err)

	require.NotPanics(t, func() {
		_ = localNode.Node()
	})

	peerID, maddrs, err := Multiaddress(localNode.Node())
	require.NoError(t, err)

	fmt.Println("peerID: ", peerID)
	fmt.Println("len maddrs: ", len(maddrs))
	fmt.Println("maddrs: ", maddrs)
}
