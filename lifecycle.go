// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

package griffino

import "sync"

// lifecycle tracks a client's started/closed state. It is safe for concurrent
// use. waitClosed blocks until markClosed is called, mirroring the Python SDK's
// ClientLifecycle backed by an asyncio.Event.
type lifecycle struct {
	mu       sync.Mutex
	started  bool
	closed   bool
	closedCh chan struct{}
}

// newLifecycle returns a lifecycle with a fresh, open closed channel.
func newLifecycle() *lifecycle {
	return &lifecycle{closedCh: make(chan struct{})}
}

// markStarted records that the client has started. If the lifecycle was
// previously closed it is reset so it can be started again with a fresh closed
// channel.
func (l *lifecycle) markStarted() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.started = true
	if l.closed {
		l.closed = false
		l.closedCh = make(chan struct{})
	}
}

// markClosed records that the client has closed and unblocks waitClosed. It is
// idempotent: closing an already-closed lifecycle is a no-op.
func (l *lifecycle) markClosed() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return
	}
	l.closed = true
	close(l.closedCh)
}

// waitClosed blocks until the lifecycle is marked closed.
func (l *lifecycle) waitClosed() {
	l.mu.Lock()
	ch := l.closedCh
	l.mu.Unlock()
	<-ch
}

// isStarted reports whether the client has started.
func (l *lifecycle) isStarted() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.started
}

// isClosed reports whether the client has closed.
func (l *lifecycle) isClosed() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.closed
}
