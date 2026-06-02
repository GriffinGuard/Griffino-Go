// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

package griffino

import (
	"reflect"
	"strconv"
	"strings"
)

// structTagName is the reflection tag namespace used to declare config fields on
// a plugin author's config struct.
const structTagName = "griffino"

// ParseConfigModel reflects over model (a struct or pointer-to-struct) and builds
// a ConfigModelDefinition of the given kind. Every exported field carrying a
// `griffino:"..."` struct tag becomes one ConfigField, in declaration order.
//
// Tag grammar: a comma-separated list of attr=value pairs. Recognized keys:
//
//	key         (required) the configuration key emitted to config.*.json
//	type        one of string|int|float|boolean|password|options|multiline_string;
//	            when omitted the type is inferred from the Go field kind
//	name        (required) human-readable field name
//	description (required) human-readable description
//	optional    bool (default false)
//	default     the default value; converted to the field's Go type
//	group       UI grouping label
//	values      options for an "options" field: "v1:Display 1|v2:Display 2"
//	validation  numeric bounds: "min:N;max:M" (also accepts minimum/maximum)
//
// Limitation: because attribute pairs are separated by commas, an attribute value
// may not itself contain a comma. The "values" attribute uses ':' and '|' as
// separators and so cannot contain those characters either.
func ParseConfigModel(model any, kind ConfigKind) (ConfigModelDefinition, error) {
	if model == nil {
		return ConfigModelDefinition{}, configError("config model must not be nil")
	}
	t := reflect.TypeOf(model)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return ConfigModelDefinition{}, configError("config model must be a struct or pointer to struct, got %s", t.Kind())
	}

	var fields []ConfigField
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			// Unexported field: skip.
			continue
		}
		tag, ok := sf.Tag.Lookup(structTagName)
		if !ok {
			continue
		}
		field, err := parseFieldTag(sf, tag)
		if err != nil {
			return ConfigModelDefinition{}, err
		}
		fields = append(fields, field)
	}
	return ConfigModelDefinition{Kind: kind, Fields: fields}, nil
}

// ParseBootConfig parses model as a boot configuration model.
func ParseBootConfig(model any) (ConfigModelDefinition, error) {
	return ParseConfigModel(model, ConfigKindBoot)
}

// ParseUserConfig parses model as a user configuration model.
func ParseUserConfig(model any) (ConfigModelDefinition, error) {
	return ParseConfigModel(model, ConfigKindUser)
}

// parseFieldTag converts a single struct field and its griffino tag into a
// ConfigField.
func parseFieldTag(sf reflect.StructField, tag string) (ConfigField, error) {
	attrs, err := parseTagAttrs(tag)
	if err != nil {
		return ConfigField{}, err
	}

	key, ok := attrs["key"]
	if !ok || key == "" {
		return ConfigField{}, configError("config field %s: missing required tag attribute \"key\"", sf.Name)
	}
	name, ok := attrs["name"]
	if !ok || name == "" {
		return ConfigField{}, configError("config field %s: missing required tag attribute \"name\"", sf.Name)
	}
	description, ok := attrs["description"]
	if !ok {
		return ConfigField{}, configError("config field %s: missing required tag attribute \"description\"", sf.Name)
	}

	field := ConfigField{
		Key:         key,
		Name:        name,
		Description: description,
	}

	if optRaw, ok := attrs["optional"]; ok {
		opt, err := strconv.ParseBool(strings.TrimSpace(optRaw))
		if err != nil {
			return ConfigField{}, configError("config field %s: invalid optional value %q", sf.Name, optRaw)
		}
		field.Optional = opt
	}

	if grp, ok := attrs["group"]; ok {
		field.Group = grp
	}

	fieldType, err := resolveFieldType(sf, attrs)
	if err != nil {
		return ConfigField{}, err
	}
	field.Type = fieldType

	if valuesRaw, ok := attrs["values"]; ok {
		field.Values = parseValues(valuesRaw)
	}

	if validationRaw, ok := attrs["validation"]; ok {
		validation, err := parseValidation(sf.Name, validationRaw)
		if err != nil {
			return ConfigField{}, err
		}
		field.Validation = validation
	}

	if defRaw, ok := attrs["default"]; ok {
		def, err := convertDefault(sf, fieldType, defRaw)
		if err != nil {
			return ConfigField{}, err
		}
		field.HasDefault = true
		field.Default = def
	}

	return field, nil
}

