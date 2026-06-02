// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

package griffino

import (
	"context"
	"log"
	"time"
)

// Client is the public entry point for building a Griffino plugin. It holds the
// plugin's metadata and configuration, the registries of providers, consumers,
// consumer capabilities, and actions, and the transport over which it talks to
// the platform. Register capabilities, then call Run (or Start/Close) to serve
// them. Use Invoke and Dispatch to call out to other plugins and the platform.
type Client struct {
	metadata  PluginMetadata
	config    Config
	transport Transport
	logger    *log.Logger
	lifecycle *lifecycle

	providers            []ProviderRegistration
	consumers            []ConsumerRegistration
	consumerCapabilities []ConsumerCapabilityRegistration
	actions              []ActionRegistration
}

// clientOptions accumulates the functional options passed to New.
type clientOptions struct {
	metadata     PluginMetadata
	hasMetadata  bool
	config       Config
	hasConfig    bool
	transport    Transport
	hasTransport bool
}

// Option configures a Client during construction.
type Option func(*clientOptions)

// WithMetadata sets the plugin metadata.
func WithMetadata(m PluginMetadata) Option {
	return func(o *clientOptions) {
		o.metadata = m
		o.hasMetadata = true
	}
}

// WithConfig sets the runtime configuration explicitly, bypassing the
// environment and placeholder fallbacks.
func WithConfig(c Config) Option {
	return func(o *clientOptions) {
		o.config = c
		o.hasConfig = true
	}
}

// WithTransport overrides the default RabbitMQ transport, primarily for tests.
func WithTransport(t Transport) Option {
	return func(o *clientOptions) {
		o.transport = t
		o.hasTransport = true
	}
}

// New constructs a Client from the given options.
//
// Configuration resolution: if WithConfig is supplied it is used as-is;
// otherwise New tries ConfigFromEnv and, if that fails and metadata carries an
// ID, falls back to PlaceholderConfig(metadata.ID). The plugin id defaults from
// metadata.ID when the config leaves it empty. If no transport is supplied, a
// RabbitMQ transport is created from the resolved config.
func New(opts ...Option) (*Client, error) {
	o := &clientOptions{}
	for _, opt := range opts {
		opt(o)
	}

	var cfg Config
	switch {
	case o.hasConfig:
		cfg = o.config
	default:
		envCfg, err := ConfigFromEnv()
		if err != nil {
			if !o.hasMetadata || o.metadata.ID == "" {
				return nil, err
			}
			cfg = PlaceholderConfig(o.metadata.ID)
		} else {
			cfg = envCfg
		}
	}
	cfg = cfg.withDefaults()
	if cfg.PluginID == "" {
		cfg.PluginID = o.metadata.ID
	}

	transport := o.transport
	if !o.hasTransport {
		transport = newRabbitMQTransport(cfg)
	}

	return &Client{
		metadata:  o.metadata,
		config:    cfg,
		transport: transport,
		logger:    log.New(log.Writer(), "griffino/client ", log.LstdFlags),
		lifecycle: newLifecycle(),
	}, nil
}

// --- registration ---

// Provider registers a capability this plugin provides.
func (c *Client) Provider(reg ProviderRegistration) {
	c.providers = append(c.providers, reg)
}

// Consumer registers a handler for a system or plugin event.
func (c *Client) Consumer(event string, h Handler) {
	c.consumers = append(c.consumers, ConsumerRegistration{Event: event, Handler: h})
}

// ConsumerCapability declares that this plugin consumes a capability type
// provided by another plugin.
func (c *Client) ConsumerCapability(reg ConsumerCapabilityRegistration) {
	c.consumerCapabilities = append(c.consumerCapabilities, reg)
}

// Action registers a user-triggered action this plugin exposes.
func (c *Client) Action(reg ActionRegistration) {
	c.actions = append(c.actions, reg)
}

// --- getters ---

// Providers returns the registered providers.
func (c *Client) Providers() []ProviderRegistration { return c.providers }

// Consumers returns the registered event consumers.
func (c *Client) Consumers() []ConsumerRegistration { return c.consumers }

// ConsumerCapabilities returns the declared consumer capabilities.
func (c *Client) ConsumerCapabilities() []ConsumerCapabilityRegistration {
	return c.consumerCapabilities
}

// Actions returns the registered actions.
func (c *Client) Actions() []ActionRegistration { return c.actions }

// Config returns the resolved runtime configuration.
func (c *Client) Config() Config { return c.config }

// Metadata returns the plugin metadata.
func (c *Client) Metadata() PluginMetadata { return c.metadata }

// --- invoke / dispatch ---

// invokeOptions accumulates per-call overrides for Invoke.
type invokeOptions struct {
	slot       string
	userID     string
	hasUserID  bool
	taskID     string
	timeout    time.Duration
	hasTimeout bool
}

// InvokeOption overrides a default on a single Invoke call.
type InvokeOption func(*invokeOptions)

