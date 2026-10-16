package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var durationType = reflect.TypeOf(time.Duration(0))
var timeType = reflect.TypeOf(time.Time{})

// applyEnv recursively traverses target (which must be a non-nil pointer to a struct),
// and populates fields annotated with the `env:"KEY"` tag using values from lookup.
// Supported field types: string, bool, int*, uint*, float*, time.Duration, []string,
// pointers to scalar types, and nested structs (value or pointer).
func applyEnv(target any, lookup func(string) (string, bool)) error {
	if target == nil {
		return nil
	}
	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Pointer || val.IsNil() {
		return fmt.Errorf("target must be a non-nil pointer to struct")
	}
	elem := val.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to struct, got pointer to %s", elem.Kind())
	}

	return applyEnvStruct(elem, lookup)
}

func applyEnvStruct(val reflect.Value, lookup func(string) (string, bool)) error {
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		fieldVal := val.Field(i)
		structField := typ.Field(i)

		if !structField.IsExported() {
			continue
		}

		// Handle nested struct or pointer to struct
		if structField.Type.Kind() == reflect.Struct && structField.Type != timeType {
			if err := applyEnvStruct(fieldVal, lookup); err != nil {
				return err
			}
		} else if structField.Type.Kind() == reflect.Pointer && structField.Type.Elem().Kind() == reflect.Struct && structField.Type.Elem() != timeType {
			if fieldVal.IsNil() {
				if hasAnyEnvSet(structField.Type.Elem(), lookup) {
					newStruct := reflect.New(structField.Type.Elem())
					if err := applyEnvStruct(newStruct.Elem(), lookup); err != nil {
						return err
					}
					fieldVal.Set(newStruct)
				}
			} else {
				if err := applyEnvStruct(fieldVal.Elem(), lookup); err != nil {
					return err
				}
			}
		}

		// Check env tag on the field itself
		envKey := structField.Tag.Get("env")
		if envKey == "" {
			continue
		}

		rawVal, exists := lookup(envKey)
		if !exists {
			continue
		}

		if err := setFieldValue(fieldVal, rawVal, envKey); err != nil {
			return err
		}
	}
	return nil
}

func hasAnyEnvSet(t reflect.Type, lookup func(string) (string, bool)) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}
		if key := sf.Tag.Get("env"); key != "" {
			if _, ok := lookup(key); ok {
				return true
			}
		}
		if sf.Type.Kind() == reflect.Struct && sf.Type != timeType {
			if hasAnyEnvSet(sf.Type, lookup) {
				return true
			}
		} else if sf.Type.Kind() == reflect.Pointer && sf.Type.Elem().Kind() == reflect.Struct && sf.Type.Elem() != timeType {
			if hasAnyEnvSet(sf.Type.Elem(), lookup) {
				return true
			}
		}
	}
	return false
}

func setFieldValue(fieldVal reflect.Value, rawVal string, envKey string) error {
	target := fieldVal
	if target.Kind() == reflect.Pointer {
		if target.IsNil() {
			target.Set(reflect.New(target.Type().Elem()))
		}
		target = target.Elem()
	}

	if target.Type() == durationType {
		d, err := time.ParseDuration(rawVal)
		if err != nil {
			return fmt.Errorf("env %s: %w", envKey, err)
		}
		target.SetInt(int64(d))
		return nil
	}

	switch target.Kind() {
	case reflect.String:
		target.SetString(rawVal)
		return nil

	case reflect.Bool:
		b, err := strconv.ParseBool(rawVal)
		if err != nil {
			return fmt.Errorf("env %s: %w", envKey, err)
		}
		target.SetBool(b)
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		iv, err := strconv.ParseInt(rawVal, 10, 64)
		if err != nil {
			return fmt.Errorf("env %s: %w", envKey, err)
		}
		target.SetInt(iv)
		return nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uv, err := strconv.ParseUint(rawVal, 10, 64)
		if err != nil {
			return fmt.Errorf("env %s: %w", envKey, err)
		}
		target.SetUint(uv)
		return nil

	case reflect.Float32, reflect.Float64:
		fv, err := strconv.ParseFloat(rawVal, target.Type().Bits())
		if err != nil {
			return fmt.Errorf("env %s: %w", envKey, err)
		}
		target.SetFloat(fv)
		return nil

	case reflect.Slice:
		if target.Type().Elem().Kind() == reflect.String {
			trimmed := strings.TrimSpace(rawVal)
			if trimmed == "" {
				target.Set(reflect.MakeSlice(target.Type(), 0, 0))
				return nil
			}
			parts := strings.Split(trimmed, ",")
			slice := reflect.MakeSlice(target.Type(), len(parts), len(parts))
			for i, p := range parts {
				slice.Index(i).SetString(strings.TrimSpace(p))
			}
			target.Set(slice)
			return nil
		}
		return fmt.Errorf("env %s: unsupported slice element type %s", envKey, target.Type().Elem())

	default:
		return fmt.Errorf("env %s: unsupported field type %s", envKey, target.Type())
	}
}
