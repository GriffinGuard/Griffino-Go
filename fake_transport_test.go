// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

package griffino

import (
	"context"
	"sync"
)

// fakeTransport is an in-memory Transport for client tests. It records Invoke
// and Dispatch calls, returns a canned invoke response, and retains the
// registrations and HandlerRunner so a test can deliver a message to a handler
// directly.
type fakeTransport struct {
	mu sync.Mutex

	started bool
	closed  bool

	invokes    []InvokeRequest
	dispatches []DispatchRequest

	// invokeResponse is returned from Invoke; invokeErr, if set, takes priority.
	invokeResponse map[string]any
	invokeErr      error

	providers []ProviderRegistration
	consumers []ConsumerRegistration
	actions   []ActionRegistration
	runner    HandlerRunner
}

var _ Transport = (*fakeTransport)(nil)

func (f *fakeTransport) Start(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = true
	return nil
}

func (f *fakeTransport) Close(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

func (f *fakeTransport) Invoke(ctx context.Context, req InvokeRequest) (map[string]any, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.invokes = append(f.invokes, req)
	if f.invokeErr != nil {
		return nil, f.invokeErr
	}
	return f.invokeResponse, nil
}

func (f *fakeTransport) Dispatch(ctx context.Context, req DispatchRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.dispatches = append(f.dispatches, req)
	return nil
}

func (f *fakeTransport) StartProviderConsumers(ctx context.Context, regs []ProviderRegistration, run HandlerRunner) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.providers = regs
	f.runner = run
	return nil
}

func (f *fakeTransport) StartEventConsumers(ctx context.Context, regs []ConsumerRegistration, run HandlerRunner) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.consumers = regs
	f.runner = run
	return nil
}

func (f *fakeTransport) StartActionConsumers(ctx context.Context, regs []ActionRegistration, run HandlerRunner) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.actions = regs
	f.runner = run
	return nil
}

// deliverToProvider runs the provider registered for capabilityID through the
// stored HandlerRunner.
func (f *fakeTransport) deliverToProvider(capabilityID string, hc *HandlerContext) (map[string]any, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, reg := range f.providers {
		if reg.CapabilityID == capabilityID {
			return f.runner(reg.Handler, hc)
		}
	}
	return nil, nil
}

// deliverToConsumer runs the consumer registered for event through the runner.
func (f *fakeTransport) deliverToConsumer(event string, hc *HandlerContext) (map[string]any, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, reg := range f.consumers {
		if reg.Event == event {
			return f.runner(reg.Handler, hc)
		}
	}
	return nil, nil
}

// deliverToAction runs the action registered for actionID through the runner.
func (f *fakeTransport) deliverToAction(actionID string, hc *HandlerContext) (map[string]any, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, reg := range f.actions {
		if reg.ActionID == actionID {
			return f.runner(reg.Handler, hc)
		}
	}
	return nil, nil
}
