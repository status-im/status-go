package onramp_tests

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/services/wallet/onramp"
	mock_token "github.com/status-im/status-go/services/wallet/token/mock/token"

	"github.com/stretchr/testify/require"

	"go.uber.org/mock/gomock"
)

func TestMercuryo(t *testing.T) {
	stageName := os.Getenv("STATUS_BUILD_PROXY_STAGE_NAME")
	params := onramp.MercuryoParams{
		SignURL:      fmt.Sprintf("https://%s.api.status.im/mercuryo/sign/", stageName),
		SignUser:     security.NewSensitiveString(os.Getenv("STATUS_BUILD_PROXY_USER")),
		SignPassword: security.NewSensitiveString(os.Getenv("STATUS_BUILD_PROXY_PASSWORD")),
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tokenManagerMock := mock_token.NewMockManagerInterface(ctrl)

	provider := onramp.NewMercuryoProvider(tokenManagerMock, params)

	isRecurrent := false
	chainID := uint64(1)
	symbol := "ETH"
	destAddress := common.HexToAddress("0x1234567890123456789012345678901234567890")

	url, err := provider.GetURL(context.Background(), onramp.Parameters{
		IsRecurrent: isRecurrent,
		ChainID:     &chainID,
		Symbol:      &symbol,
		DestAddress: &destAddress,
	})

	require.NoError(t, err)
	require.NotEmpty(t, url)
	fmt.Println("url", url)
}
