package wire

import (
	"encoding/json"
	"reflect"
	"testing"

	"google.golang.org/protobuf/proto"

	pb3 "github.com/anirudhraja/protolite/conformance_test/generated/google/protobuf"
	"github.com/anirudhraja/protolite/schema"
)

func scalarJSONBytesMessage() *schema.Message {
	return &schema.Message{
		Name: "TestAllTypesProto3",
		Fields: []*schema.Field{
			{
				Name:      "optional_bytes",
				Number:    15,
				Label:     schema.LabelOptional,
				JSONBytes: true,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeBytes,
				},
			},
		},
	}
}

func repeatedJSONBytesMessage() *schema.Message {
	return &schema.Message{
		Name: "TestAllTypesProto3",
		Fields: []*schema.Field{
			{
				Name:      "repeated_bytes",
				Number:    45,
				Label:     schema.LabelRepeated,
				JSONBytes: true,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeBytes,
				},
			},
		},
	}
}

func TestJSONBytes_DecodeFromGeneratedCode(t *testing.T) {
	session := map[string]interface{}{
		"uuid":  "dummy-uuid-abc",
		"state": "ACTIVE",
		"id":    "dummy-order-123",
		"nested": map[string]interface{}{
			"k": "v",
		},
	}
	raw, err := json.Marshal(session)
	if err != nil {
		t.Fatal(err)
	}

	wireBytes, err := proto.Marshal(&pb3.TestAllTypesProto3{OptionalBytes: raw})
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}

	decodedI, err := DecodeMessage(wireBytes, scalarJSONBytesMessage(), nil)
	if err != nil {
		t.Fatalf("protolite decode: %v", err)
	}
	decoded, ok := decodedI.(map[string]interface{})
	if !ok {
		t.Fatalf("decoded is %T, want map[string]interface{}", decodedI)
	}
	got, ok := decoded["optional_bytes"].(map[string]interface{})
	if !ok {
		t.Fatalf("optional_bytes decoded as %T, want map[string]interface{}", decoded["optional_bytes"])
	}
	if !reflect.DeepEqual(got, session) {
		t.Errorf("mismatch:\n got:  %#v\n want: %#v", got, session)
	}
}

func TestJSONBytes_EncodeToGeneratedCode(t *testing.T) {
	session := map[string]interface{}{
		"uuid":  "dummy-uuid-abc",
		"state": "ACTIVE",
		"id":    "dummy-order-123",
	}
	data := map[string]interface{}{"optional_bytes": session}

	encoded, err := EncodeMessage(data, scalarJSONBytesMessage(), nil)
	if err != nil {
		t.Fatalf("protolite encode: %v", err)
	}

	var m pb3.TestAllTypesProto3
	if err := proto.Unmarshal(encoded, &m); err != nil {
		t.Fatalf("proto.Unmarshal: %v", err)
	}
	if len(m.OptionalBytes) == 0 {
		t.Fatal("generated OptionalBytes is empty")
	}

	var got map[string]interface{}
	if err := json.Unmarshal(m.OptionalBytes, &got); err != nil {
		t.Fatalf("unmarshal generated payload: %v", err)
	}
	if !reflect.DeepEqual(got, session) {
		t.Errorf("mismatch:\n got:  %#v\n want: %#v", got, session)
	}
}

func TestJSONBytes_RepeatedInteropBothDirections(t *testing.T) {
	elems := []interface{}{
		map[string]interface{}{"id": "a"},
		map[string]interface{}{"id": "b"},
	}

	t.Run("generated_to_protolite", func(t *testing.T) {
		raws := make([][]byte, len(elems))
		for i, e := range elems {
			b, err := json.Marshal(e)
			if err != nil {
				t.Fatal(err)
			}
			raws[i] = b
		}

		wireBytes, err := proto.Marshal(&pb3.TestAllTypesProto3{RepeatedBytes: raws})
		if err != nil {
			t.Fatalf("proto.Marshal: %v", err)
		}

		decodedI, err := DecodeMessage(wireBytes, repeatedJSONBytesMessage(), nil)
		if err != nil {
			t.Fatalf("protolite decode: %v", err)
		}
		decoded := decodedI.(map[string]interface{})
		got, ok := decoded["repeated_bytes"].([]interface{})
		if !ok {
			t.Fatalf("repeated_bytes decoded as %T, want []interface{}", decoded["repeated_bytes"])
		}
		if !reflect.DeepEqual(got, elems) {
			t.Errorf("mismatch:\n got:  %#v\n want: %#v", got, elems)
		}
	})

	t.Run("protolite_to_generated", func(t *testing.T) {
		encoded, err := EncodeMessage(map[string]interface{}{"repeated_bytes": elems}, repeatedJSONBytesMessage(), nil)
		if err != nil {
			t.Fatalf("protolite encode: %v", err)
		}

		var m pb3.TestAllTypesProto3
		if err := proto.Unmarshal(encoded, &m); err != nil {
			t.Fatalf("proto.Unmarshal: %v", err)
		}
		if len(m.RepeatedBytes) != len(elems) {
			t.Fatalf("expected %d repeated_bytes, got %d", len(elems), len(m.RepeatedBytes))
		}
		got := make([]interface{}, len(m.RepeatedBytes))
		for i, r := range m.RepeatedBytes {
			var v interface{}
			if err := json.Unmarshal(r, &v); err != nil {
				t.Fatalf("unmarshal element %d: %v", i, err)
			}
			got[i] = v
		}
		if !reflect.DeepEqual(got, elems) {
			t.Errorf("mismatch:\n got:  %#v\n want: %#v", got, elems)
		}
	})
}
