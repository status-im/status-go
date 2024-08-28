package responses

import (
	"github.com/status-im/status-go/errors"
	"github.com/status-im/status-go/services/wallet/router/routes"
)

type RouterSuggestedRoutes struct {
	Uuid                  string                `json:"Uuid"`
	Best                  routes.Route          `json:"Best,omitempty"`
	Candidates            routes.Route          `json:"Candidates,omitempty"`
	TokenPrice            *float64              `json:"TokenPrice,omitempty"`
	NativeChainTokenPrice *float64              `json:"NativeChainTokenPrice,omitempty"`
	ErrorResponse         *errors.ErrorResponse `json:"ErrorResponse,omitempty"`
}
