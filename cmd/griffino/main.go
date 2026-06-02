// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

// Command griffino is a thin driver that documents the code-driven artifact
// generation flow of the Griffino Go SDK.
//
// Unlike the Python SDK, a Go program cannot introspect a plugin author's module
// at runtime to discover its registrations. Artifact generation is therefore
// driven from the plugin's own Go code by calling the SDK generator functions
// (GenerateManifest / GenerateBootConfig / GenerateUserConfig and the Write*File
// helpers) directly. This CLI explains that flow and the artifacts involved.
package main

import (
	"flag"
	"fmt"
	"os"
)

const usage = `griffino - Griffino Go SDK helper

Usage:
  griffino <command> [flags]

Commands:
  generate    Explain how to generate plugin artifacts from your Go code.
  help        Show this help.

Run "griffino generate -h" for details on the generation flow.
`

const generateUsage = `griffino generate - explain artifact generation

Artifacts are generated from your own Go program, not by this CLI: the Go
toolchain cannot introspect your module at runtime the way the Python SDK does.
In your plugin's code, build a griffino.PluginMetadata and parse your config
structs with griffino.ParseBootConfig / griffino.ParseUserConfig, then call the
generator functions and write the files:

  manifest := griffino.GenerateManifest(meta, providers, consumers, actions,
      hasBootConfig, hasUserConfig)
  _ = griffino.WriteJSONFile("plugin.manifest.json", manifest)

  bootCfg := griffino.GenerateBootConfig(meta, bootDef)
  _ = griffino.WriteJSONFile("config.boot.json", bootCfg)
  _ = griffino.WriteBootYAMLFile("plugin.boot.yml", meta, bootDef)

  userCfg := griffino.GenerateUserConfig(meta, userDef)
  _ = griffino.WriteJSONFile("config.user.json", userCfg)

Flags (for a future griffino.WriteArtifacts driver invoked from your code):
  --all            Generate every artifact.
  --manifest       Generate plugin.manifest.json.
  --boot-config    Generate config.boot.json.
  --user-config    Generate config.user.json.
  --boot-yml       Generate plugin.boot.yml.
  --output-dir     Directory to write artifacts into (default ".").
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		return
	}

	switch os.Args[1] {
	case "generate":
		runGenerate(os.Args[2:])
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "griffino: unknown command %q\n\n", os.Args[1])
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
}

func runGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	fs.Usage = func() { fmt.Print(generateUsage) }

	all := fs.Bool("all", false, "generate every artifact")
	manifest := fs.Bool("manifest", false, "generate plugin.manifest.json")
	bootConfig := fs.Bool("boot-config", false, "generate config.boot.json")
	userConfig := fs.Bool("user-config", false, "generate config.user.json")
	bootYML := fs.Bool("boot-yml", false, "generate plugin.boot.yml")
	outputDir := fs.String("output-dir", ".", "directory to write artifacts into")

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	fmt.Print(generateUsage)
	fmt.Println()
	fmt.Printf("Requested (drive these from your plugin's Go code): all=%t manifest=%t boot-config=%t user-config=%t boot-yml=%t output-dir=%q\n",
		*all, *manifest, *bootConfig, *userConfig, *bootYML, *outputDir)
}
