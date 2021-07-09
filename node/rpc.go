package node

import (
	"reflect"
	"unicode"
)

// formatName converts to first character of name to lowercase.
func formatName(name string) string {
	ret := []rune(name)
	if len(ret) > 0 {
		ret[0] = unicode.ToLower(ret[0])
	}
	return string(ret)
}

// addSuitableCallbacks iterates over the methods of the given type and adds them to
// the methods list
// This is taken from go-ethereum services
func addSuitableCallbacks(receiver reflect.Value, namespace string, methods map[string]bool) {
	typ := receiver.Type()
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		if method.PkgPath != "" {
			continue // method not exported
		}
		name := formatName(method.Name)
		methods[namespace+"_"+name] = true
	}
	return
}
