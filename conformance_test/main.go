package main

import (
	"encoding/binary"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
    "math"
    "reflect"
    "time"

	"google.golang.org/protobuf/proto"

	protolite "github.com/anirudhraja/protolite"

	conformancepb "github.com/anirudhraja/protolite/conformance_test/generated/conformance"
)

type Harness struct{
	pl protolite.Protolite
}

func main() {
	// Initialize protolite with the directory that contains test .proto files
	// Determine protos root (default to checked-in conformance_test/protos)
	protosRoot := os.Getenv("PROTOLITE_PROTOS_DIR")
	if protosRoot == "" {
		protosRoot = "conformance_test/protos"
	}
	pl := protolite.NewProtolite([]string{protosRoot})
	if err := pl.LoadSchemaFromFile("google/protobuf/test_messages_proto3.proto"); err != nil {
		log.Fatalf("failed to load schema: %v", err)
	}
	h := &Harness{pl: pl}
	totalRuns := 0

	for {
		done, err := h.ServeConformanceRequest(os.Stdin, os.Stdout)
		if err != nil {
			log.Fatalf("conformance-go: fatal error: %v", err)
		}
		if done {
			break
		}
		totalRuns++
	}

	log.Printf("conformance-go: received EOF after %d tests\n", totalRuns)
}

func (h *Harness) ServeConformanceRequest(r io.Reader, w io.Writer) (bool, error) {
	var lenBuf [4]byte
	_, err := io.ReadFull(r, lenBuf[:])
	if err == io.EOF {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("read length: %w", err)
	}

	inLen := binary.LittleEndian.Uint32(lenBuf[:])
	inBytes := make([]byte, inLen)
	if _, err := io.ReadFull(r, inBytes); err != nil {
		return false, fmt.Errorf("read request: %w", err)
	}

	var req conformancepb.ConformanceRequest
	if err := proto.Unmarshal(inBytes, &req); err != nil {
		return false, fmt.Errorf("parse ConformanceRequest: %w", err)
	}

	resp := h.RunTest(&req)

	outBytes, err := proto.Marshal(resp)
	if err != nil {
		return false, fmt.Errorf("marshal response: %w", err)
	}

	var outLen [4]byte
	binary.LittleEndian.PutUint32(outLen[:], uint32(len(outBytes)))

	if _, err := w.Write(outLen[:]); err != nil {
		return false, fmt.Errorf("write length: %w", err)
	}
	if _, err := w.Write(outBytes); err != nil {
		return false, fmt.Errorf("write response: %w", err)
	}
	return false, nil
}

