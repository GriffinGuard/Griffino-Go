# API reference

This is a tour of the public `griffino` package. The package documentation
(`go doc github.com/GriffinGuard/Griffino-Go`) is the authoritative reference;
this page summarizes the surface a plugin author works with.

## Client

`Client` is the entry point. Construct it with `New` and functional options:

```go
client, err := griffino.New(
    griffino.WithMetadata(meta),     // plugin identity and presentation
    // griffino.WithConfig(cfg),     // explicit runtime config (bypasses env)
    // griffino.WithTransport(tr),   // override the RabbitMQ transport (tests)
)
```

Configuration resolution: if `WithConfig` is given it is used as-is; otherwise
`New` reads `ConfigFromEnv` and, if that fails but metadata carries an `ID`,
falls back to `PlaceholderConfig(meta.ID)` (useful when the client exists only
to generate artifacts). If no transport is supplied, a RabbitMQ transport is
built from the resolved config.

### Registration

```go
client.Provider(griffino.ProviderRegistration{
    CapabilityID:         "ai_chat_model",
    CapabilityType:       "ai.chat.model",
    Name:                 griffino.I18n("AI Chat Model"),
    Description:          griffino.I18n("Provides AI chat completion."),
    StandardInterfaceRef: "griffino.interfaces.ai.chat@1.0.0",
    TimeoutMS:            &timeoutMs,
    Handler:              handleChat,
})

client.ConsumerCapability(griffino.ConsumerCapabilityRegistration{...})
client.Action(griffino.ActionRegistration{...})
client.Consumer("task.completed", handleEvent)
```

A `Handler` is `func(ctx *griffino.HandlerContext) (map[string]any, error)`.
`HandlerContext` carries the decoded `UserID`, `PluginID`, `TaskID`, `MsgID`,
`Payload`, and the optional `NodeID`/`NodeContext`/`ActionID`. Provider handlers
return the response payload; consumer and action handlers typically return
`nil`. A panic in a handler is recovered and surfaced as an error.

### Invoke and Dispatch

`Invoke` sends a synchronous capability request and returns the provider's
reply:

```go
reply, err := client.Invoke(ctx, "ai.chat.model", payload,
    griffino.WithSlot("chat"),
    griffino.WithUserID("u-123"),
    griffino.WithTaskID("t-9"),
    griffino.WithTimeout(30*time.Second),
)
```

`Dispatch` publishes a fire-and-forget event:

```go
err := client.Dispatch(ctx, "media.video_link", payload,
    griffino.WithDispatchUserID("u-123"),
    griffino.WithDispatchTaskID("t-9"),
)
```

User id defaults to the configured `UserID` and the invoke timeout to the
configured `InvokeTimeout` unless overridden.

### Lifecycle

```go
client.Start(ctx)   // open transport, begin serving
client.WaitClosed() // block until closed
client.Close(ctx)   // tear down (idempotent)
client.Run(ctx)     // Start, then block until closed or ctx is canceled, then Close
```

`Run` is the typical plugin main loop; wire it to `signal.NotifyContext` for
`SIGINT`/`SIGTERM`.

## Configuration struct tags

Boot and user configuration are declared as struct fields carrying a
`griffino:"..."` tag. The tag is a comma-separated list of `attr=value` pairs:

| Attribute | Meaning |
| --- | --- |
| `key` | (required) the configuration key emitted to `config.*.json`. |
| `type` | `string`, `int`, `float`, `boolean`, `password`, `options`, `multiline_string`; inferred from the Go field kind when omitted. |
| `name` | (required) human-readable field name. |
| `description` | (required) human-readable description. |
| `optional` | bool, default `false`. |
| `default` | default value, converted to the field's type. |
| `group` | UI grouping label. |
| `values` | options for an `options` field: `v1:Display 1\|v2:Display 2`. |
| `validation` | numeric bounds: `min:N;max:M`. |

Because pairs are comma-separated, an attribute value may not contain a comma;
`values` additionally reserves `:` and `|`.

```go
type BootConfig struct {
    APIKey string `griffino:"key=DEEPSEEK_API_KEY,type=password,name=DeepSeek API Key,description=DeepSeek platform API key.,optional=false,group=API Configuration"`
    Tokens int    `griffino:"key=MAX_TOKENS,type=int,name=Max Tokens,description=Maximum tokens.,optional=true,default=4096,validation=min:256;max:8192,group=Model Parameters"`
}
```

`ParseBootConfig(model)` and `ParseUserConfig(model)` reflect over such a struct
and return a `ConfigModelDefinition`.

## Generator functions

The low-level generators build the artifact data structures:

- `GenerateManifest(meta, providers, consumerCaps, actions, hasBoot, hasUser)`
- `GenerateBootConfig(meta, def)` / `GenerateUserConfig(meta, def)`
- `GenerateBootYAMLData(meta, def)`

And the writers persist them:

- `WriteJSONFile(path, data)` — 2-space-indented JSON, backing up any existing
  file first.
- `WriteBootYAMLFile(path, meta, def)` — `plugin.boot.yml` with a build/image
  scaffold.

`SYSTEM_ENVIRONMENT_VARIABLES` holds the fixed broker/cache variables the
platform injects, placed first in the boot YAML environment.

### WriteArtifacts

`WriteArtifacts` is the one-call convenience that ties a `Client` to the
generators:

```go
err := griffino.WriteArtifacts(client, ".", &BootConfig{}, &UserConfig{})
```

It always writes `plugin.manifest.json`; when `bootModel` is non-nil it writes
`config.boot.json` and `plugin.boot.yml`; when `userModel` is non-nil it writes
`config.user.json`. Pass `nil` to skip a config kind. See the
[deepseek example generator](../examples/deepseek/gen/main.go).
