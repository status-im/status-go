package responses

import (
	"github.com/status-im/status-go/internal/errors"
	"github.com/status-im/status-go/services/wallet/router/routes"
)

type RouterSuggestedRoutes struct {
	Uuid          string                `json:"Uuid"`
	Route         routes.Route          `json:"Route,omitempty"`
	UpdatedPrices map[string]float64    `json:"UpdatedPrices,omitempty"`
	ErrorResponse *errors.ErrorResponse `json:"ErrorResponse,omitempty"`
}