func (h *Harness) RunTest(req *conformancepb.ConformanceRequest) *conformancepb.ConformanceResponse {
	resp := &conformancepb.ConformanceResponse{}

	if req.MessageType == "" {
		resp.Result = &conformancepb.ConformanceResponse_ParseError{
			ParseError: "no message type provided",
		}
		return resp
	}

    // Skip unsupported suites early: proto2/editions messages and JSPB tests.
    if strings.Contains(req.MessageType, ".proto2.") || strings.Contains(req.MessageType, ".editions.") {
        resp.Result = &conformancepb.ConformanceResponse_Skipped{Skipped: "proto2/editions not supported"}
        return resp
    }
    if req.TestCategory == conformancepb.TestCategory_JSPB_TEST {
        resp.Result = &conformancepb.ConformanceResponse_Skipped{Skipped: "JSPB not supported"}
        return resp
    }

	var (
		obj map[string]interface{}
		err error
	)

	switch payload := req.Payload.(type) {
	case *conformancepb.ConformanceRequest_ProtobufPayload:
		obj, err = h.pl.UnmarshalWithSchema(payload.ProtobufPayload, req.MessageType)
		if err != nil {
			resp.Result = &conformancepb.ConformanceResponse_ParseError{
				ParseError: fmt.Sprintf("parse error: %v", err),
			}
			return resp
		}

	case *conformancepb.ConformanceRequest_JsonPayload:
		dec := json.NewDecoder(strings.NewReader(payload.JsonPayload))
		dec.UseNumber()
		if err := dec.Decode(&obj); err != nil {
			resp.Result = &conformancepb.ConformanceResponse_ParseError{ParseError: fmt.Sprintf("parse error: %v", err)}
			return resp
		}
		// Validate JSON by attempting to marshal to protobuf using schema
		bytesOut, merr := h.pl.MarshalWithSchema(obj, req.MessageType)
		if merr != nil {
			resp.Result = &conformancepb.ConformanceResponse_ParseError{ParseError: fmt.Sprintf("json input invalid: %v", merr)}
			return resp
		}
		// Stash validated bytes in context via obj replacement for downstream
		obj["__validated_proto__"] = bytesOut

	case *conformancepb.ConformanceRequest_TextPayload:
		resp.Result = &conformancepb.ConformanceResponse_Skipped{Skipped: "text format not supported by protolite"}
		return resp

	default:
		resp.Result = &conformancepb.ConformanceResponse_ParseError{
			ParseError: "unknown or missing payload type",
		}
		return resp
	}

	switch req.RequestedOutputFormat {
	case conformancepb.WireFormat_PROTOBUF:
		// If JSON was provided and pre-validated, reuse bytes
		if b, ok := obj["__validated_proto__"].([]byte); ok {
			resp.Result = &conformancepb.ConformanceResponse_ProtobufPayload{ProtobufPayload: b}
			return resp
		}
		data, err := h.pl.MarshalWithSchema(obj, req.MessageType)
		if err != nil {
			resp.Result = &conformancepb.ConformanceResponse_SerializeError{
				SerializeError: fmt.Sprintf("serialize error: %v", err),
			}
			return resp
		}
		resp.Result = &conformancepb.ConformanceResponse_ProtobufPayload{
			ProtobufPayload: data,
		}

    case conformancepb.WireFormat_JSON:
		// Convert well-known types to canonical JSON strings and validate bounds
		// If JSON was provided, first decode from validated proto bytes for canonical JSON
		if b, ok := obj["__validated_proto__"].([]byte); ok {
			decoded, derr := h.pl.UnmarshalWithSchema(b, req.MessageType)
			if derr == nil { obj = decoded }
		}
		// Ensure helper field is not emitted
		delete(obj, "__validated_proto__")
		norm, err := h.normalizeForJSON(obj, "")
		if err != nil {
			resp.Result = &conformancepb.ConformanceResponse_SerializeError{SerializeError: fmt.Sprintf("json serialize error: %v", err)}
			return resp
		}
		data, err := json.Marshal(norm)
		if err != nil {
			resp.Result = &conformancepb.ConformanceResponse_SerializeError{
				SerializeError: fmt.Sprintf("json serialize error: %v", err),
			}
			return resp
		}
		resp.Result = &conformancepb.ConformanceResponse_JsonPayload{
			JsonPayload: string(data),
		}

	case conformancepb.WireFormat_TEXT_FORMAT:
		resp.Result = &conformancepb.ConformanceResponse_Skipped{Skipped: "text format not supported by protolite"}
		return resp

	default:
		resp.Result = &conformancepb.ConformanceResponse_RuntimeError{
			RuntimeError: fmt.Sprintf("unknown output format: %v", req.RequestedOutputFormat),
		}
	}
	return resp
}

