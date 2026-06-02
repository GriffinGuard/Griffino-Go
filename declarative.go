// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

package griffino

// ConfigFieldType is the platform's configuration field type. It controls how
// the management UI renders and validates a field.
type ConfigFieldType string

// Supported configuration field types, matching the platform schema.
const (
	ConfigTypeString          ConfigFieldType = "string"
	ConfigTypeInt             ConfigFieldType = "int"
	ConfigTypeFloat           ConfigFieldType = "float"
	ConfigTypeBoolean         ConfigFieldType = "boolean"
	ConfigTypePassword        ConfigFieldType = "password"
	ConfigTypeOptions         ConfigFieldType = "options"
	ConfigTypeMultilineString ConfigFieldType = "multiline_string"
)

// ConfigKind distinguishes boot configuration (admin-set, one per plugin) from
// user configuration (per-user).
type ConfigKind string

// Configuration kinds.
const (
	ConfigKindBoot ConfigKind = "boot"
	ConfigKindUser ConfigKind = "user"
)

// ConfigOption is a single choice for an "options" field.
type ConfigOption struct {
	Value   string
	Display string
}

// ConfigField is one fully-resolved configuration field, ready to be rendered
// into config.boot.json / config.user.json. It is produced by reflecting over a
// tagged config struct (see RegisterBootConfig / RegisterUserConfig).
type ConfigField struct {
	Key         string
	Type        ConfigFieldType
	Name        string
	Description string
	Optional    bool
	// HasDefault reports whether Default carries a meaningful value. It mirrors
	// the Python SDK's MISSING sentinel: when false, no "default" member is
	// emitted.
	HasDefault bool
	Default    any
	Validation map[string]any
	Group      string
	Values     []ConfigOption
}

// ConfigModelDefinition is the parsed form of a config struct: its kind plus the
// ordered set of fields the generator emits.
type ConfigModelDefinition struct {
	Kind   ConfigKind
	Fields []ConfigField
}
