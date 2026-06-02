# Architecture

Griffino is a plugin-orchestration platform. Plugins run as containers and
communicate with the platform and with one another over RabbitMQ. The Go SDK
gives a plugin author a typed entry point for that interaction. This document
describes the model the SDK implements.

## The capability model

A plugin declares what it can do and what it needs through three kinds of
registration on a `griffino.Client`:

- **Provider** — a capability this plugin *offers*. Each provider has a stable
  capability id, a capability type (e.g. `ai.chat.model`), and a `Handler` that
  receives an invocation and returns a result. Providers are reachable
  synchronously via `Invoke`.
- **Consumer capability** — a declaration that this plugin *depends on* a
  capability type provided by some other plugin. It carries no handler; it tells
  the platform to wire a provider into this plugin's configured slots.
- **Action** — a user-triggered operation the plugin exposes (optionally with a
  confirmation prompt and a cooldown).

A plugin may also register **event consumers** (`Consumer`) that handle system
or plugin events delivered fire-and-forget.

All of these are recorded on the client and rendered into the manifest at
generation time (see [Configuration generation](#configuration-generation)).

## Transport: RabbitMQ

The default transport is RabbitMQ. The platform injects the broker credentials
and the plugin identity into the container environment, and `griffino.New`
reads them through `ConfigFromEnv`:

```
RABBITMQ_HOST, RABBITMQ_PORT, RABBITMQ_USER, RABBITMQ_PASSWORD
GRIFFINO_PLUGIN_ID, GRIFFINO_USER_ID (optional)
```

Two topic exchanges are used:

- `griffino.router` — invoke/dispatch routing between the platform router and
  plugins.
- `griffino.plugins` — per-plugin delivery.

`Client.Start` dials the broker, declares the exchanges passively, sets QoS,
and starts consuming for the registered providers, event consumers, and
actions. `Client.Close` tears the connection down; both are idempotent.

## Envelopes and routing keys

Every message body is a JSON envelope carrying `userId`, `pluginId`, `msgId`,
the `payload`, and optional `taskId`, `nodeId`, and `context`. The SDK decodes
this into a `griffino.HandlerContext` before calling a handler, validating the
required string fields.

Routing keys are versioned and namespaced by user:

| Operation | Routing key |
| --- | --- |
| Invoke (request) | `invoke.griffino.router.{userId}.{capabilityType}.v1` |
| Dispatch (event) | `dispatch.griffino.router.{userId}.{event}.v1` |
| Blueprint node result | `dispatch.griffino.router.{userId}.node.result.v1` |
| Action | `action.{pluginId}.{userId}.{actionId}.v1` |

The provider entry point published in the manifest uses the pattern
`invoke.{pluginId}.{capabilityId}.v1`.

### Reply matching

A synchronous `Invoke` publishes the request with a `correlationId` and the
direct reply-to queue (`amq.rabbitmq.reply-to`) as `replyTo`, then blocks until
the matching reply arrives or the context/timeout fires. Incoming provider
deliveries take one of two paths:

- **Synchronous invoke** — the delivery carries `replyTo`/`correlationId`; the
  handler's result is published back as the reply.
- **Blueprint task node** — no `replyTo` but a `nodeId` is present; the result
  is dispatched as a node result rather than replied directly.

## Configuration generation

A plugin ships static descriptor files the platform reads at install time. The
SDK generates them from the same metadata and config structs the runtime uses:

- `plugin.manifest.json` — identity, capabilities, actions, permissions, i18n,
  and which configuration files are present.
- `config.boot.json` — admin-set boot configuration (one service block).
- `config.user.json` — per-user configuration (top-level fields).
- `plugin.boot.yml` — the runtime boot spec: the system environment variables
  followed by the plugin's own config keys, plus a build/image scaffold.

Config fields are declared as Go struct fields tagged `griffino:"..."`;
reflection turns them into the field entries in the JSON files. See
[api.md](api.md) for the tag grammar and the generator functions, and the
[deepseek example](../examples/deepseek) for a worked end-to-end plugin.