// --- JSON normalization helpers (WKT + FieldMask) ---
func (h *Harness) normalizeForJSON(v interface{}, parentKey string) (interface{}, error) {
    // Handle maps with non-string keys by converting to map[string]interface{}
    if rv := reflect.ValueOf(v); rv.IsValid() && rv.Kind() == reflect.Map && rv.Type().Key().Kind() != reflect.String {
        out := make(map[string]interface{}, rv.Len())
        for _, mk := range rv.MapKeys() {
            keyStr := ""
            switch mk.Kind() {
            case reflect.String:
                keyStr = mk.String()
            case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
                keyStr = fmt.Sprintf("%d", mk.Int())
            case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
                keyStr = fmt.Sprintf("%d", mk.Uint())
            case reflect.Bool:
                if mk.Bool() { keyStr = "true" } else { keyStr = "false" }
            default:
                keyStr = fmt.Sprintf("%v", mk.Interface())
            }
            val := rv.MapIndex(mk).Interface()
            nv, err := h.normalizeForJSON(val, parentKey)
            if err != nil { return nil, err }
            out[keyStr] = nv
        }
        return out, nil
    }
    switch t := v.(type) {
    case map[string]interface{}:
        // Detect Timestamp/Duration/FieldMask by shape + parent key hint
        if looksLikeSecondsNanos(t) {
            sec, ns := readSecNs(t)
            if strings.Contains(parentKey, "timestamp") {
                if !isValidTimestamp(sec, ns) { return nil, fmt.Errorf("timestamp out of range") }
                return formatRFC3339(sec, ns), nil
            }
            if strings.Contains(parentKey, "duration") {
                if !isValidDuration(sec, ns) { return nil, fmt.Errorf("duration out of range") }
                return formatDuration(sec, ns), nil
            }
        }
        if paths, ok := t["paths"].([]interface{}); ok && strings.Contains(parentKey, "field_mask") {
            out := make([]string, 0, len(paths))
            for _, it := range paths {
                s, ok := it.(string)
                if !ok { continue }
                if !isValidSnakePath(s) { return nil, fmt.Errorf("invalid fieldmask path") }
                out = append(out, toLowerCamelLocal(s))
            }
            return strings.Join(out, ","), nil
        }
        // Any: {type_url:string, value:bytes}
        if tu, ok := t["type_url"].(string); ok {
            // Only treat as Any when we also see a value field
            if rawVal, exists := t["value"]; exists {
                at := map[string]interface{}{"@type": tu}
                // Try to expand the Any payload if known
                var b []byte
                switch rv := rawVal.(type) {
                case []byte:
                    b = rv
                case string:
                    // value may already be base64; keep string for fallback and try decode
                    db, err := base64.StdEncoding.DecodeString(rv)
                    if err == nil { b = db }
                }
                if len(b) > 0 {
                    typeName := tu
                    if i := strings.LastIndex(typeName, "/"); i >= 0 && i+1 < len(typeName) { typeName = typeName[i+1:] }
                    if inner, err := h.pl.UnmarshalWithSchema(b, typeName); err == nil {
                        // Normalize inner and merge its fields
                        norm, err := h.normalizeForJSON(inner, "")
                        if err == nil {
                            if mm, ok := norm.(map[string]interface{}); ok {
                                for k, v := range mm { at[k] = v }
                                return at, nil
                            }
                        }
                    }
                }
                // Fallback: keep base64 value
                switch rv := rawVal.(type) {
                case []byte:
                    at["value"] = base64.StdEncoding.EncodeToString(rv)
                case string:
                    at["value"] = rv
                default:
                    // nothing
                }
                return at, nil
            }
        }
        // Struct: {fields: map<string, Value>}
        if fields, ok := t["fields"].(map[string]interface{}); ok {
            out := make(map[string]interface{}, len(fields))
            for k, vv := range fields {
                out[k] = h.valueMessageToJSON(vv)
            }
            return out, nil
        }
        // Value: oneof mapping
        if isValueMessage(t) {
            return h.valueMessageToJSON(t), nil
        }
        // Recurse into entries
        res := make(map[string]interface{}, len(t))
        for k, val := range t {
            nv, err := h.normalizeForJSON(val, k)
            if err != nil { return nil, err }
            res[k] = nv
        }
        return res, nil
    case []interface{}:
        arr := make([]interface{}, len(t))
        for i := range t {
            nv, err := h.normalizeForJSON(t[i], parentKey)
            if err != nil { return nil, err }
            arr[i] = nv
        }
        return arr, nil
    default:
        // Stringify 64-bit integers per JSON mapping
        switch x := v.(type) {
        case int64:
            return fmt.Sprintf("%d", x), nil
        case uint64:
            return fmt.Sprintf("%d", x), nil
        case float64:
            if math.IsNaN(x) { return "NaN", nil }
            if math.IsInf(x, +1) { return "Infinity", nil }
            if math.IsInf(x, -1) { return "-Infinity", nil }
            return v, nil
        case float32:
            f := float64(x)
            if math.IsNaN(f) { return "NaN", nil }
            if math.IsInf(f, +1) { return "Infinity", nil }
            if math.IsInf(f, -1) { return "-Infinity", nil }
            return v, nil
        default:
            return v, nil
        }
    }
}

