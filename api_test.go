package protolite

import (
	"reflect"
	"strings"
	"testing"

	"github.com/protolite/schema"
	"github.com/protolite/wire"
)

func TestProtolite_Parse(t *testing.T) {
	proto := NewProtolite()

	t.Run("empty_data", func(t *testing.T) {
		result, err := proto.Parse([]byte{})
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if len(result) != 0 {
			t.Errorf("Expected empty result, got %v", result)
		}
	})

	t.Run("simple_varint", func(t *testing.T) {
		// Create a simple protobuf message: field 1 = varint 42
		encoder := wire.NewEncoder()

		// Field tag for field 1, wire type varint (0)
		ve := wire.NewVarintEncoder(encoder)
		tag := wire.MakeTag(wire.FieldNumber(1), wire.WireVarint)
		ve.EncodeVarint(uint64(tag))
		ve.EncodeVarint(42)

		result, err := proto.Parse(encoder.Bytes())
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		expected := map[string]interface{}{
			"field_1": map[string]interface{}{
				"type":  "varint",
				"value": uint64(42),
			},
		}

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("multiple_fields", func(t *testing.T) {
		// Create protobuf with multiple fields
		encoder := wire.NewEncoder()
		ve := wire.NewVarintEncoder(encoder)
		be := wire.NewBytesEncoder(encoder)

		// Field 1: varint 123
		tag1 := wire.MakeTag(wire.FieldNumber(1), wire.WireVarint)
		ve.EncodeVarint(uint64(tag1))
		ve.EncodeVarint(123)

		// Field 2: string "hello"
		tag2 := wire.MakeTag(wire.FieldNumber(2), wire.WireBytes)
		ve.EncodeVarint(uint64(tag2))
		be.EncodeString("hello")

		result, err := proto.Parse(encoder.Bytes())
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("Expected 2 fields, got %d", len(result))
		}

		// Check field 1
		field1, ok := result["field_1"].(map[string]interface{})
		if !ok {
			t.Errorf("field_1 should be a map")
		} else {
			if field1["type"] != "varint" || field1["value"] != uint64(123) {
				t.Errorf("field_1 incorrect: %v", field1)
			}
		}

		// Check field 2
		field2, ok := result["field_2"].(map[string]interface{})
		if !ok {
			t.Errorf("field_2 should be a map")
		} else {
			if field2["type"] != "bytes" {
				t.Errorf("field_2 type incorrect: %v", field2["type"])
			}
			if bytes, ok := field2["value"].([]byte); !ok || string(bytes) != "hello" {
				t.Errorf("field_2 value incorrect: %v", field2["value"])
			}
		}
	})
}

func TestProtolite_WithSchema(t *testing.T) {
	proto := NewProtolite()

	// Define a test message schema
	testMessage := &schema.Message{
		Name: "TestMessage",
		Fields: []*schema.Field{
			{
				Name:   "id",
				Number: 1,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeInt32,
				},
			},
			{
				Name:   "name",
				Number: 2,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeString,
				},
			},
			{
				Name:   "active",
				Number: 3,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeBool,
				},
			},
		},
	}

	// We need to manually create a registry with the message since RegisterSchema isn't implemented
	// For testing, we'll use the wire functions directly
	testData := map[string]interface{}{
		"id":     int32(123),
		"name":   "test message",
		"active": true,
	}

	t.Run("marshal_unmarshal_roundtrip", func(t *testing.T) {
		// Encode using wire functions
		encodedData, err := wire.EncodeMessage(testData, testMessage, nil)
		if err != nil {
			t.Fatalf("Failed to encode: %v", err)
		}

		// Parse using schema-less parsing
		result, err := proto.Parse(encodedData)
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		// Should have 3 fields
		if len(result) != 3 {
			t.Errorf("Expected 3 fields, got %d", len(result))
		}

		// Check that all fields are present
		fields := []string{"field_1", "field_2", "field_3"}
		for _, field := range fields {
			if _, ok := result[field]; !ok {
				t.Errorf("Missing field: %s", field)
			}
		}
	})
}

