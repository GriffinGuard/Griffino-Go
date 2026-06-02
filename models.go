// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

package griffino

// Handler processes an incoming message and optionally returns a JSON object as
// its result. Provider handlers return the response payload; consumer and
// action handlers typically return nil. A non-nil error is logged and, for
// synchronous provider invocations, surfaced to the caller as a provider error.
type Handler func(ctx *HandlerContext) (map[string]any, error)

// I18nString is an internationalizable string. It serializes to
// {"default": ...} and, when Key is set, additionally emits "_i18n_key".
type I18nString struct {
	Default string
	Key     string
}

// I18n constructs an I18nString. Pass an optional i18n key as the second
// argument: I18n("Hello") or I18n("Hello", "greeting.hello").
func I18n(def string, key ...string) I18nString {
	s := I18nString{Default: def}
	if len(key) > 0 {
		s.Key = key[0]
	}
	return s
}

// toManifest renders the i18n value for a manifest file. An empty value yields
// the backward-compatible {"default": ""} form.
func (s I18nString) toManifest() map[string]string {
	m := map[string]string{"default": s.Default}
	if s.Key != "" {
		m["_i18n_key"] = s.Key
	}
	return m
}

// CapabilitySlot is a named slot shared by provider and consumer capabilities.
type CapabilitySlot struct {
	ID          string
	Name        I18nString
	Description I18nString
}

// ProviderRegistration describes a capability this plugin provides.
type ProviderRegistration struct {
	CapabilityID         string
	CapabilityType       string
	Handler              Handler
	Name                 I18nString
	Description          I18nString
	StandardInterfaceRef string
	InputSchemaRef       string
	OutputSchemaRef      string
	// TimeoutMS is the capability's default timeout in milliseconds. Nil means
	// unset (the manifest omits defaultTimeoutMilliseconds).
	TimeoutMS *int
	Slots     []CapabilitySlot
}

// ConsumerRegistration describes a system or plugin event this plugin listens for.
type ConsumerRegistration struct {
	Event   string
	Handler Handler
}

// ConsumerCapabilityRegistration declares that this plugin consumes a capability
// type provided by some other plugin, optionally exposing named slots.
type ConsumerCapabilityRegistration struct {
	CapabilityID   string
	CapabilityType string
	Name           I18nString
	Description    I18nString
	Optional       bool
	Slots          []CapabilitySlot
}

// ActionRegistration describes a user-triggered action this plugin exposes.
type ActionRegistration struct {
	ActionID             string
	Handler              Handler
	Name                 I18nString
	Description          I18nString
	ConfirmationRequired bool
	// ConfirmationMessage is shown when confirmation is required. Nil omits it.
	ConfirmationMessage *I18nString
	// CooldownMS is the minimum interval between triggers in milliseconds. Nil
	// means unset (the manifest omits cooldownMs).
	CooldownMS *int
}

// PluginPermission is a permission the plugin requests from the platform.
type PluginPermission struct {
	Name        string
	Description I18nString
	Optional    bool
}

// PluginInternationalization declares the plugin's localization catalogs.
type PluginInternationalization struct {
	DefaultLanguage    string
	SupportedLanguages []string
	LocalizationFiles  map[string]string
}

// PluginMetadata is the static identity and presentation data for a plugin,
// used to render plugin.manifest.json and the config files.
type PluginMetadata struct {
	ID                   string
	Name                 I18nString
	Version              string
	Description          I18nString
	MainServiceID        string
	Author               string
	License              string
	Site                 string
	MainSystemVersion    string
	RequiredPlugins      []string
	PermissionsRequested []PluginPermission
	Tutorial             string
	Internationalization *PluginInternationalization
	ManifestSchema       string
	BootConfigSchema     string
	UserConfigSchema     string
}
