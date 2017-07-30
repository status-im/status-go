package params

import validator "gopkg.in/go-playground/validator.v9"

// NewValidator returns a new Validate
// with custom validation functions.
func NewValidator() *validator.Validate {
	validate := validator.New()

	validate.RegisterValidation("network", networkValidator)

	return validate
}

func networkValidator(fl validator.FieldLevel) bool {
	id := int(fl.Field().Uint())
	val, ok := NetworkIDs[id]
	if !ok {
		return false
	}

	return val
}