func TestProtolite_UnmarshalToStruct(t *testing.T) {
	proto := &protolite{}

	// Define a Go struct for testing
	type TestStruct struct {
		ID     int32  `json:"id"`
		Name   string `json:"name"`
		Active bool   `json:"active"`
	}

	testData := map[string]interface{}{
		"id":     int32(123),
		"name":   "test name",
		"active": true,
	}

	t.Run("map_to_struct", func(t *testing.T) {
		var result TestStruct
		err := proto.mapToStruct(testData, &result)
		if err != nil {
			t.Fatalf("mapToStruct failed: %v", err)
		}

		if result.ID != 123 {
			t.Errorf("Expected ID=123, got %d", result.ID)
		}
		if result.Name != "test name" {
			t.Errorf("Expected Name='test name', got '%s'", result.Name)
		}
		if !result.Active {
			t.Errorf("Expected Active=true, got %v", result.Active)
		}
	})

	t.Run("snake_case_conversion", func(t *testing.T) {
		type TestStruct2 struct {
			UserID   int32  `json:"user_id"`
			UserName string `json:"user_name"`
		}

		testData2 := map[string]interface{}{
			"user_id":   int32(456),
			"user_name": "john doe",
		}

		var result TestStruct2
		err := proto.mapToStruct(testData2, &result)
		if err != nil {
			t.Fatalf("mapToStruct failed: %v", err)
		}

		if result.UserID != 456 {
			t.Errorf("Expected UserID=456, got %d", result.UserID)
		}
		if result.UserName != "john doe" {
			t.Errorf("Expected UserName='john doe', got '%s'", result.UserName)
		}
	})

	t.Run("invalid_target", func(t *testing.T) {
		var notAPointer TestStruct
		err := proto.mapToStruct(testData, notAPointer)
		if err == nil {
			t.Error("Expected error for non-pointer target")
		}

		var notAStruct *string
		err = proto.mapToStruct(testData, notAStruct)
		if err == nil {
			t.Error("Expected error for non-struct target")
		}
	})
}

