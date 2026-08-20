package onramp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/status-im/status-go/pkg/security"

	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/mercuryo"
	"github.com/status-im/status-go/pkg/services/wallet/token"
)

const mercuryoID = "mercuryo"
const mercuryioNoFeesBaseURL = "https://exchange.mercuryo.io/?type=buy&networks=ETHEREUM,ARBITRUM,OPTIMISM,BASE&currency=ETH"
const supportedAssetsUpdateInterval = 24 * time.Hour

type MercuryoProvider struct {
	supportedTokens          []*types.Token
	supportedTokensTimestamp time.Time
	supportedTokensLock      sync.RWMutex
	httpClient               *mercuryo.Client
	tokenManager             token.ManagerInterface
	params                   MercuryoParams
}

type MercuryoParams struct {
	SignURL      string
	SignUser     security.SensitiveString
	SignPassword security.SensitiveString
}

func NewMercuryoProvider(tokenManager token.ManagerInterface, params MercuryoParams) *MercuryoProvider {
	return &MercuryoProvider{
		httpClient:   mercuryo.NewClient(),
		tokenManager: tokenManager,
		params:       params,
	}
}

// Returns a widget-id and a signature (the SHA512 hash of the concatenated address and the widget's secret).
func (p *MercuryoProvider) getMercuryoSignatureParams(ctx context.Context, address common.Address) (widgetID string, signature string, err error) {
	if p.params.SignURL == "" || p.params.SignUser.Empty() || p.params.SignPassword.Empty() {
		return "", "", errors.New("signURL, signUser, and signPassword are required")
	}

	queryParams := url.Values{
		"address": {address.Hex()},
	}

	client := thirdparty.NewHTTPClient()
	resp, err := client.DoGetRequestWithCredentials(ctx, p.params.SignURL, queryParams, &thirdparty.BasicCreds{
		User:     p.params.SignUser,
		Password: p.params.SignPassword,
	})
	if err != nil {
		return "", "", err
	}

	type signResponse struct {
		WidgetID  string `json:"widget_id"`
		Signature string `json:"signature"`
	}
	var response signResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return "", "", err
	}

	return response.WidgetID, response.Signature, nil
}

func getMercuryoCurrency(symbol string) string {
	return strings.ToUpper(symbol)
}

func (p *MercuryoProvider) GetURL(ctx context.Context, parameters Parameters) (string, error) {
	const baseURL = "https://exchange.mercuryo.io/?type=buy"

	if parameters.DestAddress == nil || *parameters.DestAddress == walletCommon.ZeroAddress() {
		return "", errors.New("destination address is required")
	}

	if parameters.ChainID == nil || *parameters.ChainID == walletCommon.UnknownChainID {
		return "", errors.New("chainID is required")
	}

	if parameters.Symbol == nil || *parameters.Symbol == "" {
		return "", errors.New("symbol is required")
	}

	network := mercuryo.CommonChainIDToNetwork(*parameters.ChainID)
	if network == "" {
		return "", errors.New("unsupported chainID")
	}

	currency := getMercuryoCurrency(*parameters.Symbol)
	if currency == "" {
		return "", errors.New("unsupported symbol")
	}

	widgetID, signature, err := p.getMercuryoSignatureParams(ctx, *parameters.DestAddress)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s&network=%s&currency=%s&address=%s&hide_address=false&fix_address=true&signature=%s&widget_id=%s",
		baseURL, network, currency, parameters.DestAddress.Hex(), signature, widgetID)

	if parameters.IsRecurrent {
		url = url + "&widget_flow=recurrent"
	}

	return url, nil
}
