// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

package griffino

import (
	"context"
	"errors"
	"testing"
	"time"
)

func testClient(t *testing.T, ft *fakeTransport) *Client {
	t.Helper()
	c, err := New(
		WithMetadata(PluginMetadata{ID: "test.plugin"}),
		WithConfig(Config{PluginID: "test.plugin", UserID: "user-1", InvokeTimeout: 5 * time.Second}),
		WithTransport(ft),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestNewDefaultsPluginIDFromMetadata(t *testing.T) {
	ft := &fakeTransport{}
	c, err := New(
		WithMetadata(PluginMetadata{ID: "meta.id"}),
		WithConfig(Config{}),
		WithTransport(ft),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := c.Config().PluginID; got != "meta.id" {
		t.Fatalf("PluginID = %q, want meta.id", got)
	}
}

func TestInvokeCapturesRoutingFields(t *testing.T) {
	ft := &fakeTransport{invokeResponse: map[string]any{"ok": true}}
	c := testClient(t, ft)

	resp, err := c.Invoke(context.Background(), "llm.chat", map[string]any{"q": "hi"},
		WithSlot("primary"), WithTaskID("task-9"))
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if resp["ok"] != true {
		t.Fatalf("response = %v", resp)
	}
	if len(ft.invokes) != 1 {
		t.Fatalf("expected 1 invoke, got %d", len(ft.invokes))
	}
	req := ft.invokes[0]
	if req.CapabilityType != "llm.chat" || req.UserID != "user-1" || req.PluginID != "test.plugin" {
		t.Fatalf("unexpected request fields: %+v", req)
	}
	if req.Slot != "primary" || req.TaskID != "task-9" {
		t.Fatalf("slot/task not captured: %+v", req)
	}
	if req.TimeoutMS != 5000 {
		t.Fatalf("TimeoutMS = %d, want 5000", req.TimeoutMS)
	}
}

func TestInvokeOptionOverrides(t *testing.T) {
	ft := &fakeTransport{}
	c := testClient(t, ft)

	_, err := c.Invoke(context.Background(), "cap", nil,
		WithUserID("other-user"), WithTimeout(2*time.Second))
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	req := ft.invokes[0]
	if req.UserID != "other-user" {
		t.Fatalf("UserID = %q, want other-user", req.UserID)
	}
	if req.TimeoutMS != 2000 {
		t.Fatalf("TimeoutMS = %d, want 2000", req.TimeoutMS)
	}
}

func TestDispatchCapturesFields(t *testing.T) {
	ft := &fakeTransport{}
	c := testClient(t, ft)

	if err := c.Dispatch(context.Background(), "evt.fired", map[string]any{"x": 1},
		WithDispatchTaskID("t-7")); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if len(ft.dispatches) != 1 {
		t.Fatalf("expected 1 dispatch, got %d", len(ft.dispatches))
	}
	d := ft.dispatches[0]
	if d.Event != "evt.fired" || d.UserID != "user-1" || d.TaskID != "t-7" {
		t.Fatalf("unexpected dispatch fields: %+v", d)
	}
}

func TestRegistrationAndDelivery(t *testing.T) {
	ft := &fakeTransport{}
	c := testClient(t, ft)

	var providerSawNode, consumerCalled bool
	var actionSawID string

	c.Provider(ProviderRegistration{
		CapabilityID:   "cap-1",
		CapabilityType: "cap.type",
		Handler: func(hc *HandlerContext) (map[string]any, error) {
			providerSawNode = hc.NodeID == "node-x"
			return map[string]any{"echo": hc.Payload["in"]}, nil
		},
	})
	c.Consumer("evt", func(hc *HandlerContext) (map[string]any, error) {
		consumerCalled = true
		return nil, nil
	})
	c.Action(ActionRegistration{
		ActionID: "act-1",
		Handler: func(hc *HandlerContext) (map[string]any, error) {
			actionSawID = hc.ActionID
			return nil, nil
		},
	})

	if len(c.Providers()) != 1 || len(c.Consumers()) != 1 || len(c.Actions()) != 1 {
		t.Fatalf("registries not populated: %d/%d/%d", len(c.Providers()), len(c.Consumers()), len(c.Actions()))
	}

	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	resp, err := ft.deliverToProvider("cap-1", &HandlerContext{NodeID: "node-x", Payload: map[string]any{"in": "v"}})
	if err != nil {
		t.Fatalf("provider delivery: %v", err)
	}
	if resp["echo"] != "v" || !providerSawNode {
		t.Fatalf("provider handler did not run as expected: %v", resp)
	}

	if _, err := ft.deliverToConsumer("evt", &HandlerContext{}); err != nil {
		t.Fatalf("consumer delivery: %v", err)
	}
	if !consumerCalled {
		t.Fatal("consumer handler not called")
	}

	if _, err := ft.deliverToAction("act-1", &HandlerContext{ActionID: "act-1"}); err != nil {
		t.Fatalf("action delivery: %v", err)
	}
	if actionSawID != "act-1" {
		t.Fatalf("action id = %q, want act-1", actionSawID)
	}
}

func TestRunRecoversPanic(t *testing.T) {
	ft := &fakeTransport{}
	c := testClient(t, ft)

	result, err := c.run(func(hc *HandlerContext) (map[string]any, error) {
		panic("boom")
	}, &HandlerContext{})

	if result != nil {
		t.Fatalf("expected nil result, got %v", result)
	}
	if err == nil {
		t.Fatal("expected error from panic recovery")
	}
	if !errors.Is(err, ErrCapability) {
		t.Fatalf("error not classified under ErrCapability: %v", err)
	}
}

func TestRunPassesThroughError(t *testing.T) {
	ft := &fakeTransport{}
	c := testClient(t, ft)
	sentinel := errors.New("handler failure")

	_, err := c.run(func(hc *HandlerContext) (map[string]any, error) {
		return nil, sentinel
	}, &HandlerContext{})
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want wrapped sentinel", err)
	}
}

func TestLifecycle(t *testing.T) {
	ft := &fakeTransport{}
	c := testClient(t, ft)

	if c.lifecycle.isStarted() {
		t.Fatal("should not be started yet")
	}
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !c.lifecycle.isStarted() {
		t.Fatal("should be started")
	}
	if !ft.started {
		t.Fatal("transport not started")
	}

	closedCh := make(chan struct{})
	go func() {
		c.WaitClosed()
		close(closedCh)
	}()

	if err := c.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !ft.closed {
		t.Fatal("transport not closed")
	}

	select {
	case <-closedCh:
	case <-time.After(time.Second):
		t.Fatal("WaitClosed did not unblock")
	}

	// Close is idempotent.
	if err := c.Close(context.Background()); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestRunStopsOnContextCancel(t *testing.T) {
	ft := &fakeTransport{}
	c := testClient(t, ft)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- c.Run(ctx) }()

	// Give Run a moment to start, then cancel.
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after context cancel")
	}
	if !ft.closed {
		t.Fatal("transport not closed after Run returned")
	}
}
