// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

package griffino

import (
	"errors"
	"fmt"
)

// Error sentinels form the SDK error hierarchy. They mirror the Python SDK's
// exception classes and chain through errors.Is:
//
//	ErrGriffino                       (base)
//	├── ErrConfiguration
//	├── ErrTransport
//	└── ErrCapability
//	    ├── ErrSlotNotConfigured
//	    └── ErrTimeout
//
// Every error returned by this package satisfies errors.Is against ErrGriffino
// and against the most specific sentinel that applies, so callers can branch
// with errors.Is(err, griffino.ErrTimeout), errors.Is(err, griffino.ErrCapability),
// and so on.
var (
	// ErrGriffino is the root of the SDK error hierarchy.
	ErrGriffino = errors.New("griffino: error")
	// ErrConfiguration indicates invalid or incomplete SDK configuration.
	ErrConfiguration = fmt.Errorf("griffino: configuration error: %w", ErrGriffino)
	// ErrTransport indicates a failure in the underlying message transport.
	ErrTransport = fmt.Errorf("griffino: transport error: %w", ErrGriffino)
	// ErrCapability indicates a capability invocation failed.
	ErrCapability = fmt.Errorf("griffino: capability error: %w", ErrGriffino)
	// ErrSlotNotConfigured indicates a requested capability slot is not configured.
	ErrSlotNotConfigured = fmt.Errorf("griffino: slot not configured: %w", ErrCapability)
	// ErrTimeout indicates an invoke call exceeded its deadline.
	ErrTimeout = fmt.Errorf("griffino: invoke timed out: %w", ErrCapability)
)

// sdkError is a concrete error carrying a human-readable message, a sentinel
// "kind" used for errors.Is classification, and an optional wrapped cause.
type sdkError struct {
	kind  error
	cause error
	msg   string
}

func (e *sdkError) Error() string {
	if e.cause != nil {
		return e.msg + ": " + e.cause.Error()
	}
	return e.msg
}

// Unwrap returns both the sentinel kind and the wrapped cause so errors.Is and
// errors.As traverse the full hierarchy as well as any underlying error.
func (e *sdkError) Unwrap() []error {
	if e.cause != nil {
		return []error{e.kind, e.cause}
	}
	return []error{e.kind}
}

// newError builds an SDK error classified under kind, wrapping cause (which may
// be nil), with a message formatted from format and args.
func newError(kind, cause error, format string, args ...any) error {
	return &sdkError{kind: kind, cause: cause, msg: fmt.Sprintf(format, args...)}
}

// configError builds an error classified under ErrConfiguration.
func configError(format string, args ...any) error {
	return newError(ErrConfiguration, nil, format, args...)
}

// transportError builds an error classified under ErrTransport, wrapping cause.
func transportError(cause error, format string, args ...any) error {
	return newError(ErrTransport, cause, format, args...)
}

// capabilityError builds an error classified under ErrCapability, wrapping cause.
func capabilityError(cause error, format string, args ...any) error {
	return newError(ErrCapability, cause, format, args...)
}

// timeoutError builds an error classified under ErrTimeout, wrapping cause.
func timeoutError(cause error, format string, args ...any) error {
	return newError(ErrTimeout, cause, format, args...)
}

// slotNotConfiguredError builds an error classified under ErrSlotNotConfigured.
func slotNotConfiguredError(format string, args ...any) error {
	return newError(ErrSlotNotConfigured, nil, format, args...)
}