// parseTagAttrs splits a tag into its attr=value pairs. Attribute values may
// contain spaces but not commas (see ParseConfigModel for the limitation).
func parseTagAttrs(tag string) (map[string]string, error) {
	attrs := make(map[string]string)
	for _, pair := range strings.Split(tag, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		eq := strings.IndexByte(pair, '=')
		if eq < 0 {
			return nil, configError("invalid tag segment %q: expected attr=value", pair)
		}
		attrKey := strings.TrimSpace(pair[:eq])
		attrVal := pair[eq+1:]
		if attrKey == "" {
			return nil, configError("invalid tag segment %q: empty attribute name", pair)
		}
		attrs[attrKey] = attrVal
	}
	return attrs, nil
}

// resolveFieldType returns the explicit type from the tag, or infers it from the
// Go field kind.
func resolveFieldType(sf reflect.StructField, attrs map[string]string) (ConfigFieldType, error) {
	if raw, ok := attrs["type"]; ok {
		t := ConfigFieldType(strings.TrimSpace(raw))
		switch t {
		case ConfigTypeString, ConfigTypeInt, ConfigTypeFloat, ConfigTypeBoolean,
			ConfigTypePassword, ConfigTypeOptions, ConfigTypeMultilineString:
			return t, nil
		default:
			return "", configError("config field %s: unknown type %q", sf.Name, raw)
		}
	}
	switch sf.Type.Kind() {
	case reflect.String:
		return ConfigTypeString, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return ConfigTypeInt, nil
	case reflect.Float32, reflect.Float64:
		return ConfigTypeFloat, nil
	case reflect.Bool:
		return ConfigTypeBoolean, nil
	default:
		return "", configError("config field %s: cannot infer config field type from Go type %s; set type=... explicitly", sf.Name, sf.Type.Kind())
	}
}

// parseValues parses the "values" attribute into a slice of ConfigOption. The
// format is "value1:Display 1|value2:Display 2". A value entry with no ':' uses
// the value as its own display text.
func parseValues(raw string) []ConfigOption {
	var options []ConfigOption
	for _, item := range strings.Split(raw, "|") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		value := item
		display := item
		if idx := strings.IndexByte(item, ':'); idx >= 0 {
			value = strings.TrimSpace(item[:idx])
			display = strings.TrimSpace(item[idx+1:])
		}
		options = append(options, ConfigOption{Value: value, Display: display})
	}
	return options
}

// parseValidation parses the "validation" attribute ("min:N;max:M") into a map
// keyed by "minimum"/"maximum". Bounds are emitted as int when integral and
// float64 otherwise.
func parseValidation(fieldName, raw string) (map[string]any, error) {
	validation := make(map[string]any)
	for _, item := range strings.Split(raw, ";") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		idx := strings.IndexByte(item, ':')
		if idx < 0 {
			return nil, configError("config field %s: invalid validation segment %q: expected key:value", fieldName, item)
		}
		key := strings.TrimSpace(item[:idx])
		valStr := strings.TrimSpace(item[idx+1:])
		switch key {
		case "min", "minimum":
			key = "minimum"
		case "max", "maximum":
			key = "maximum"
		default:
			return nil, configError("config field %s: unknown validation key %q", fieldName, key)
		}
		validation[key] = parseNumber(valStr)
	}
	if len(validation) == 0 {
		return nil, nil
	}
	return validation, nil
}

// parseNumber returns an int when s is integral, otherwise a float64, falling
// back to the original string when it is not numeric.
func parseNumber(s string) any {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}

// convertDefault converts the raw default string to the field's resolved type.
func convertDefault(sf reflect.StructField, fieldType ConfigFieldType, raw string) (any, error) {
	switch fieldType {
	case ConfigTypeInt:
		n, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil {
			return nil, configError("config field %s: default %q is not an integer", sf.Name, raw)
		}
		return n, nil
	case ConfigTypeFloat:
		f, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil {
			return nil, configError("config field %s: default %q is not a float", sf.Name, raw)
		}
		return f, nil
	case ConfigTypeBoolean:
		b, err := strconv.ParseBool(strings.TrimSpace(raw))
		if err != nil {
			return nil, configError("config field %s: default %q is not a boolean", sf.Name, raw)
		}
		return b, nil
	default:
		return raw, nil
	}
}