func TestProtolite_toSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"ID", "id"},
		{"UserID", "user_id"},
		{"UserName", "user_name"},
		{"XMLParser", "xml_parser"},
		{"HTTPSConnection", "https_connection"},
		{"SimpleField", "simple_field"},
		{"alreadySnake", "already_snake"},
	}

	for _, test := range tests {
		result := toSnakeCase(test.input)
		if result != test.expected {
			t.Errorf("toSnakeCase(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestProtolite_setFieldValue(t *testing.T) {
	proto := &protolite{}

	t.Run("string_field", func(t *testing.T) {
		type TestStruct struct {
			Name string
		}
		var s TestStruct
		field := reflect.ValueOf(&s).Elem().Field(0)

		err := proto.setFieldValue(field, "test value")
		if err != nil {
			t.Fatalf("setFieldValue failed: %v", err)
		}
		if s.Name != "test value" {
			t.Errorf("Expected 'test value', got '%s'", s.Name)
		}
	})

	t.Run("int_field", func(t *testing.T) {
		type TestStruct struct {
			ID int32
		}
		var s TestStruct
		field := reflect.ValueOf(&s).Elem().Field(0)

		err := proto.setFieldValue(field, int32(123))
		if err != nil {
			t.Fatalf("setFieldValue failed: %v", err)
		}
		if s.ID != 123 {
			t.Errorf("Expected 123, got %d", s.ID)
		}
	})

	t.Run("bool_field", func(t *testing.T) {
		type TestStruct struct {
			Active bool
		}
		var s TestStruct
		field := reflect.ValueOf(&s).Elem().Field(0)

		err := proto.setFieldValue(field, true)
		if err != nil {
			t.Fatalf("setFieldValue failed: %v", err)
		}
		if !s.Active {
			t.Errorf("Expected true, got %v", s.Active)
		}
	})

	t.Run("type_mismatch", func(t *testing.T) {
		type TestStruct struct {
			Name string
		}
		var s TestStruct
		field := reflect.ValueOf(&s).Elem().Field(0)

		err := proto.setFieldValue(field, 123)
		if err == nil {
			t.Error("Expected error for type mismatch")
		}
	})

	t.Run("nil_value", func(t *testing.T) {
		type TestStruct struct {
			Name string
		}
		var s TestStruct
		field := reflect.ValueOf(&s).Elem().Field(0)

		err := proto.setFieldValue(field, nil)
		if err != nil {
			t.Fatalf("setFieldValue failed for nil: %v", err)
		}
		// Name should remain empty string
		if s.Name != "" {
			t.Errorf("Expected empty string, got '%s'", s.Name)
		}
	})
}

func TestProtolite_SchemaRequired(t *testing.T) {
	proto := NewProtolite()

	t.Run("load_schema_from_file", func(t *testing.T) {
		// Test that LoadSchemaFromFile works (even if the file doesn't exist, it should return a proper error)
		err := proto.LoadSchemaFromFile("/nonexistent/path.proto")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
		// Should get a "path does not exist" error from the registry
		if !strings.Contains(err.Error(), "path does not exist") {
			t.Errorf("Expected path error, got: %v", err)
		}
	})
}

func TestProtolite_Integration(t *testing.T) {
	// This test demonstrates the full workflow of encoding and parsing
	t.Run("encode_and_parse_workflow", func(t *testing.T) {
		// Define a message schema
		message := &schema.Message{
			Name: "User",
			Fields: []*schema.Field{
				{
					Name:   "id",
					Number: 1,
					Type: schema.FieldType{
						Kind:          schema.KindPrimitive,
						PrimitiveType: schema.TypeInt32,
					},
				},
				{
					Name:   "email",
					Number: 2,
					Type: schema.FieldType{
						Kind:          schema.KindPrimitive,
						PrimitiveType: schema.TypeString,
					},
				},
				{
					Name:   "verified",
					Number: 3,
					Type: schema.FieldType{
						Kind:          schema.KindPrimitive,
						PrimitiveType: schema.TypeBool,
					},
				},
			},
		}

		// Create test data
		userData := map[string]interface{}{
			"id":       int32(12345),
			"email":    "user@example.com",
			"verified": true,
		}

		// Encode using wire functions (since MarshalWithSchema needs registry)
		encodedData, err := wire.EncodeMessage(userData, message, nil)
		if err != nil {
			t.Fatalf("Failed to encode: %v", err)
		}

		// Parse without schema using Protolite
		proto := NewProtolite()
		parsedData, err := proto.Parse(encodedData)
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		// Verify we got the expected structure
		if len(parsedData) != 3 {
			t.Errorf("Expected 3 fields, got %d", len(parsedData))
		}

		// Check field 1 (id)
		if field1, ok := parsedData["field_1"].(map[string]interface{}); ok {
			if field1["type"] != "varint" {
				t.Errorf("field_1 should be varint type")
			}
			if field1["value"] != uint64(12345) {
				t.Errorf("field_1 value should be 12345, got %v", field1["value"])
			}
		} else {
			t.Error("field_1 missing or wrong type")
		}

		// Check field 2 (email)
		if field2, ok := parsedData["field_2"].(map[string]interface{}); ok {
			if field2["type"] != "bytes" {
				t.Errorf("field_2 should be bytes type")
			}
			if bytes, ok := field2["value"].([]byte); !ok || string(bytes) != "user@example.com" {
				t.Errorf("field_2 value should be 'user@example.com', got %v", field2["value"])
			}
		} else {
			t.Error("field_2 missing or wrong type")
		}

		// Check field 3 (verified)
		if field3, ok := parsedData["field_3"].(map[string]interface{}); ok {
			if field3["type"] != "varint" {
				t.Errorf("field_3 should be varint type")
			}
			if field3["value"] != uint64(1) { // true = 1
				t.Errorf("field_3 value should be 1, got %v", field3["value"])
			}
		} else {
			t.Error("field_3 missing or wrong type")
		}
	})
}
