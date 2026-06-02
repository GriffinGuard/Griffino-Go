# Griffino Go SDK

A Go SDK for building [Griffino](https://github.com/MorCherlf) plugins.
Griffino is a plugin-orchestration platform: plugins run as containers and
communicate with the platform and with one another over RabbitMQ. This SDK
gives a plugin author a small, typed surface for that interaction — register
capabilities, invoke other plugins, declare configuration, and generate the
platform descriptor files.

## Install

```bash
go get github.com/GriffinGuard/Griffino-Go
```

```go
import "github.com/GriffinGuard/Griffino-Go"
```

The package name is `griffino`. Requires Go 1.25 or newer.

## Quickstart

A plugin builds a `Client`, registers what it provides, and runs the serve
loop:

```go
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/GriffinGuard/Griffino-Go"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	client, err := griffino.New(griffino.WithMetadata(griffino.PluginMetadata{
		ID:            "com.example.echo",
		Name:          griffino.I18n("Echo"),
		Version:       "0.1.0",
		MainServiceID: "echo-service",
		Author:        "you",
	}))
	if err != nil {
		log.Fatal(err)
	}

	client.Provider(griffino.ProviderRegistration{
		CapabilityID:   "echo",
		CapabilityType: "text.echo",
		Name:           griffino.I18n("Echo"),
		Handler: func(ctx *griffino.HandlerContext) (map[string]any, error) {
			return map[string]any{"echo": ctx.Payload}, nil
		},
	})

	if err := client.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
```

`New` reads its broker credentials and plugin identity from the environment the
platform injects (`RABBITMQ_HOST`, `RABBITMQ_USER`, `RABBITMQ_PASSWORD`,
`GRIFFINO_PLUGIN_ID`, ...). Use `griffino.WithConfig` to supply them explicitly,
or `griffino.WithTransport` to inject a fake transport in tests.

Call other plugins with `Invoke` (synchronous) and publish events with
`Dispatch` (fire-and-forget).

## Configuration via struct tags

Boot and user configuration are declared as Go structs whose fields carry a
`griffino:"..."` tag:

```go
type BootConfig struct {
	APIKey string `griffino:"key=API_KEY,type=password,name=API Key,description=Your API key.,optional=false,group=API"`
	Tokens int    `griffino:"key=MAX_TOKENS,type=int,name=Max Tokens,description=Token budget.,optional=true,default=4096,validation=min:256;max:8192,group=Model"`
}
```

See [docs/api.md](docs/api.md) for the full tag grammar.

## Generating artifacts

The platform reads static descriptor files for a plugin. Generate them from
your own program with one call:

```go
err := griffino.WriteArtifacts(client, ".", &BootConfig{}, &UserConfig{})
```

This writes `plugin.manifest.json`, and — when the corresponding model is
non-nil — `config.boot.json`, `plugin.boot.yml`, and `config.user.json`. Pass
`nil` to skip a config kind.

## Example

A complete, runnable plugin lives in
[`examples/deepseek`](examples/deepseek): metadata, a capability provider, boot
and user config, a generator, a `Dockerfile`, and the generated artifacts.

## Documentation

- [docs/architecture.md](docs/architecture.md) — the capability model,
  RabbitMQ transport, envelopes and routing keys, and configuration generation.
- [docs/api.md](docs/api.md) — the public API reference.

## License

MIT. See [LICENSE](LICENSE).
