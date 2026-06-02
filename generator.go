// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

package griffino

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// SYSTEM_ENVIRONMENT_VARIABLES is the fixed set of broker/cache environment
// variables the platform injects into every plugin service container. It mirrors
// the Python SDK generator.
var SYSTEM_ENVIRONMENT_VARIABLES = []string{
	"RABBITMQ_HOST={{system.rabbitmq.host}}",
	"RABBITMQ_PORT={{system.rabbitmq.port}}",
	"RABBITMQ_USER={{system.rabbitmq.user}}",
	"RABBITMQ_PASSWORD={{system.rabbitmq.password}}",
	"REDIS_HOST={{system.redis.host}}",
	"REDIS_PORT={{system.redis.port}}",
	"REDIS_USER={{system.redis.user}}",
	"REDIS_PASSWORD={{system.redis.password}}",
}

// slotsToManifest renders capability slots into their manifest form.
func slotsToManifest(slots []CapabilitySlot) []map[string]any {
	out := make([]map[string]any, 0, len(slots))
	for _, slot := range slots {
		out = append(out, map[string]any{
			"id":          slot.ID,
			"name":        slot.Name.toManifest(),
			"description": slot.Description.toManifest(),
		})
	}
	return out
}

// GenerateManifest builds the plugin.manifest.json data. It mirrors the Python
// SDK generator's generate_manifest_data exactly, including the conditional
// inclusion of each member. hasBootConfig/hasUserConfig drive the
// configurationFiles block.
func GenerateManifest(
	meta PluginMetadata,
	providers []ProviderRegistration,
	consumerCaps []ConsumerCapabilityRegistration,
	actions []ActionRegistration,
	hasBootConfig, hasUserConfig bool,
) map[string]any {
	capabilities := make([]map[string]any, 0, len(providers)+len(consumerCaps))
	for _, provider := range providers {
		name := provider.Name
		if name.Default == "" && name.Key == "" {
			name = I18nString{Default: provider.CapabilityID}
		}
		capability := map[string]any{
			"id":          provider.CapabilityID,
			"role":        "provider",
			"type":        provider.CapabilityType,
			"name":        name.toManifest(),
			"description": provider.Description.toManifest(),
			"entryPoint": map[string]any{
				"type": "rabbitmq_topic",
				"details": map[string]any{
					"requestTopicPattern": fmt.Sprintf("invoke.%s.%s.v1", meta.ID, provider.CapabilityID),
				},
			},
		}
		if provider.StandardInterfaceRef != "" {
			capability["standardInterfaceRef"] = provider.StandardInterfaceRef
		}
		if provider.InputSchemaRef != "" {
			capability["inputSchemaRef"] = provider.InputSchemaRef
		}
		if provider.OutputSchemaRef != "" {
			capability["outputSchemaRef"] = provider.OutputSchemaRef
		}
		if len(provider.Slots) > 0 {
			capability["slots"] = slotsToManifest(provider.Slots)
		}
		if provider.TimeoutMS != nil {
			capability["defaultTimeoutMilliseconds"] = *provider.TimeoutMS
		}
		capabilities = append(capabilities, capability)
	}
	for _, consumer := range consumerCaps {
		capability := map[string]any{
			"id":                     consumer.CapabilityID,
			"role":                   "consumer",
			"type":                   consumer.CapabilityType,
			"consumesCapabilityType": consumer.CapabilityType,
			"name":                   consumer.Name.toManifest(),
			"description":            consumer.Description.toManifest(),
			"optional":               consumer.Optional,
		}
		if len(consumer.Slots) > 0 {
			capability["slots"] = slotsToManifest(consumer.Slots)
		}
		capabilities = append(capabilities, capability)
	}

	manifest := map[string]any{
		"griffinoPluginManifestVersion": "1.0",
		"id":                            meta.ID,
		"pluginVersion":                 meta.Version,
		"name":                          meta.Name.toManifest(),
		"description":                   meta.Description.toManifest(),
		"capabilities":                  capabilities,
	}

	if len(actions) > 0 {
		actionEntries := make([]map[string]any, 0, len(actions))
		for _, action := range actions {
			confirmation := map[string]any{"required": action.ConfirmationRequired}
			if action.ConfirmationMessage != nil {
				confirmation["message"] = action.ConfirmationMessage.toManifest()
			}
			entry := map[string]any{
				"id":           action.ActionID,
				"name":         action.Name.toManifest(),
				"description":  action.Description.toManifest(),
				"confirmation": confirmation,
			}
			if action.CooldownMS != nil {
				entry["cooldownMs"] = *action.CooldownMS
			}
			actionEntries = append(actionEntries, entry)
		}
		manifest["actions"] = actionEntries
	}

	if meta.ManifestSchema != "" {
		manifest["$schema"] = meta.ManifestSchema
	}
	if meta.Author != "" {
		manifest["author"] = meta.Author
	}
	// site always present, defaulting to "".
	manifest["site"] = meta.Site
	if meta.Tutorial != "" {
		manifest["tutorial"] = meta.Tutorial
	}
	if meta.Internationalization != nil {
		i18n := meta.Internationalization
		langs := i18n.SupportedLanguages
		if langs == nil {
			langs = []string{}
		}
		manifest["internationalization"] = map[string]any{
			"defaultLanguage":    i18n.DefaultLanguage,
			"supportedLanguages": langs,
			"localizationFiles":  i18n.LocalizationFiles,
		}
	}
	if len(meta.PermissionsRequested) > 0 {
		permissions := make([]map[string]any, 0, len(meta.PermissionsRequested))
		for _, permission := range meta.PermissionsRequested {
			permissions = append(permissions, map[string]any{
				"name":        permission.Name,
				"description": permission.Description.toManifest(),
				"optional":    permission.Optional,
			})
		}
		manifest["permissionsRequested"] = permissions
	}

	configurationFiles := map[string]string{}
	if hasBootConfig {
		configurationFiles["bootConfig"] = "config.boot.json"
		configurationFiles["runtimeBoot"] = "plugin.boot.yml"
	}
	if hasUserConfig {
		configurationFiles["userConfig"] = "config.user.json"
	}
	if len(configurationFiles) > 0 {
		manifest["configurationFiles"] = configurationFiles
	}

	return manifest
}

