package requests

import (
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/pkg/backend"

	"go.uber.org/zap"
	"gopkg.in/go-playground/validator.v9"
)

// AppStateChange represents a request to change the app state from mobile
type AppStateChange struct {
	State backend.AppState `json:"state" validate:"required,app_state"`
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
	state := backend.AppState(fl.Field().String())
	switch state {
	case backend.AppStateBackground, backend.AppStateForeground, backend.AppStateInactive:
		return true
	default:
		return false
	}
}

// Validate checks if the request is valid
func (r *AppStateChange) Validate() error {
	return validate.Struct(r)
}
