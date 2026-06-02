# DeepSeek example plugin

A minimal Griffino plugin written with the Go SDK. It registers a single
capability provider of type `ai.chat.model` (`ai_chat_model`) and declares boot
and user configuration via struct tags.

The chat handler in [`plugin/plugin.go`](plugin/plugin.go) is an illustrative
stub: it validates the incoming payload and returns a canned completion. A real
plugin would read its configuration and call the DeepSeek API (an
OpenAI-compatible chat completions endpoint) from the same place.

## Layout

| Path | Purpose |
| --- | --- |
| `main.go` | Runtime entry point — builds the client and runs the serve loop. |
| `gen/main.go` | Generator — writes the platform artifact files. |
| `plugin/plugin.go` | Shared metadata, config structs, and the `NewClient` builder. |
| `plugin.manifest.json`, `config.boot.json`, `config.user.json`, `plugin.boot.yml` | Generated artifacts. |
| `Dockerfile` | Multi-stage build producing the plugin binary. |

## Build

From the repository root:

```bash
go build ./examples/deepseek
```

## Generate the artifacts

Run the generator from this directory so the files land alongside the example:

```bash
cd examples/deepseek
go run ./gen
```

This (re)writes `plugin.manifest.json`, `config.boot.json`, `config.user.json`,
and `plugin.boot.yml`. Existing files are backed up to `*.1`, `*.2`, ... first.

## Run

The plugin reads its broker credentials and identity from the environment the
platform injects (`RABBITMQ_HOST`, `RABBITMQ_USER`, `RABBITMQ_PASSWORD`,
`GRIFFINO_PLUGIN_ID`, ...). With those set:

```bash
go run ./examples/deepseek
```

It serves the registered provider until it receives `SIGINT`/`SIGTERM`.

## Container image

The provided `Dockerfile` is built from the repository root so it can compile
against the local SDK module:

```bash
docker build -t griffino-deepseek -f examples/deepseek/Dockerfile .
```
