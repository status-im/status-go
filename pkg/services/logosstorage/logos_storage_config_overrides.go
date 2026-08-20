//go:build use_logos_storage

package logosstorage

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/status-im/status-go/params"
)

func ApplyLogosStorageConfigOverrides(cfg *params.LogosStorageConfig, overrides map[string]string) error {
	if cfg == nil || len(overrides) == 0 {
		return nil
	}

	for key, raw := range overrides {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if err := setStructFieldValue(reflect.ValueOf(cfg), key, raw); err != nil {
			return fmt.Errorf("failed to apply LogosStorageConfig override %q: %w", key, err)
		}
	}

	return nil
}

func setStructFieldValue(target reflect.Value, path, raw string) error {
	if target.Kind() != reflect.Pointer {
		return fmt.Errorf("target must be a pointer, got %s", target.Kind())
	}

	current := target.Elem()
	if !current.IsValid() {
		return fmt.Errorf("invalid target for path %q", path)
	}

	parts := strings.Split(path, ".")
	for idx, part := range parts {
		if part == "" {
			return fmt.Errorf("invalid empty segment in path %q", path)
		}
		field := current.FieldByName(part)
		if !field.IsValid() {
			return fmt.Errorf("unknown field %q in path %q", part, path)
		}

		if idx == len(parts)-1 {
			if !field.CanSet() {
				return fmt.Errorf("cannot set field %q in path %q", part, path)
			}
			return assignValue(field, raw)
		}

		switch field.Kind() {
		case reflect.Struct:
			current = field
		case reflect.Pointer:
			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}
			current = field.Elem()
		default:
			return fmt.Errorf("field %q in path %q is not addressable struct or pointer", part, path)
		}
	}

	return nil
}

func assignValue(field reflect.Value, raw string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(raw)
	case reflect.Bool:
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		field.SetBool(parsed)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(raw, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(parsed)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parsed, err := strconv.ParseUint(raw, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(parsed)
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(raw, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetFloat(parsed)
	case reflect.Slice:
		return assignSlice(field, raw)
	default:
		return fmt.Errorf("unsupported field kind %s", field.Kind())
	}

	return nil
}

func assignSlice(field reflect.Value, raw string) error {
	elemKind := field.Type().Elem().Kind()
	switch elemKind {
	case reflect.String:
		if raw == "" {
			field.Set(reflect.Zero(field.Type()))
			return nil
		}

		var parsed []string
		if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
			chunks := strings.SplitSeq(raw, ",")
			for chunk := range chunks {
				chunk = strings.TrimSpace(chunk)
				if chunk == "" {
					continue
				}
				parsed = append(parsed, chunk)
			}
		}
		field.Set(reflect.ValueOf(parsed))
	default:
		return fmt.Errorf("unsupported slice element kind %s", elemKind)
	}

	return nil
}