func isValueMessage(m map[string]interface{}) bool {
    if _, ok := m["null_value"]; ok { return true }
    if _, ok := m["number_value"]; ok { return true }
    if _, ok := m["string_value"]; ok { return true }
    if _, ok := m["bool_value"]; ok { return true }
    if _, ok := m["struct_value"]; ok { return true }
    if _, ok := m["list_value"]; ok { return true }
    return false
}

func (h *Harness) valueMessageToJSON(v interface{}) interface{} {
    // v may already be a plain JSON value; handle maps specially
    if m, ok := v.(map[string]interface{}); ok {
        if _, ok := m["null_value"]; ok {
            return nil
        }
        if num, ok := m["number_value"]; ok {
            return num
        }
        if s, ok := m["string_value"]; ok {
            return s
        }
        if b, ok := m["bool_value"]; ok {
            return b
        }
        if sv, ok := m["struct_value"].(map[string]interface{}); ok {
            if fields, ok := sv["fields"].(map[string]interface{}); ok {
                out := make(map[string]interface{}, len(fields))
                for k, vv := range fields { out[k] = h.valueMessageToJSON(vv) }
                return out
            }
        }
        if lv, ok := m["list_value"].(map[string]interface{}); ok {
            if arr, ok := lv["values"].([]interface{}); ok {
                out := make([]interface{}, len(arr))
                for i := range arr { out[i] = h.valueMessageToJSON(arr[i]) }
                return out
            }
        }
    }
    // unchanged
    return v
}

func looksLikeSecondsNanos(m map[string]interface{}) bool {
    _, hasSec := m["seconds"]
    if !hasSec { return false }
    // nanos optional
    return true
}

func readSecNs(m map[string]interface{}) (int64, int32) {
    var sec int64
    var ns int32
    switch s := m["seconds"].(type) {
    case int64:
        sec = s
    case uint64:
        sec = int64(s)
    case float64:
        sec = int64(s)
    }
    switch n := m["nanos"].(type) {
    case int32:
        ns = n
    case int64:
        ns = int32(n)
    case float64:
        ns = int32(n)
    default:
        ns = 0
    }
    return sec, ns
}

func isValidTimestamp(sec int64, ns int32) bool {
    if ns < 0 || ns > 999_999_999 { return false }
    if sec < -62135596800 || sec > 253402300799 { return false }
    return true
}

func isValidDuration(sec int64, ns int32) bool {
    if ns <= -1_000_000_000 || ns >= 1_000_000_000 { return false }
    if sec < -315576000000 || sec > 315576000000 { return false }
    return true
}

func formatRFC3339(sec int64, ns int32) string {
    return time.Unix(sec, int64(ns)).UTC().Format(time.RFC3339Nano)
}

func formatDuration(sec int64, ns int32) string {
    sign := ""
    if sec < 0 || (sec == 0 && ns < 0) {
        sign = "-"
        if ns != 0 {
            sec = -sec - 1
            ns = 1_000_000_000 - ns
        } else {
            sec = -sec
        }
    }
    if ns == 0 { return sign + fmt.Sprintf("%ds", sec) }
    frac := fmt.Sprintf("%09d", ns)
    i := len(frac)
    for i > 0 && frac[i-1] == '0' { i-- }
    frac = frac[:i]
    return fmt.Sprintf("%s%d.%ss", sign, sec, frac)
}

func toLowerCamelLocal(s string) string {
    if s == "" { return s }
    out := make([]byte, 0, len(s))
    upperNext := false
    for i := 0; i < len(s); i++ {
        c := s[i]
        if c == '_' { upperNext = true; continue }
        if len(out) == 0 {
            if c >= 'A' && c <= 'Z' { c = c - 'A' + 'a' }
            out = append(out, c)
            upperNext = false
            continue
        }
        if upperNext && c >= 'a' && c <= 'z' { c = c - 'a' + 'A' }
        upperNext = false
        out = append(out, c)
    }
    return string(out)
}

func isValidSnakePath(p string) bool {
    if p == "" { return true }
    if p[0] == '_' || p[len(p)-1] == '_' { return false }
    prevUnderscore := false
    for i := 0; i < len(p); i++ {
        c := p[i]
        if c == '_' {
            if prevUnderscore { return false }
            prevUnderscore = true
            continue
        }
        prevUnderscore = false
        if c >= 'a' && c <= 'z' { continue }
        if c >= '0' && c <= '9' { continue }
        return false
    }
    return true
}
