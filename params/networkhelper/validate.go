package networkhelper

import (
	"gopkg.in/go-playground/validator.v9"

	"github.com/status-im/status-go/params"
)

func GetValidator() *validator.Validate {
	validate := validator.New()

	// Register struct-level validation for RpcProvider
	validate.RegisterStructValidation(rpcProviderStructLevelValidation, params.RpcProvider{})

	return validate
}

func rpcProviderStructLevelValidation(sl validator.StructLevel) {
	provider := sl.Current().Interface().(params.RpcProvider)

	switch provider.AuthType {
	case params.NoAuth:
		if !provider.AuthLogin.Empty() || !provider.AuthPassword.Empty() || !provider.AuthToken.Empty() {
			sl.ReportError(provider.AuthLogin, "AuthLogin", "authLogin", "noauth_fields_empty", "")
			sl.ReportError(provider.AuthPassword, "AuthPassword", "authPassword", "noauth_fields_empty", "")
			sl.ReportError(provider.AuthToken, "AuthToken", "authToken", "noauth_fields_empty", "")
		}
	case params.BasicAuth:
		if provider.AuthLogin.Empty() {
			sl.ReportError(provider.AuthLogin, "AuthLogin", "authLogin", "required", "")
		}
		if provider.AuthPassword.Empty() {
			sl.ReportError(provider.AuthPassword, "AuthPassword", "authPassword", "required", "")
		}
		if !provider.AuthToken.Empty() {
			sl.ReportError(provider.AuthToken, "AuthToken", "authToken", "basic_auth_token_empty", "")
		}
	case params.TokenAuth:
		if provider.AuthToken.Empty() {
			sl.ReportError(provider.AuthToken, "AuthToken", "authToken", "required", "")
		}
		if !provider.AuthLogin.Empty() || !provider.AuthPassword.Empty() {
			sl.ReportError(provider.AuthLogin, "AuthLogin", "authLogin", "tokenauth_fields_empty", "")
			sl.ReportError(provider.AuthPassword, "AuthPassword", "authPassword", "tokenauth_fields_empty", "")
		}
	default:
		sl.ReportError(provider.AuthType, "AuthType", "authType", "invalid_auth_type", "")
	}
}
