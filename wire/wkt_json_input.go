package wire

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// applyWKTJSONInput converts JSON-native forms for well-known types into their
// protobuf message map shapes prior to nested message encoding.
func (me *MessageEncoder) applyWKTJSONInput(messageTypeName string, value interface{}) (interface{}, error) {
	// String inputs for Timestamp/Duration/FieldMask
	if s, ok := value.(string); ok {
		switch messageTypeName {
		case "google.protobuf.Timestamp":
			sec, ns, perr := parseRFC3339ToSecondsNanos(s)
			if perr != nil {
				return nil, perr
			}
			return map[string]interface{}{"seconds": sec, "nanos": ns}, nil
		case "google.protobuf.Duration":
			dsec, dns, perr := parseDurationString(s)
			if perr != nil {
				return nil, perr
			}
			return map[string]interface{}{"seconds": dsec, "nanos": dns}, nil
		case "google.protobuf.FieldMask":
			paths, perr := parseFieldMaskString(s)
			if perr != nil {
				return nil, perr
			}
			return map[string]interface{}{"paths": paths}, nil
		}
	}

	// Map/array inputs for Any/Struct/Value/ListValue
	switch messageTypeName {
	case "google.protobuf.Any":
		if m, ok := value.(map[string]interface{}); ok {
			return me.normalizeAnyInput(m)
		}
	case "google.protobuf.Struct":
		if m, ok := value.(map[string]interface{}); ok {
			// If already in message shape {fields: ...}, accept as-is
			if _, isShaped := m["fields"]; isShaped {
				return m, nil
			}
			fields := make(map[string]interface{}, len(m))
			for k, v := range m {
				fields[k] = me.jsonToValueMessage(v)
			}
			return map[string]interface{}{"fields": fields}, nil
		}
	case "google.protobuf.Value":
		// If already a Value-shaped map, accept as-is
		if m, ok := value.(map[string]interface{}); ok && isValueMessageMap(m) {
			return m, nil
		}
		return me.jsonToValueMessage(value), nil
	case "google.protobuf.ListValue":
		if arr, ok := value.([]interface{}); ok {
			out := make([]interface{}, len(arr))
			for i := range arr {
				out[i] = me.jsonToValueMessage(arr[i])
			}
			return map[string]interface{}{"values": out}, nil
		}
		// If already in message shape {values: [...]}, accept as-is
		if m, ok := value.(map[string]interface{}); ok {
			if _, isShaped := m["values"]; isShaped {
				return m, nil
			}
		}
	}
	return value, nil
}

// normalizeAnyInput converts an Any JSON object of the form {"@type": typeUrl, ...payload...}
// or {"type_url":..., "value": base64} into the message map {type_url:string, value:[]byte}.
func (me *MessageEncoder) normalizeAnyInput(m map[string]interface{}) (map[string]interface{}, error) {
	var typeURL string
	if at, ok := m["@type"].(string); ok && at != "" {
		typeURL = at
	} else if tu, ok := m["type_url"].(string); ok && tu != "" {
		typeURL = tu
	}
	if typeURL == "" {
		return nil, fmt.Errorf("Any missing @type/type_url")
	}
	if !strings.Contains(typeURL, "/") {
		typeURL = "type.googleapis.com/" + typeURL
	}
	if raw, ok := m["value"]; ok {
		switch rv := raw.(type) {
		case string:
			b, err := base64.StdEncoding.DecodeString(rv)
			if err != nil {
				return nil, fmt.Errorf("Any.value not base64: %w", err)
			}
			return map[string]interface{}{"type_url": typeURL, "value": b}, nil
		case []byte:
			return map[string]interface{}{"type_url": typeURL, "value": rv}, nil
		case map[string]interface{}:
			b, err := me.packAnyPayload(rv, typeURL)
			if err != nil {
				return nil, err
			}
			return map[string]interface{}{"type_url": typeURL, "value": b}, nil
		default:
			return nil, fmt.Errorf("unsupported Any.value type %T", raw)
		}
	}
	payload := make(map[string]interface{})
	for k, v := range m {
		if k == "@type" || k == "type_url" {
			continue
		}
		payload[k] = v
	}
	b, err := me.packAnyPayload(payload, typeURL)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"type_url": typeURL, "value": b}, nil
}

func (me *MessageEncoder) packAnyPayload(payload map[string]interface{}, typeURL string) ([]byte, error) {
	i := strings.LastIndex(typeURL, "/")
	typeName := typeURL
	if i >= 0 && i+1 < len(typeURL) {
		typeName = typeURL[i+1:]
	}
	msg, err := me.encoder.registry.GetMessage(typeName)
	if err != nil {
		return nil, fmt.Errorf("unknown Any type %s", typeName)
	}
	enc := NewEncoder()
	enc.registry = me.encoder.registry
	if err := NewMessageEncoder(enc).EncodeMessage(payload, msg); err != nil {
		return nil, err
	}
	return enc.Bytes(), nil
}