// WithSlot selects a named capability slot for the invoke.
func WithSlot(slot string) InvokeOption {
	return func(o *invokeOptions) { o.slot = slot }
}

// WithUserID overrides the user id for the invoke.
func WithUserID(userID string) InvokeOption {
	return func(o *invokeOptions) {
		o.userID = userID
		o.hasUserID = true
	}
}

// WithTaskID sets the task id for the invoke.
func WithTaskID(taskID string) InvokeOption {
	return func(o *invokeOptions) { o.taskID = taskID }
}

// WithTimeout overrides the invoke deadline.
func WithTimeout(d time.Duration) InvokeOption {
	return func(o *invokeOptions) {
		o.timeout = d
		o.hasTimeout = true
	}
}

// Invoke sends a synchronous capability request and returns the provider's
// reply. The user id defaults to the configured UserID and the timeout to the
// configured InvokeTimeout unless overridden.
func (c *Client) Invoke(ctx context.Context, capabilityType string, payload map[string]any, opts ...InvokeOption) (map[string]any, error) {
	o := &invokeOptions{}
	for _, opt := range opts {
		opt(o)
	}

	userID := c.config.UserID
	if o.hasUserID {
		userID = o.userID
	}
	timeout := c.config.InvokeTimeout
	if o.hasTimeout {
		timeout = o.timeout
	}

	req := InvokeRequest{
		CapabilityType: capabilityType,
		Payload:        payload,
		UserID:         userID,
		PluginID:       c.config.PluginID,
		TaskID:         o.taskID,
		Slot:           o.slot,
		TimeoutMS:      int(timeout / time.Millisecond),
	}
	return c.transport.Invoke(ctx, req)
}

// dispatchOptions accumulates per-call overrides for Dispatch.
type dispatchOptions struct {
	userID    string
	hasUserID bool
	taskID    string
}

// DispatchOption overrides a default on a single Dispatch call.
type DispatchOption func(*dispatchOptions)

// WithDispatchUserID overrides the user id for the dispatch.
func WithDispatchUserID(userID string) DispatchOption {
	return func(o *dispatchOptions) {
		o.userID = userID
		o.hasUserID = true
	}
}

// WithDispatchTaskID sets the task id for the dispatch.
func WithDispatchTaskID(taskID string) DispatchOption {
	return func(o *dispatchOptions) { o.taskID = taskID }
}

// Dispatch publishes a fire-and-forget event. The user id defaults to the
// configured UserID unless overridden.
func (c *Client) Dispatch(ctx context.Context, event string, payload map[string]any, opts ...DispatchOption) error {
	o := &dispatchOptions{}
	for _, opt := range opts {
		opt(o)
	}

	userID := c.config.UserID
	if o.hasUserID {
		userID = o.userID
	}

	req := DispatchRequest{
		Event:    event,
		Payload:  payload,
		UserID:   userID,
		PluginID: c.config.PluginID,
		TaskID:   o.taskID,
	}
	return c.transport.Dispatch(ctx, req)
}

// run invokes a handler, recovering any panic and converting it to an error so
// a misbehaving handler cannot crash a consumer goroutine. It is passed to the
// transport as the HandlerRunner.
func (c *Client) run(h Handler, hc *HandlerContext) (result map[string]any, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = nil
			err = capabilityError(nil, "handler panicked: %v", r)
		}
	}()
	return h(hc)
}

// --- lifecycle ---

// Start opens the transport and begins serving the registered providers,
// consumers, and actions. It is safe to call when already started.
func (c *Client) Start(ctx context.Context) error {
	if c.lifecycle.isStarted() && !c.lifecycle.isClosed() {
		return nil
	}
	if err := c.transport.Start(ctx); err != nil {
		return err
	}
	if err := c.transport.StartProviderConsumers(ctx, c.providers, c.run); err != nil {
		return err
	}
	if err := c.transport.StartEventConsumers(ctx, c.consumers, c.run); err != nil {
		return err
	}
	if err := c.transport.StartActionConsumers(ctx, c.actions, c.run); err != nil {
		return err
	}
	c.lifecycle.markStarted()
	return nil
}

// Close tears down the transport. It is idempotent.
func (c *Client) Close(ctx context.Context) error {
	if c.lifecycle.isClosed() {
		return nil
	}
	err := c.transport.Close(ctx)
	c.lifecycle.markClosed()
	return err
}

// WaitClosed blocks until the client is closed.
func (c *Client) WaitClosed() {
	c.lifecycle.waitClosed()
}

// Run starts the client and blocks until it is closed or the context is
// canceled, then closes the transport. It is the typical plugin main loop.
func (c *Client) Run(ctx context.Context) error {
	if err := c.Start(ctx); err != nil {
		return err
	}

	done := make(chan struct{})
	go func() {
		c.WaitClosed()
		close(done)
	}()

	select {
	case <-ctx.Done():
	case <-done:
	}

	return c.Close(context.WithoutCancel(ctx))
}
