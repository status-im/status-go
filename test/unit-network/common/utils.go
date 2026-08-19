package common

import (
	"os"

	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/protocol/requests"
)

func GetWalletSecretsConfigFromEnv() *requests.WalletSecretsConfig {
	return &requests.WalletSecretsConfig{
		StatusProxyStageName: os.Getenv("STATUS_BUILD_PROXY_STAGE_NAME"),
		EthRpcProxyUser:      security.NewSensitiveString(os.Getenv("STATUS_BUILD_ETH_RPC_PROXY_USER")),
		EthRpcProxyPassword:  security.NewSensitiveString(os.Getenv("STATUS_BUILD_ETH_RPC_PROXY_PASSWORD")),
		EthRpcProxyUrl:       security.NewSensitiveString(os.Getenv("STATUS_BUILD_ETH_RPC_PROXY_URL")),
	}
}
