package router

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"

	"github.com/stretchr/testify/assert"
)

func TestValidateInputData(t *testing.T) {
	testCases := []struct {
		name          string
		input         *RouteInputParams
		expectedError error
	}{
		{
			name: "ENSRegister valid data on testnet",
			input: &RouteInputParams{
				SendType:    ENSRegister,
				Username:    "validusername.eth",
				PublicKey:   "validpublickey",
				TokenID:     pathprocessor.SttSymbol,
				TestnetMode: true,
			},
			expectedError: nil,
		},
		{
			name: "ENSRegister valid data on mainnet",
			input: &RouteInputParams{
				SendType:  ENSRegister,
				Username:  "validusername.eth",
				PublicKey: "validpublickey",
				TokenID:   pathprocessor.SntSymbol,
			},
			expectedError: nil,
		},
		{
			name: "ENSRegister missing username",
			input: &RouteInputParams{
				SendType:    ENSRegister,
				PublicKey:   "validpublickey",
				TokenID:     pathprocessor.SttSymbol,
				TestnetMode: true,
			},
			expectedError: ErrorENSRegisterRequires,
		},
		{
			name: "ENSRegister missing public key",
			input: &RouteInputParams{
				SendType:    ENSRegister,
				Username:    "validusername.eth",
				TokenID:     pathprocessor.SttSymbol,
				TestnetMode: true,
			},
			expectedError: ErrorENSRegisterRequires,
		},
		{
			name: "ENSRegister invalid token on testnet",
			input: &RouteInputParams{
				SendType:    ENSRegister,
				Username:    "validusername.eth",
				PublicKey:   "validpublickey",
				TokenID:     "invalidtoken",
				TestnetMode: true,
			},
			expectedError: ErrorENSRegisterTestNetSTTOnly,
		},
		{
			name: "ENSRegister invalid token on mainnet",
			input: &RouteInputParams{
				SendType:  ENSRegister,
				Username:  "validusername.eth",
				PublicKey: "validpublickey",
				TokenID:   "invalidtoken",
			},
			expectedError: ErrorENSRegisterSNTOnly,
		},
		{
			name: "ENSRelease valid data",
			input: &RouteInputParams{
				SendType: ENSRelease,
				Username: "validusername.eth",
			},
			expectedError: nil,
		},
		{
			name: "ENSRelease missing username",
			input: &RouteInputParams{
				SendType: ENSRelease,
			},
			expectedError: ErrorENSReleaseRequires,
		},
		{
			name: "ENSSetPubKey valid data",
			input: &RouteInputParams{
				SendType:  ENSSetPubKey,
				Username:  "validusername.eth",
				PublicKey: "validpublickey",
			},
			expectedError: nil,
		},
		{
			name: "ENSSetPubKey missing username",
			input: &RouteInputParams{
				SendType:  ENSSetPubKey,
				PublicKey: "validpublickey",
			},
			expectedError: ErrorENSSetPubKeyRequires,
		},
		{
			name: "ENSSetPubKey missing public key",
			input: &RouteInputParams{
				SendType: ENSSetPubKey,
				Username: "validusername",
			},
			expectedError: ErrorENSSetPubKeyRequires,
		},
		{
			name: "ENSSetPubKey invalid ENS username",
			input: &RouteInputParams{
				SendType:  ENSSetPubKey,
				Username:  "invalidusername",
				PublicKey: "validpublickey",
			},
			expectedError: ErrorENSSetPubKeyRequires,
		},
		{
			name: "fromLockedAmount with supported network on testnet",
			input: &RouteInputParams{
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumSepolia: (*hexutil.Big)(big.NewInt(10)),
				},
				TestnetMode: true,
			},
			expectedError: nil,
		},
		{
			name: "fromLockedAmount with supported network on mainnet",
			input: &RouteInputParams{
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(10)),
				},
			},
			expectedError: nil,
		},
		{
			name: "fromLockedAmount with supported mainnet network while in test mode",
			input: &RouteInputParams{
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(10)),
				},
				TestnetMode: true,
			},
			expectedError: errors.New("locked amount is not supported for the selected network"),
		},
		{
			name: "fromLockedAmount with unsupported network on testnet",
			input: &RouteInputParams{
				FromLockedAmount: map[uint64]*hexutil.Big{
					999: (*hexutil.Big)(big.NewInt(10)),
				},
				TestnetMode: true,
			},
			expectedError: errors.New("locked amount is not supported for the selected network"),
		},
		{
			name: "fromLockedAmount with unsupported network on mainnet",
			input: &RouteInputParams{
				FromLockedAmount: map[uint64]*hexutil.Big{
					999: (*hexutil.Big)(big.NewInt(10)),
				},
			},
			expectedError: ErrorLockedAmountNotSupportedNetwork,
		},
		{
			name: "fromLockedAmount with negative amount",
			input: &RouteInputParams{
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(-10)),
				},
			},
			expectedError: ErrorLockedAmountNotNegative,
		},
		{
			name: "fromLockedAmount with zero amount",
			input: &RouteInputParams{
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(0)),
				},
			},
			expectedError: nil,
		},
		{
			name: "fromLockedAmount with zero amounts",
			input: &RouteInputParams{
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(0)),
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(0)),
				},
			},
			expectedError: nil,
		},
		{
			name: "fromLockedAmount with all supported networks with zero amount",
			input: &RouteInputParams{
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(0)),
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(0)),
					walletCommon.ArbitrumMainnet: (*hexutil.Big)(big.NewInt(0)),
				},
			},
			expectedError: ErrorLockedAmountExcludesAllSupported,
		},
		{
			name: "fromLockedAmount with all supported test networks with zero amount",
			input: &RouteInputParams{
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumSepolia: (*hexutil.Big)(big.NewInt(0)),
					walletCommon.OptimismSepolia: (*hexutil.Big)(big.NewInt(0)),
					walletCommon.ArbitrumSepolia: (*hexutil.Big)(big.NewInt(0)),
				},
				TestnetMode: true,
			},
			expectedError: ErrorLockedAmountExcludesAllSupported,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateInputData(tc.input)
			if tc.expectedError == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError.Error())
			}
		})
	}
}
