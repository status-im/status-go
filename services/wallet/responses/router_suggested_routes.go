package responses

import (
	"github.com/status-im/status-go/v10/errors"
	"github.com/status-im/status-go/v10/services/wallet/router/routes"
)

type RouterSuggestedRoutes struct {
	Uuid          string                `json:"Uuid"`
	Best          routes.Route          `json:"Best,omitempty"`
	Candidates    routes.Route          `json:"Candidates,omitempty"`
	UpdatedPrices map[string]float64    `json:"UpdatedPrices,omitempty"`
	ErrorResponse *errors.ErrorResponse `json:"ErrorResponse,omitempty"`
}
