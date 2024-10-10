package requests

import (
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/logutils"

	"go.uber.org/zap"
	"gopkg.in/go-playground/validator.v9"
)

// AppStateChange represents a request to change the app state from mobile
type AppStateChange struct {
	State api.AppState `json:"state" validate:"required,app_state"`
}

var validate *validator.Validate

func init() {
	validate = validator.New()
	err := validate.RegisterValidation("app_state", validateAppState)
	if err != nil {
		logutils.ZapLogger().Error("register app state validation failed", zap.Error(err))
	}
}

func validateAppState(fl validator.FieldLevel) bool {
	state := api.AppState(fl.Field().String())
	switch state {
	case api.AppStateBackground, api.AppStateForeground, api.AppStateInactive:
		return true
	default:
		return false
	}
}

// Validate checks if the request is valid
func (r *AppStateChange) Validate() error {
	return validate.Struct(r)
}
