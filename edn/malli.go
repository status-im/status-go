package edn

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

var primitiveGoTypeToMalli = map[string]string{
	"bool":    ":boolean",
	"string":  ":string",
	"float64": ":double",
	"int":     ":int",
	"int32":   ":int",
	"uint8":   ":int",
	"uint32":  ":int",
	"uint64":  ":int",
}

func convertCamelToKebab(input string) string {
	re := regexp.MustCompile("([a-z])([A-Z])")
	kebab := re.ReplaceAllString(input, "${1}-${2}")
	return strings.ToLower(kebab)
}

func handleType(fieldType reflect.Type) any {
	kind := fieldType.Kind()

	if kind == reflect.Interface {
		return ":any"
	}

	if kind == reflect.Struct {
		return ConvertStructToMalli(fieldType)
	}

	if kind == reflect.Slice {
		elemType := fieldType.Elem()
		// Handle the type that the pointer points to.
		if elemType.Kind() == reflect.Ptr {
			return []any{handleType(elemType.Elem())}
		}
		return []any{handleType(elemType)}
	}

	return primitiveGoTypeToMalli[kind.String()]
}

func ConvertStructToMalli(typ reflect.Type) map[string]any {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem() // Dereference if it's a pointer to a struct type
	}

	if typ.Kind() != reflect.Struct {
		panic("ConvertStructToMalli expects a struct type or pointer to a struct type")
	}

	malliSchema := make(map[string]any)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fmt.Println(field)

		if !field.IsExported() {
			continue
		}

		kebabFieldName := convertCamelToKebab(field.Name)

		if field.Type.Kind() == reflect.Ptr {
			malliSchema[kebabFieldName] = handleType(field.Type.Elem())
		} else {
			malliSchema[kebabFieldName] = handleType(field.Type)
		}
	}
	return malliSchema
}