// buildConfigFieldEntry renders a single config field into its config.*.json
// entry, mirroring the Python SDK generator's build_config_field_entry.
func buildConfigFieldEntry(field ConfigField) map[string]any {
	entry := map[string]any{
		"key":         field.Key,
		"type":        string(field.Type),
		"name":        field.Name,
		"description": field.Description,
		"optional":    field.Optional,
	}
	if field.HasDefault {
		entry["default"] = field.Default
	}
	if field.Validation != nil {
		entry["validation"] = field.Validation
	}
	if field.Group != "" {
		entry["group"] = field.Group
	}
	if len(field.Values) > 0 {
		values := make([]map[string]any, 0, len(field.Values))
		for _, option := range field.Values {
			values = append(values, map[string]any{
				"value":   option.Value,
				"display": option.Display,
			})
		}
		entry["values"] = values
	}
	return entry
}

func configFieldEntries(def ConfigModelDefinition) []map[string]any {
	entries := make([]map[string]any, 0, len(def.Fields))
	for _, field := range def.Fields {
		entries = append(entries, buildConfigFieldEntry(field))
	}
	return entries
}

// GenerateBootConfig builds the config.boot.json data, wrapping the fields in a
// single service entry. It mirrors the Python SDK generator for the boot kind.
func GenerateBootConfig(meta PluginMetadata, def ConfigModelDefinition) map[string]any {
	serviceID := meta.MainServiceID
	if serviceID == "" {
		serviceID = "main-service"
	}
	payload := map[string]any{
		"GriffinoPluginConfigVersion": "1.0",
		"pluginId":                    meta.ID,
		"pluginVersion":               meta.Version,
		"name":                        meta.Name.Default,
		"site":                        meta.Site,
		"services": []map[string]any{
			{
				"id":      serviceID,
				"configs": configFieldEntries(def),
			},
		},
	}
	if meta.BootConfigSchema != "" {
		payload["$schema"] = meta.BootConfigSchema
	}
	if meta.Tutorial != "" {
		payload["tutorial"] = meta.Tutorial
	}
	return payload
}

// GenerateUserConfig builds the config.user.json data with the fields at the top
// level. It mirrors the Python SDK generator for the user kind.
func GenerateUserConfig(meta PluginMetadata, def ConfigModelDefinition) map[string]any {
	payload := map[string]any{
		"GriffinoPluginConfigVersion": "1.0",
		"pluginId":                    meta.ID,
		"pluginVersion":               meta.Version,
		"name":                        meta.Name.Default,
		"site":                        meta.Site,
		"configs":                     configFieldEntries(def),
	}
	if meta.UserConfigSchema != "" {
		payload["$schema"] = meta.UserConfigSchema
	}
	if meta.Tutorial != "" {
		payload["tutorial"] = meta.Tutorial
	}
	return payload
}