// isValueMessageMap returns true if m appears to already be a google.protobuf.Value message map.
func isValueMessageMap(m map[string]interface{}) bool {
	if _, ok := m["null_value"]; ok {
		return true
	}
	if _, ok := m["number_value"]; ok {
		return true
	}
	if _, ok := m["string_value"]; ok {
		return true
	}
	if _, ok := m["bool_value"]; ok {
		return true
	}
	if sv, ok := m["struct_value"].(map[string]interface{}); ok {
		if _, ok := sv["fields"].(map[string]interface{}); ok {
			return true
		}
	}
	if lv, ok := m["list_value"].(map[string]interface{}); ok {
		if _, ok := lv["values"].([]interface{}); ok {
			return true
		}
	}
	return false
}

// jsonToValueMessage converts a plain JSON value into a google.protobuf.Value message map
func (me *MessageEncoder) jsonToValueMessage(v interface{}) map[string]interface{} {
	// If it already looks like a Value-shaped map, return as-is to avoid re-wrapping
	if m, ok := v.(map[string]interface{}); ok && isValueMessageMap(m) {
		return m
	}
	switch t := v.(type) {
	case nil:
		return map[string]interface{}{"null_value": "NULL_VALUE"}
	case bool:
		return map[string]interface{}{"bool_value": t}
	case string:
		return map[string]interface{}{"string_value": t}
	case json.Number:
		f, _ := strconv.ParseFloat(t.String(), 64)
		return map[string]interface{}{"number_value": f}
	case float64:
		return map[string]interface{}{"number_value": t}
	case float32:
		return map[string]interface{}{"number_value": float64(t)}
	case int64:
		return map[string]interface{}{"number_value": float64(t)}
	case int32:
		return map[string]interface{}{"number_value": float64(t)}
	case uint64:
		return map[string]interface{}{"number_value": float64(t)}
	case uint32:
		return map[string]interface{}{"number_value": float64(t)}
	case map[string]interface{}:
		fields := make(map[string]interface{}, len(t))
		for k, vv := range t {
			fields[k] = me.jsonToValueMessage(vv)
		}
		return map[string]interface{}{"struct_value": map[string]interface{}{"fields": fields}}
	case []interface{}:
		arr := make([]interface{}, len(t))
		for i := range t {
			arr[i] = me.jsonToValueMessage(t[i])
		}
		return map[string]interface{}{"list_value": map[string]interface{}{"values": arr}}
	default:
		b, _ := json.Marshal(t)
		return map[string]interface{}{"string_value": string(b)}
	}
}

func parseFieldMaskString(s string) ([]string, error) {
	if strings.TrimSpace(s) == "" {
		return []string{}, nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, camelToSnake(p))
	}
	return out, nil
}

// camelToSnake converts lowerCamelCase to snake_case
func camelToSnake(s string) string {
	if s == "" {
		return s
	}
	out := make([]byte, 0, len(s)+4)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			if i != 0 {
				out = append(out, '_')
			}
			c = c - 'A' + 'a'
		}
		out = append(out, c)
	}
	return string(out)
}

// parseRFC3339ToSecondsNanos parses RFC3339(ish) timestamp into seconds and nanos.
func parseRFC3339ToSecondsNanos(ts string) (int64, int32, error) {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid timestamp: %w", err)
	}
	sec := t.Unix()
	ns := int32(t.Nanosecond())
	if sec < -62135596800 || sec > 253402300799 {
		return 0, 0, fmt.Errorf("timestamp out of range")
	}
	if ns < 0 || ns > 999999999 {
		return 0, 0, fmt.Errorf("timestamp nanos out of range")
	}
	return sec, ns, nil
}

// parseDurationString parses protobuf JSON duration (e.g. "1.010000001s")
func parseDurationString(ds string) (int64, int32, error) {
	if !strings.HasSuffix(ds, "s") {
		return 0, 0, fmt.Errorf("invalid duration: missing 's' suffix")
	}
	core := strings.TrimSuffix(ds, "s")
	neg := false
	if strings.HasPrefix(core, "-") {
		neg = true
		core = core[1:]
	} else if strings.HasPrefix(core, "+") {
		core = core[1:]
	}
	var secPart, fracPart string
	if i := strings.IndexByte(core, '.'); i >= 0 {
		secPart = core[:i]
		fracPart = core[i+1:]
	} else {
		secPart = core
		fracPart = ""
	}
	if secPart == "" {
		secPart = "0"
	}
	secU, err := strconv.ParseInt(secPart, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid duration seconds: %w", err)
	}
	if len(fracPart) > 9 {
		return 0, 0, fmt.Errorf("invalid duration nanos precision")
	}
	for len(fracPart) < 9 {
		fracPart += "0"
	}
	var nsU int64 = 0
	if fracPart != "" {
		n64, err := strconv.ParseInt(fracPart, 10, 32)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid duration nanos: %w", err)
		}
		nsU = n64
	}
	if neg {
		secU = -secU
		if nsU != 0 {
			secU = secU - 1
			nsU = 1_000_000_000 - nsU
		}
	}
	if secU < -315576000000 || secU > 315576000000 {
		return 0, 0, fmt.Errorf("duration out of range")
	}
	return secU, int32(nsU), nil
}
