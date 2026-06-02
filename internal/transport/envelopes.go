// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

package transport

// The envelope builders below produce the exact camelCase JSON shapes the
// platform expects on the wire. Optional fields are omitted (not set to null)
// unless the platform requires the key, matching the Python SDK byte-for-byte.

// BuildInvokeEnvelope builds the body of a synchronous capability invocation.
// The slot key is included only when non-empty.
func BuildInvokeEnvelope(taskID, msgID, userID, pluginID string, payload map[string]any, slot string) map[string]any {
	env := map[string]any{
		"taskId":   nullableString(taskID),
		"msgId":    msgID,
		"userId":   userID,
		"pluginId": pluginID,
		"payload":  payload,
	}
	if slot != "" {
		env["slot"] = slot
	}
	return env
}

// BuildDispatchEnvelope builds the body of a fire-and-forget event dispatch.
func BuildDispatchEnvelope(taskID, msgID, userID, pluginID, event string, payload map[string]any) map[string]any {
	return map[string]any{
		"taskId":   nullableString(taskID),
		"msgId":    msgID,
		"userId":   userID,
		"pluginId": pluginID,
		"event":    event,
		"payload":  payload,
	}
}

// BuildNodeResultEnvelope builds the result a provider emits when invoked as a
// blueprint task node. failReason is included only when non-empty.
func BuildNodeResultEnvelope(taskID, msgID, userID, pluginID, nodeID string, ok bool, output map[string]any, failReason string) map[string]any {
	env := map[string]any{
		"taskId":   nullableString(taskID),
		"msgId":    msgID,
		"userId":   userID,
		"pluginId": pluginID,
		"nodeId":   nodeID,
		"ok":       ok,
		"output":   output,
	}
	if failReason != "" {
		env["failReason"] = failReason
	}
	return env
}

// BuildProviderError builds the body returned when a provider handler fails.
func BuildProviderError(message string) map[string]any {
	return map[string]any{"error": message}
}

// nullableString maps an empty string to a JSON null, since the platform models
// absent task ids as null rather than "".
func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
