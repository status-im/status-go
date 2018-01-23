// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package openconfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openconfig/reference/rpc/openconfig"
)

// joinPath builds a string out of an Element
func joinPath(path *openconfig.Path) string {
	if path == nil {
		return ""
	}
	return strings.Join(path.Element, "/")
}

func convertUpdate(update *openconfig.Update) (interface{}, error) {
	switch update.Value.Type {
	case openconfig.Type_JSON:
		var value interface{}
		decoder := json.NewDecoder(bytes.NewReader(update.Value.Value))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			return nil, fmt.Errorf("Malformed JSON update %q in %s",
				update.Value.Value, update)
		}
		return value, nil
	case openconfig.Type_BYTES:
		return update.Value.Value, nil
	default:
		return nil,
			fmt.Errorf("Unhandled type of value %v in %s", update.Value.Type, update)
	}
}

// NotificationToJSON converts a Notification into a JSON string
func NotificationToJSON(notif *openconfig.Notification) (string, error) {
	m := make(map[string]interface{}, 1)
	m["timestamp"] = notif.Timestamp
	m["path"] = "/" + joinPath(notif.Prefix)
	if len(notif.Update) != 0 {
		updates := make(map[string]interface{}, len(notif.Update))
		var err error
		for _, update := range notif.Update {
			updates[joinPath(update.Path)], err = convertUpdate(update)
			if err != nil {
				return "", err
			}
		}
		m["updates"] = updates
	}
	if len(notif.Delete) != 0 {
		deletes := make([]string, len(notif.Delete))
		for i, del := range notif.Delete {
			deletes[i] = joinPath(del)
		}
		m["deletes"] = deletes
	}
	m = map[string]interface{}{"notification": m}
	js, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}
	return string(js), nil
}

// SubscribeResponseToJSON converts a SubscribeResponse into a JSON string
func SubscribeResponseToJSON(resp *openconfig.SubscribeResponse) (string, error) {
	m := make(map[string]interface{}, 1)
	var err error
	switch resp := resp.Response.(type) {
	case *openconfig.SubscribeResponse_Update:
		return NotificationToJSON(resp.Update)
	case *openconfig.SubscribeResponse_Heartbeat:
		m["heartbeat"] = resp.Heartbeat.Interval
	case *openconfig.SubscribeResponse_SyncResponse:
		m["syncResponse"] = resp.SyncResponse
	default:
		return "", fmt.Errorf("Unknown type of response: %T: %s", resp, resp)
	}
	js, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}
	return string(js), nil
}

// EscapeFunc is the escaping method for attribute names
type EscapeFunc func(k string) string

// escapeValue looks for maps in an interface and escapes their keys
func escapeValue(value interface{}, escape EscapeFunc) interface{} {
	valueMap, ok := value.(map[string]interface{})
	if !ok {
		return value
	}
	escapedMap := make(map[string]interface{}, len(valueMap))
	for k, v := range valueMap {
		escapedKey := escape(k)
		escapedMap[escapedKey] = escapeValue(v, escape)
	}
	return escapedMap
}

// addPathToMap creates a map[string]interface{} from a path. It returns the node in
// the map corresponding to the last element in the path
func addPathToMap(root map[string]interface{}, path []string, escape EscapeFunc) (
	map[string]interface{}, error) {
	parent := root
	for _, element := range path {
		k := escape(element)
		node, found := parent[k]
		if !found {
			node = map[string]interface{}{}
			parent[k] = node
		}
		var ok bool
		parent, ok = node.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf(
				"Node %s is of type %T (expected map[string]interface traversing %q)",
				element, node, path)
		}
	}
	return parent, nil
}

// NotificationToMap maps a Notification into a nested map of entities
func NotificationToMap(addr string, notification *openconfig.Notification,
	escape EscapeFunc) (map[string]interface{}, error) {
	if escape == nil {
		escape = func(name string) string {
			return name
		}
	}
	prefix := notification.GetPrefix()

	// Convert deletes
	var deletes map[string]interface{}
	notificationDeletes := notification.GetDelete()
	if notificationDeletes != nil {
		deletes = make(map[string]interface{})
		node := deletes
		if prefix != nil {
			var err error
			node, err = addPathToMap(node, prefix.Element, escape)
			if err != nil {
				return nil, err
			}
		}
		for _, delete := range notificationDeletes {
			_, err := addPathToMap(node, delete.Element, escape)
			if err != nil {
				return nil, err
			}
		}
	}

	// Convert updates
	var updates map[string]interface{}
	notificationUpdates := notification.GetUpdate()
	if notificationUpdates != nil {
		updates = make(map[string]interface{})
		node := updates
		if prefix != nil {
			var err error
			node, err = addPathToMap(node, prefix.Element, escape)
			if err != nil {
				return nil, err
			}
		}
		for _, update := range notificationUpdates {
			updateNode := node
			path := update.GetPath()
			elementLen := len(path.Element)

			// Convert all elements before the leaf
			if elementLen > 1 {
				parentElements := path.Element[:elementLen-1]
				var err error
				updateNode, err = addPathToMap(updateNode, parentElements, escape)
				if err != nil {
					return nil, err
				}
			}

			// Convert the value in the leaf
			value := update.GetValue()
			var unmarshaledValue interface{}
			switch value.Type {
			case openconfig.Type_JSON:
				if err := json.Unmarshal(value.Value, &unmarshaledValue); err != nil {
					return nil, err
				}
			case openconfig.Type_BYTES:
				unmarshaledValue = update.Value.Value
			default:
				return nil, fmt.Errorf("Unexpected value type %s for path %v",
					value.Type, path)
			}
			updateNode[escape(path.Element[elementLen-1])] = escapeValue(
				unmarshaledValue, escape)
		}
	}

	// Build the complete map to return
	root := map[string]interface{}{
		"timestamp": notification.Timestamp,
	}
	if addr != "" {
		root["dataset"] = addr
	}
	if deletes != nil {
		root["delete"] = deletes
	}
	if updates != nil {
		root["update"] = updates
	}
	return root, nil
}

// NotificationToJSONDocument maps a Notification into a single JSON document
func NotificationToJSONDocument(addr string, notification *openconfig.Notification,
	escape EscapeFunc) ([]byte, error) {
	m, err := NotificationToMap(addr, notification, escape)
	if err != nil {
		return nil, err
	}
	return json.Marshal(m)
}
