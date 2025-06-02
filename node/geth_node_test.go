package node

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/t/utils"
)

func TestMakeNodeDefaultConfig(t *testing.T) {
	utils.Init()
	config, err := utils.MakeTestNodeConfig(3)
	require.NoError(t, err)

	_, err = MakeNode(config, &accounts.Manager{})
	require.NoError(t, err)
}

func TestNewGethNodeConfig(t *testing.T) {
	config, err := params.NewNodeConfig("", params.SepoliaNetworkID)
	require.NoError(t, err)
	config.HTTPEnabled = true
	config.HTTPVirtualHosts = []string{"my.domain.com"}
	config.HTTPCors = []string{"http://my.domain.com"}

	nc, err := newGethNodeConfig(config)
	require.NoError(t, err)
	require.Equal(t, []string{"my.domain.com"}, nc.HTTPVirtualHosts)
	require.Equal(t, []string{"http://my.domain.com"}, nc.HTTPCors)
}
