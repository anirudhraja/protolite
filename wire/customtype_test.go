package wire

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/anirudhraja/protolite/schema"
)

// customTypeMessage returns a message with a single bytes field that is
// annotated with (gogoproto.customtype), i.e. it carries a JSON-encoded value
// on the wire (the pattern used for @thrift GraphQL scalars).
func customTypeMessage() *schema.Message {
	return &schema.Message{
		Name: "ThriftScalarHolder",
		Fields: []*schema.Field{
			{
				Name:       "session",
				Number:     1,
				Label:      schema.LabelOptional,
				CustomType: true,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeBytes,
				},
			},
		},
	}
}

func TestCustomType_RoundTrip(t *testing.T) {
	msg := customTypeMessage()

	session := map[string]interface{}{
		"uuid":  "dummy-uuid-abc",
		"state": "ACTIVE",
		"id":    "dummy-order-123",
	}
	data := map[string]interface{}{"session": session}

	encoded, err := EncodeMessage(data, msg, nil)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	// The field must be length-delimited (wire type 2). This is what the
	// gateway's protolite decoder previously choked on when the value was a
	// nested wrapper message rather than a flat JSON-bytes scalar.
	_, wireType := ParseTag(Tag(encoded[0]))
	if wireType != WireBytes {
		t.Fatalf("expected wire type %d (bytes), got %d", WireBytes, wireType)
	}

	decodedI, err := DecodeMessage(encoded, msg, nil)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	decoded, ok := decodedI.(map[string]interface{})
	if !ok {
		t.Fatalf("decoded is %T, want map[string]interface{}", decodedI)
	}

	got, ok := decoded["session"].(map[string]interface{})
	if !ok {
		t.Fatalf("session decoded as %T, want map[string]interface{}", decoded["session"])
	}
	if !reflect.DeepEqual(got, session) {
		t.Errorf("round-trip mismatch:\n got:  %#v\n want: %#v", got, session)
	}
}

// TestCustomType_DecodesRawJSONBytes mimics the real gateway scenario: a peer
// service (mirror) emits a plain bytes field whose contents are json.Marshal of
// the thrift model. Protolite must json.Unmarshal it back into a structured
// value rather than surfacing the raw base64 bytes.
func TestCustomType_DecodesRawJSONBytes(t *testing.T) {
	msg := customTypeMessage()

	payload := map[string]interface{}{"uuid": "abc", "nested": map[string]interface{}{"k": "v"}}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	// Hand-build the wire bytes: tag for field 1 (bytes) + length-delimited JSON.
	enc := NewEncoder()
	NewVarintEncoder(enc).EncodeVarint(uint64(MakeTag(1, WireBytes)))
	NewBytesEncoder(enc).EncodeBytes(raw)

	decodedI, err := DecodeMessage(enc.Bytes(), msg, nil)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	decoded := decodedI.(map[string]interface{})
	got, ok := decoded["session"].(map[string]interface{})
	if !ok {
		t.Fatalf("session decoded as %T, want map[string]interface{}", decoded["session"])
	}
	if !reflect.DeepEqual(got, payload) {
		t.Errorf("mismatch:\n got:  %#v\n want: %#v", got, payload)
	}
}

func TestCustomType_Repeated(t *testing.T) {
	msg := &schema.Message{
		Name: "RepeatedHolder",
		Fields: []*schema.Field{
			{
				Name:       "sessions",
				Number:     1,
				Label:      schema.LabelRepeated,
				CustomType: true,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeBytes,
				},
			},
		},
	}

	elems := []interface{}{
		map[string]interface{}{"id": "a"},
		map[string]interface{}{"id": "b"},
	}
	data := map[string]interface{}{"sessions": elems}

	encoded, err := EncodeMessage(data, msg, nil)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decodedI, err := DecodeMessage(encoded, msg, nil)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	decoded := decodedI.(map[string]interface{})
	got, ok := decoded["sessions"].([]interface{})
	if !ok {
		t.Fatalf("sessions decoded as %T, want []interface{}", decoded["sessions"])
	}
	if !reflect.DeepEqual(got, elems) {
		t.Errorf("mismatch:\n got:  %#v\n want: %#v", got, elems)
	}
}
