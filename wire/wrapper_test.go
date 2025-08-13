package wire

import (
	"testing"

	"github.com/anirudhraja/protolite/schema"
)

func TestWrapperTypes_Encoding_Decoding(t *testing.T) {
	tests := []struct {
		name        string
		wrapperType schema.WrapperType
		value       interface{}
		expectValue interface{}
		expectNil   bool
	}{
		{
			name:        "DoubleValue",
			wrapperType: schema.WrapperDoubleValue,
			value:       float64(3.14159),
			expectValue: float64(3.14159),
		},
		{
			name:        "FloatValue",
			wrapperType: schema.WrapperFloatValue,
			value:       float32(2.718),
			expectValue: float32(2.718),
		},
		{
			name:        "Int64Value",
			wrapperType: schema.WrapperInt64Value,
			value:       int64(-9223372036854775808),
			expectValue: int64(-9223372036854775808),
		},
		{
			name:        "UInt64Value",
			wrapperType: schema.WrapperUInt64Value,
			value:       uint64(18446744073709551615),
			expectValue: uint64(18446744073709551615),
		},
		{
			name:        "Int32Value",
			wrapperType: schema.WrapperInt32Value,
			value:       int32(-2147483648),
			expectValue: int32(-2147483648),
		},
		{
			name:        "UInt32Value",
			wrapperType: schema.WrapperUInt32Value,
			value:       uint32(4294967295),
			expectValue: uint32(4294967295),
		},
		{
			name:        "BoolValue_true",
			wrapperType: schema.WrapperBoolValue,
			value:       true,
			expectValue: true,
		},
		{
			name:        "BoolValue_false",
			wrapperType: schema.WrapperBoolValue,
			value:       false,
			expectValue: false,
		},
		{
			name:        "StringValue",
			wrapperType: schema.WrapperStringValue,
			value:       "Hello, wrapper types!",
			expectValue: "Hello, wrapper types!",
		},
		{
			name:        "BytesValue",
			wrapperType: schema.WrapperBytesValue,
			value:       []byte{0x01, 0x02, 0x03, 0xFF},
			expectValue: []byte{0x01, 0x02, 0x03, 0xFF},
		},
		{
			name:        "NilValue",
			wrapperType: schema.WrapperStringValue,
			value:       nil,
			expectValue: nil,
			expectNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test message with wrapper field
			message := &schema.Message{
				Name: "TestMessage",
				Fields: []*schema.Field{
					{
						Name:   "wrapper_field",
						Number: 1,
						Type: schema.FieldType{
							Kind:        schema.KindWrapper,
							WrapperType: tt.wrapperType,
						},
					},
				},
			}

			// Create test data
			var testData map[string]interface{}
			if tt.expectNil {
				testData = map[string]interface{}{}
			} else {
				testData = map[string]interface{}{
					"wrapper_field": tt.value,
				}
			}

			// Encode the message
			encodedData, err := EncodeMessage(testData, message, nil)
			if err != nil {
				t.Fatalf("Failed to encode message: %v", err)
			}

			// Decode the message
			decodedDataI, err := DecodeMessage(encodedData, message, nil)
			if err != nil {
				t.Fatalf("Failed to decode message: %v", err)
			}
			decodedData, ok := decodedDataI.(map[string]interface{})
			if !ok {
				t.Fatalf("decoded data must be of type map[string]interface{} , got: %T", decodedDataI)
			}

			// Verify the result
			if tt.expectNil {
				if wrapperField, exists := decodedData["wrapper_field"]; exists {
					if wrapperField != nil {
						t.Errorf("Expected nil value for wrapper field, got %v", wrapperField)
					}
				}
			} else {
				wrapperField, exists := decodedData["wrapper_field"]
				if !exists {
					t.Errorf("Expected wrapper field to exist")
					return
				}

				if !compareValues(wrapperField, tt.expectValue) {
					t.Errorf("Expected %v (%T), got %v (%T)", tt.expectValue, tt.expectValue, wrapperField, wrapperField)
				}
			}
		})
	}
}

