// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

// Package transport holds wire-level helpers — JSON codec and message envelope
// builders — shared by the SDK's transport implementation. It depends on no
// other SDK package so it can be reused without import cycles.
package transport

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
)

// EncodeJSON serializes v as compact UTF-8 JSON without HTML escaping, matching
// the Python SDK's json.dumps(separators=(",", ":"), ensure_ascii=False).
func EncodeJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	// Encoder appends a trailing newline; trim it for wire-format parity.
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// DecodeJSON parses a JSON object from body. It is an error for the body to
// decode to anything other than an object.
func DecodeJSON(body []byte) (map[string]any, error) {
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return nil, err
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("message body must decode to an object")
	}
	return obj, nil
}

// NewMessageID returns a random RFC 4122 version 4 UUID string, used as the
// message/correlation id.
func NewMessageID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failure is not recoverable in any meaningful way here.
		panic(fmt.Sprintf("griffino: failed to read random bytes: %v", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
