package onramp

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/services/wallet/token"
)

type Provider interface {
	ID() string
	GetCryptoOnRamp(ctx context.Context) (CryptoOnRamp, error)
	GetURL(ctx context.Context, parameters Parameters) (string, error)
}

type Parameters struct {
	IsRecurrent bool            `json:"isRecurrent"`
	DestAddress *common.Address `json:"destAddress,omitempty"`
	ChainID     *uint64         `json:"chainID,omitempty"`
	Symbol      *string         `json:"symbol,omitempty"`
}

type CryptoOnRamp struct {
	ID                        string         `json:"id"`
	Name                      string         `json:"name"`
	Description               string         `json:"description"`
	Fees                      string         `json:"fees"`
	LogoURL                   string         `json:"logoUrl"`
	Hostname                  string         `json:"hostname"`
	SupportsSinglePurchase    bool           `json:"supportsSinglePurchase"`
	SupportsRecurrentPurchase bool           `json:"supportsRecurrentPurchase"`
	SupportedChainIDs         []uint64       `json:"supportedChainIds"`
	SupportedTokens           []*token.Token `json:"supportedTokens"`    // Empty array means supported assets are not specified
	URLsNeedParameters        bool           `json:"urlsNeedParameters"` // True means Parameters are required for URL generation
	// Deprecated fields below, only used by mobile
	Params           map[string]string `json:"params"`
	SiteURL          string            `json:"siteUrl"`          // Replaced by call to GetURL
	RecurrentSiteURL string            `json:"recurrentSiteUrl"` // Replaced by call to GetURL
}