// GenerateBootYAMLData returns the structured plugin.boot.yml data for the main
// service: the system environment variables followed by the boot config keys. It
// requires meta.MainServiceID and a boot-kind definition.
func GenerateBootYAMLData(meta PluginMetadata, def ConfigModelDefinition) (map[string]any, error) {
	if meta.MainServiceID == "" {
		return nil, configError("MainServiceID is required to generate plugin.boot.yml")
	}
	if def.Kind != ConfigKindBoot {
		return nil, configError("plugin.boot.yml requires a boot config model")
	}
	environment := make([]string, 0, len(SYSTEM_ENVIRONMENT_VARIABLES)+len(def.Fields))
	environment = append(environment, SYSTEM_ENVIRONMENT_VARIABLES...)
	for _, field := range def.Fields {
		environment = append(environment, field.Key)
	}
	return map[string]any{
		"pluginBootSpecVersion": "1.0",
		"pluginId":              meta.ID,
		"pluginVersion":         meta.Version,
		"mainServiceId":         meta.MainServiceID,
		"services": map[string]any{
			meta.MainServiceID: map[string]any{
				"image":       "",
				"environment": environment,
				"ports":       []string{},
				"volumes":     []string{},
			},
		},
	}, nil
}

// WriteJSONFile writes data as 2-space-indented JSON with a trailing newline,
// backing up any existing file first (see backupExistingFile).
func WriteJSONFile(path string, data any) error {
	if err := backupExistingFile(path); err != nil {
		return err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return configError("failed to encode JSON for %s: %v", path, err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return configError("failed to write %s: %v", path, err)
	}
	return nil
}

// WriteBootYAMLFile writes plugin.boot.yml by hand-building the lines, mirroring
// the Python SDK generator's write_boot_yaml_file (including the
// build_instructions / image / ports / volumes TODO scaffold).
func WriteBootYAMLFile(path string, meta PluginMetadata, def ConfigModelDefinition) error {
	data, err := GenerateBootYAMLData(meta, def)
	if err != nil {
		return err
	}
	if err := backupExistingFile(path); err != nil {
		return err
	}

	lines := []string{
		`pluginBootSpecVersion: "1.0"`,
		fmt.Sprintf(`pluginId: "%s"`, data["pluginId"]),
		fmt.Sprintf(`pluginVersion: "%s"`, data["pluginVersion"]),
		fmt.Sprintf(`mainServiceId: "%s"`, data["mainServiceId"]),
		"",
		"services:",
	}

	services := data["services"].(map[string]any)
	service := services[meta.MainServiceID].(map[string]any)
	environment := buildBootYAMLEnvironment(service["environment"].([]string))

	lines = append(lines,
		fmt.Sprintf("  %s:", meta.MainServiceID),
		"    # TODO: choose either build_instructions or image",
		"    build_instructions:",
		`      context: "."`,
		`      dockerfile: "Dockerfile"`,
		"",
		`    image: ""  # TODO: set your image URI`,
		"",
		"    environment:",
	)
	for _, key := range environment {
		lines = append(lines, fmt.Sprintf(`      - "%s"`, key))
	}
	lines = append(lines,
		"    ports: []      # TODO: declare exposed ports",
		"    volumes: []    # TODO: declare persistent volumes",
		"",
		"  # TODO: add auxiliary services (pre-built images) here",
	)

	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return configError("failed to write %s: %v", path, err)
	}
	return nil
}

// systemEnvKeys returns the set of system environment variable names (the part
// before '=').
func systemEnvKeys() map[string]bool {
	keys := make(map[string]bool, len(SYSTEM_ENVIRONMENT_VARIABLES))
	for _, entry := range SYSTEM_ENVIRONMENT_VARIABLES {
		key := entry
		if idx := strings.IndexByte(entry, '='); idx >= 0 {
			key = entry[:idx]
		}
		keys[key] = true
	}
	return keys
}

// buildBootYAMLEnvironment places the system environment variables first,
// followed by the plugin-specific entries, mirroring the Python SDK generator.
func buildBootYAMLEnvironment(environment []string) []string {
	systemKeys := systemEnvKeys()
	plugin := make([]string, 0, len(environment))
	for _, entry := range environment {
		key := entry
		if idx := strings.IndexByte(entry, '='); idx >= 0 {
			key = entry[:idx]
		}
		if !systemKeys[key] {
			plugin = append(plugin, entry)
		}
	}
	out := make([]string, 0, len(SYSTEM_ENVIRONMENT_VARIABLES)+len(plugin))
	out = append(out, SYSTEM_ENVIRONMENT_VARIABLES...)
	out = append(out, plugin...)
	return out
}

// backupExistingFile renames an existing file at path to path.1, path.2, ...
// choosing the first free index, mirroring the Python SDK generator. It is a
// no-op when path does not exist.
func backupExistingFile(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return configError("failed to stat %s: %v", path, err)
	}
	for index := 1; ; index++ {
		backupPath := fmt.Sprintf("%s.%d", path, index)
		if _, err := os.Stat(backupPath); err != nil {
			if os.IsNotExist(err) {
				if err := os.Rename(path, backupPath); err != nil {
					return configError("failed to back up %s: %v", path, err)
				}
				return nil
			}
			return configError("failed to stat %s: %v", backupPath, err)
		}
	}
}
