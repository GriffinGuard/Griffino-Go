// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

// Command gen writes the platform artifact files for the DeepSeek example
// (plugin.manifest.json, config.boot.json, config.user.json, plugin.boot.yml)
// into the example directory. Run it from examples/deepseek:
//
//	go run ./gen
//
// It reuses the same metadata, provider registration, and config structs as the
// runtime entry point, so the generated files always match the code.
package main

import (
	"log"

	"github.com/GriffinGuard/Griffino-Go"
	"github.com/GriffinGuard/Griffino-Go/examples/deepseek/plugin"
)

func main() {
	client, err := plugin.NewClient()
	if err != nil {
		log.Fatalf("gen: failed to build client: %v", err)
	}

	if err := griffino.WriteArtifacts(client, ".", &plugin.BootConfig{}, &plugin.UserConfig{}); err != nil {
		log.Fatalf("gen: failed to write artifacts: %v", err)
	}

	log.Println("gen: wrote plugin.manifest.json, config.boot.json, config.user.json, plugin.boot.yml")
}
