package onramp

import (
	"context"
	"fmt"

	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

const rampID = "ramp"
const rampSiteURL = "https://ramp.network/buy?hostApiKey=zrtf9u2uqebeyzcs37fu5857tktr3eg9w5tffove&swapAsset=DAI,ETH,USDC,USDT"

type RampProvider struct{}

func NewRampProvider() *RampProvider {
	return &RampProvider{}
}

func (p *RampProvider) ID() string {
	return rampID
}

func (p *RampProvider) GetCryptoOnRamp(ctx context.Context) (CryptoOnRamp, error) {
	const (
		logoRamp = "data:image/png;base64,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"
	)

	onramp := CryptoOnRamp{
		ID:                        rampID,
		Name:                      "Ramp",
		Description:               "Global crypto to fiat flow",
		Fees:                      "0.49% - 2.9%",
		LogoURL:                   logoRamp,
		Hostname:                  "ramp.network",
		SupportsSinglePurchase:    true,
		SupportsRecurrentPurchase: false,
		SupportedChainIDs:         []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet, walletCommon.OptimismMainnet},
		URLsNeedParameters:        false,
		SiteURL:                   rampSiteURL,
	}

	return onramp, nil
}

func (p *RampProvider) GetURL(ctx context.Context, parameters Parameters) (string, error) {
	if !parameters.IsRecurrent {
		return rampSiteURL, nil
	}
	return "", fmt.Errorf("recurrent transactions are not supported by Ramp")
}
