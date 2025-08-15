package utilities

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/fabioluissilva/microservicetemplate/commonconfig"
	"github.com/fabioluissilva/microservicetemplate/commonlogger"
)

// maskSensitive masks sensitive fields as requested.
func maskSensitive(value string) string {
	if len(value) >= 8 {
		return value[:2] + "****" + value[len(value)-2:]
	}
	return "****"
}

func ToMaskedJSON(cfg any) (string, error) {
	v := reflect.ValueOf(cfg)

	// Unwrap interface and pointer layers until we reach a struct
	for {
		if v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return "{}", nil
			}
			v = v.Elem()
			continue
		}
		break
	}

	if v.Kind() != reflect.Struct {
		return "", fmt.Errorf("ToMaskedJSON: expected struct or *struct, got %s", v.Kind())
	}

	m, err := structToMaskedMap(v)
	if err != nil {
		return "", err
	}

	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func structToMaskedMap(v reflect.Value) (map[string]any, error) {
	t := v.Type()
	out := make(map[string]any, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		fv := v.Field(i)

		// Handle pointer-to-struct fields
		if fv.Kind() == reflect.Pointer && !fv.IsNil() && fv.Elem().Kind() == reflect.Struct {
			fv = fv.Elem()
		}

		tag := sf.Tag.Get("mapstructure")
		tagParts := strings.Split(tag, ",")
		key := strings.TrimSpace(tagParts[0]) // first part before comma (may be "")

		// squash if anonymous embed OR explicit ",squash" in tag
		hasSquash := false
		for _, p := range tagParts[1:] {
			if strings.TrimSpace(p) == "squash" {
				hasSquash = true
				break
			}
		}
		if sf.Anonymous || hasSquash {
			if fv.Kind() == reflect.Struct {
				child, err := structToMaskedMap(fv)
				if err != nil {
					return nil, err
				}
				for k, val := range child {
					out[k] = val
				}
				continue
			}
		}

		// If no explicit key, fall back to field name
		if key == "" {
			key = sf.Name
		}

		var val any

		switch fv.Kind() {
		case reflect.Struct:
			child, err := structToMaskedMap(fv)
			if err != nil {
				return nil, err
			}
			val = child
		case reflect.Slice, reflect.Array:
			arr := make([]any, fv.Len())
			for j := 0; j < fv.Len(); j++ {
				elem := fv.Index(j)
				if elem.Kind() == reflect.Pointer && !elem.IsNil() && elem.Elem().Kind() == reflect.Struct {
					elem = elem.Elem()
				}
				if elem.Kind() == reflect.Struct {
					child, err := structToMaskedMap(elem)
					if err != nil {
						return nil, err
					}
					arr[j] = child
				} else {
					arr[j] = elem.Interface()
				}
			}
			val = arr
		case reflect.Map:
			m := make(map[string]any, fv.Len())
			iter := fv.MapRange()
			for iter.Next() {
				k := fmt.Sprint(iter.Key().Interface())
				ev := iter.Value()
				if ev.Kind() == reflect.Pointer && !ev.IsNil() && ev.Elem().Kind() == reflect.Struct {
					ev = ev.Elem()
				}
				if ev.Kind() == reflect.Struct {
					child, err := structToMaskedMap(ev)
					if err != nil {
						return nil, err
					}
					m[k] = child
				} else {
					m[k] = ev.Interface()
				}
			}
			val = m
		default:
			val = fv.Interface()
		}

		// Mask sensitive string leaves
		if sf.Tag.Get("sensitive") == "true" && fv.Kind() == reflect.String {
			s := fv.String()
			val = maskSensitive(s)
		}

		out[key] = val
	}
	return out, nil
}

func ReadReleaseNotes() (string, error) {
	releaseNotesPath := "releasenotes.txt"
	commonlogger.Debug(fmt.Sprintf("Reading Release Notes from: %s", releaseNotesPath), "service", commonconfig.GetConfig().GetServiceName())
	content, err := os.ReadFile(releaseNotesPath)
	if err != nil {
		commonlogger.Error("Error reading release notes:", "error", err, "service", commonconfig.GetConfig().GetServiceName())
		return "", err
	}
	return string(content), nil
}
