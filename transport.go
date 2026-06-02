// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

package griffino

import "context"

// InvokeRequest is an outbound synchronous capability invocation.
type InvokeRequest struct {
	CapabilityType string
	Payload        map[string]any
	UserID         string
	PluginID       string
	TaskID         string
	Slot           string
	TimeoutMS      int
}

// DispatchRequest is an outbound fire-and-forget event dispatch.
type DispatchRequest struct {
	Event    string
	Payload  map[string]any
	UserID   string
	PluginID string
	TaskID   string
}

// HandlerRunner invokes a registered handler with a decoded context. The client
// supplies one to the transport so cross-cutting concerns (panic recovery,
// logging, context propagation) stay in the client layer while the transport
// owns only message plumbing.
type HandlerRunner func(h Handler, hc *HandlerContext) (map[string]any, error)

// Transport abstracts the message bus the client speaks to. The default
// implementation is RabbitMQ; tests substitute an in-memory fake. All methods
// honor the provided context for cancellation.
type Transport interface {
	// Start establishes the connection and any shared resources. It is
	// idempotent: calling it more than once is a no-op once started.
	Start(ctx context.Context) error
	// Close tears down consumers and the connection.
	Close(ctx context.Context) error

	// Invoke sends a synchronous capability request and waits for the reply.
	Invoke(ctx context.Context, req InvokeRequest) (map[string]any, error)
	// Dispatch publishes a fire-and-forget event.
	Dispatch(ctx context.Context, req DispatchRequest) error

	// StartProviderConsumers begins consuming requests for each provider.
	StartProviderConsumers(ctx context.Context, regs []ProviderRegistration, run HandlerRunner) error
	// StartEventConsumers begins consuming events for each consumer.
	StartEventConsumers(ctx context.Context, regs []ConsumerRegistration, run HandlerRunner) error
	// StartActionConsumers begins consuming user-triggered actions.
	StartActionConsumers(ctx context.Context, regs []ActionRegistration, run HandlerRunner) error
}
