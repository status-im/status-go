package fetcher

import (
	"errors"

	"github.com/xeipuuv/gojsonschema"
)

func validateJsonAgainstSchema(jsonData string, schemaLoader gojsonschema.JSONLoader) error {
	docLoader := gojsonschema.NewStringLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, docLoader)
	if err != nil {
		return err
	}

	if !result.Valid() {
		return errors.New("token list does not match schema")
	}

	return nil
}
