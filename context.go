// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

package griffino

// HandlerContext carries the decoded metadata and payload of an incoming
// message into a Handler. TaskID, NodeID, NodeContext, and ActionID are
// populated only when present on the message.
type HandlerContext struct {
	UserID      string
	PluginID    string
	TaskID      string
	MsgID       string
	Payload     map[string]any
	NodeID      string
	NodeContext map[string]any
	ActionID    string
}

// contextFromMessage builds a HandlerContext from a decoded message envelope.
// It mirrors the Python SDK's validation: userId, pluginId (falling back to
// defaultPluginID), and msgId are required strings; taskId, nodeId, and the
// node context are optional. Type mismatches yield an ErrTransport-classified
// error.
func contextFromMessage(msg map[string]any, defaultPluginID string) (*HandlerContext, error) {
	payload, _ := msg["payload"].(map[string]any)
	if payload == nil {
		payload = map[string]any{}
	}

	userID, ok := msg["userId"].(string)
	if !ok {
		return nil, transportError(nil, "message field userId must be a string")
	}

	pluginID, ok := msg["pluginId"].(string)
	if !ok {
		if _, present := msg["pluginId"]; present {
			return nil, transportError(nil, "message field pluginId must be a string")
		}
		pluginID = defaultPluginID
	}
	if pluginID == "" {
		return nil, transportError(nil, "message field pluginId must be a string (or a default plugin id must be provided)")
	}

	msgID, ok := msg["msgId"].(string)
	if !ok {
		return nil, transportError(nil, "message field msgId must be a string")
	}

	taskID, err := optionalString(msg, "taskId")
	if err != nil {
		return nil, err
	}
	nodeID, err := optionalString(msg, "nodeId")
	if err != nil {
		return nil, err
	}

	var nodeContext map[string]any
	if raw, present := msg["context"]; present && raw != nil {
		nodeContext, ok = raw.(map[string]any)
		if !ok {
			return nil, transportError(nil, "message field context must be an object or null")
		}
	}

	return &HandlerContext{
		UserID:      userID,
		PluginID:    pluginID,
		TaskID:      taskID,
		MsgID:       msgID,
		Payload:     payload,
		NodeID:      nodeID,
		NodeContext: nodeContext,
	}, nil
}

// optionalString reads a string field that may be absent or null, rejecting any
// other type.
func optionalString(msg map[string]any, field string) (string, error) {
	raw, present := msg[field]
	if !present || raw == nil {
		return "", nil
	}
	s, ok := raw.(string)
	if !ok {
		return "", transportError(nil, "message field %s must be a string or null", field)
	}
	return s, nil
}
