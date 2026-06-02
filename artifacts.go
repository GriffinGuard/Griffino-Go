// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

package griffino

import "path/filepath"

// Artifact file names emitted by WriteArtifacts. They match the platform's
// expected layout and the Python SDK output.
const (
	manifestFileName   = "plugin.manifest.json"
	bootConfigFileName = "config.boot.json"
	userConfigFileName = "config.user.json"
	bootYAMLFileName   = "plugin.boot.yml"
)

// WriteArtifacts generates the platform artifact files for a plugin into dir
// using the client's metadata and registered capabilities.
//
// It always writes plugin.manifest.json. When bootModel is non-nil it parses
// the boot config struct and writes config.boot.json and plugin.boot.yml. When
// userModel is non-nil it parses the user config struct and writes
// config.user.json. Both models are plain Go structs carrying `griffino:"..."`
// field tags (see ParseConfigModel). Pass nil to skip a config kind.
//
// The manifest's configurationFiles block reflects which models were supplied.
// This lets a plugin author emit all artifacts from a small generator program
// built on top of the public API, without touching the SDK internals.
func WriteArtifacts(c *Client, dir string, bootModel, userModel any) error {
	meta := c.Metadata()

	manifest := GenerateManifest(
		meta,
		c.Providers(),
		c.ConsumerCapabilities(),
		c.Actions(),
		bootModel != nil,
		userModel != nil,
	)
	if err := WriteJSONFile(filepath.Join(dir, manifestFileName), manifest); err != nil {
		return err
	}

	if bootModel != nil {
		bootDef, err := ParseBootConfig(bootModel)
		if err != nil {
			return err
		}
		bootConfig := GenerateBootConfig(meta, bootDef)
		if err := WriteJSONFile(filepath.Join(dir, bootConfigFileName), bootConfig); err != nil {
			return err
		}
		if err := WriteBootYAMLFile(filepath.Join(dir, bootYAMLFileName), meta, bootDef); err != nil {
			return err
		}
	}

	if userModel != nil {
		userDef, err := ParseUserConfig(userModel)
		if err != nil {
			return err
		}
		userConfig := GenerateUserConfig(meta, userDef)
		if err := WriteJSONFile(filepath.Join(dir, userConfigFileName), userConfig); err != nil {
			return err
		}
	}

	return nil
}
