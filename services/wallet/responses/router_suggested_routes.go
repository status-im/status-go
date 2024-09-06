package responses

import (
	"github.com/status-im/status-go/errors"
	"github.com/status-im/status-go/services/wallet/router/routes"
)

type RouterSuggestedRoutes struct {
	Uuid          string                `json:"Uuid"`
	Best          routes.Route          `json:"Best,omitempty"`
	Candidates    routes.Route          `json:"Candidates,omitempty"`
	UpdatedPrices map[string]float64    `json:"UpdatedPrices,omitempty"`
	ErrorResponse *errors.ErrorResponse `json:"ErrorResponse,omitempty"`
}