func TestWrapperTypes_RepeatedFields(t *testing.T) {
	// Test repeated wrapper fields
	message := &schema.Message{
		Name: "TestMessage",
		Fields: []*schema.Field{
			{
				Name:   "repeated_strings",
				Number: 1,
				Label:  schema.LabelRepeated,
				Type: schema.FieldType{
					Kind:        schema.KindWrapper,
					WrapperType: schema.WrapperStringValue,
				},
			},
			{
				Name:   "repeated_ints",
				Number: 2,
				Label:  schema.LabelRepeated,
				Type: schema.FieldType{
					Kind:        schema.KindWrapper,
					WrapperType: schema.WrapperInt32Value,
				},
			},
		},
	}

	testData := map[string]interface{}{
		"repeated_strings": []interface{}{"hello", "world", "test"},
		"repeated_ints":    []interface{}{int32(1), int32(2), int32(3)},
	}

	// Encode the message
	encodedData, err := EncodeMessage(testData, message, nil)
	if err != nil {
		t.Fatalf("Failed to encode message: %v", err)
	}

	// Decode the message
	decodedDataI, err := DecodeMessage(encodedData, message, nil)
	if err != nil {
		t.Fatalf("Failed to decode message: %v", err)
	}
	decodedData, ok := decodedDataI.(map[string]interface{})
	if !ok {
		t.Fatalf("decoded data must be of type map[string]interface{} , got: %T", decodedDataI)
	}
	// Verify repeated strings
	stringField, exists := decodedData["repeated_strings"]
	if !exists {
		t.Error("Expected repeated_strings field to exist")
		return
	}

	stringSlice, ok := stringField.([]interface{})
	if !ok {
		t.Errorf("Expected []interface{} for repeated_strings, got %T", stringField)
		return
	}

	expectedStrings := []string{"hello", "world", "test"}
	if len(stringSlice) != len(expectedStrings) {
		t.Errorf("Expected %d strings, got %d", len(expectedStrings), len(stringSlice))
		return
	}

	for i, expected := range expectedStrings {
		if actual, ok := stringSlice[i].(string); !ok || actual != expected {
			t.Errorf("Expected string %s at index %d, got %v", expected, i, stringSlice[i])
		}
	}
	// Verify repeated ints
	intField, exists := decodedData["repeated_ints"]
	if !exists {
		t.Error("Expected repeated_ints field to exist")
		return
	}

	intSlice, ok := intField.([]interface{})
	if !ok {
		t.Errorf("Expected []interface{} for repeated_ints, got %T", intField)
		return
	}

	expectedInts := []int32{1, 2, 3}
	if len(intSlice) != len(expectedInts) {
		t.Errorf("Expected %d ints, got %d", len(expectedInts), len(intSlice))
		return
	}

	for i, expected := range expectedInts {
		if actual, ok := intSlice[i].(int32); !ok || actual != expected {
			t.Errorf("Expected int32 %d at index %d, got %v", expected, i, intSlice[i])
		}
	}
}

func TestWrapperTypes_EdgeCases(t *testing.T) {
	t.Run("empty_string_wrapper", func(t *testing.T) {
		message := &schema.Message{
			Name: "TestMessage",
			Fields: []*schema.Field{
				{
					Name:   "empty_string",
					Number: 1,
					Type: schema.FieldType{
						Kind:        schema.KindWrapper,
						WrapperType: schema.WrapperStringValue,
					},
				},
			},
		}

		testData := map[string]interface{}{
			"empty_string": "",
		}

		// Encode and decode
		encodedData, err := EncodeMessage(testData, message, nil)
		if err != nil {
			t.Fatalf("Failed to encode: %v", err)
		}

		decodedDataI, err := DecodeMessage(encodedData, message, nil)
		if err != nil {
			t.Fatalf("Failed to decode: %v", err)
		}
		decodedData, ok := decodedDataI.(map[string]interface{})
		if !ok {
			t.Fatalf("decoded data must be of type map[string]interface{} , got: %T", decodedDataI)
		}
		if field, exists := decodedData["empty_string"]; !exists {
			t.Error("Expected empty_string field to exist")
		} else if str, ok := field.(string); !ok || str != "" {
			t.Errorf("Expected empty string, got %v (%T)", field, field)
		}
	})

	t.Run("zero_values", func(t *testing.T) {
		message := &schema.Message{
			Name: "TestMessage",
			Fields: []*schema.Field{
				{
					Name:   "zero_int",
					Number: 1,
					Type: schema.FieldType{
						Kind:        schema.KindWrapper,
						WrapperType: schema.WrapperInt32Value,
					},
				},
				{
					Name:   "zero_bool",
					Number: 2,
					Type: schema.FieldType{
						Kind:        schema.KindWrapper,
						WrapperType: schema.WrapperBoolValue,
					},
				},
				{
					Name:   "zero_double",
					Number: 3,
					Type: schema.FieldType{
						Kind:        schema.KindWrapper,
						WrapperType: schema.WrapperDoubleValue,
					},
				},
			},
		}

		testData := map[string]interface{}{
			"zero_int":    int32(0),
			"zero_bool":   false,
			"zero_double": float64(0.0),
		}

		// Encode and decode
		encodedData, err := EncodeMessage(testData, message, nil)
		if err != nil {
			t.Fatalf("Failed to encode: %v", err)
		}

		decodedDataI, err := DecodeMessage(encodedData, message, nil)
		if err != nil {
			t.Fatalf("Failed to decode: %v", err)
		}
		decodedData, ok := decodedDataI.(map[string]interface{})
		if !ok {
			t.Fatalf("decoded data must be of type map[string]interface{} , got: %T", decodedDataI)
		}
		// Verify all zero values are correctly preserved
		if field, exists := decodedData["zero_int"]; !exists {
			t.Error("Expected zero_int field to exist")
		} else if val, ok := field.(int32); !ok || val != 0 {
			t.Errorf("Expected int32(0), got %v (%T)", field, field)
		}

		if field, exists := decodedData["zero_bool"]; !exists {
			t.Error("Expected zero_bool field to exist")
		} else if val, ok := field.(bool); !ok || val != false {
			t.Errorf("Expected false, got %v (%T)", field, field)
		}

		if field, exists := decodedData["zero_double"]; !exists {
			t.Error("Expected zero_double field to exist")
		} else if val, ok := field.(float64); !ok || val != 0.0 {
			t.Errorf("Expected float64(0.0), got %v (%T)", field, field)
		}
	})
}

// Helper function to compare values (handles byte slices specially)
func compareValues(a, b interface{}) bool {
	switch aVal := a.(type) {
	case []byte:
		if bVal, ok := b.([]byte); ok {
			if len(aVal) != len(bVal) {
				return false
			}
			for i := range aVal {
				if aVal[i] != bVal[i] {
					return false
				}
			}
			return true
		}
		return false
	default:
		return a == b
	}
}
