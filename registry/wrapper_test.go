package registry

import (
	"testing"

	"github.com/anirudhraja/protolite/schema"
)

func TestRegistry_WrapperTypeDetection(t *testing.T) {
	registry := NewRegistry([]string{""})

	tests := []struct {
		protoType       string
		expectedKind    schema.TypeKind
		expectedWrapper schema.WrapperType
	}{
		{
			protoType:       "google.protobuf.DoubleValue",
			expectedKind:    schema.KindWrapper,
			expectedWrapper: schema.WrapperDoubleValue,
		},
		{
			protoType:       "google.protobuf.FloatValue",
			expectedKind:    schema.KindWrapper,
			expectedWrapper: schema.WrapperFloatValue,
		},
		{
			protoType:       "google.protobuf.Int64Value",
			expectedKind:    schema.KindWrapper,
			expectedWrapper: schema.WrapperInt64Value,
		},
		{
			protoType:       "google.protobuf.UInt64Value",
			expectedKind:    schema.KindWrapper,
			expectedWrapper: schema.WrapperUInt64Value,
		},
		{
			protoType:       "google.protobuf.Int32Value",
			expectedKind:    schema.KindWrapper,
			expectedWrapper: schema.WrapperInt32Value,
		},
		{
			protoType:       "google.protobuf.UInt32Value",
			expectedKind:    schema.KindWrapper,
			expectedWrapper: schema.WrapperUInt32Value,
		},
		{
			protoType:       "google.protobuf.BoolValue",
			expectedKind:    schema.KindWrapper,
			expectedWrapper: schema.WrapperBoolValue,
		},
		{
			protoType:       "google.protobuf.StringValue",
			expectedKind:    schema.KindWrapper,
			expectedWrapper: schema.WrapperStringValue,
		},
		{
			protoType:       "google.protobuf.BytesValue",
			expectedKind:    schema.KindWrapper,
			expectedWrapper: schema.WrapperBytesValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.protoType, func(t *testing.T) {
			fieldType,err := registry.convertProtoType(tt.protoType,make(map[string]struct{}),"")
			if err != nil {
				t.Errorf("Expected no error for type resolution, got: %v", err)
			}

			if fieldType.Kind != tt.expectedKind {
				t.Errorf("Expected kind %s, got %s", tt.expectedKind, fieldType.Kind)
			}

			if fieldType.WrapperType != tt.expectedWrapper {
				t.Errorf("Expected wrapper type %s, got %s", tt.expectedWrapper, fieldType.WrapperType)
			}
		})
	}
}

func TestRegistry_WrapperTypeResolution(t *testing.T) {
	registry := NewRegistry([]string{""})
	registry.messages = make(map[string]*schema.Message)
	registry.enums = make(map[string]*schema.Enum)

	// Create a message with wrapper fields
	message := &schema.Message{
		Name: "TestMessage",
		Fields: []*schema.Field{
			{
				Name:   "optional_string",
				Number: 1,
				Type: schema.FieldType{
					Kind:        schema.KindWrapper,
					WrapperType: schema.WrapperStringValue,
				},
			},
			{
				Name:   "optional_int",
				Number: 2,
				Type: schema.FieldType{
					Kind:        schema.KindWrapper,
					WrapperType: schema.WrapperInt32Value,
				},
			},
		},
	}

	// Test that wrapper types are correctly resolved (shouldn't error)
	err := registry.resolveMessageFields(message, "test.pkg")
	if err != nil {
		t.Errorf("Expected no error for wrapper type resolution, got: %v", err)
	}

	// Verify the field types are still wrapper types
	for _, field := range message.Fields {
		if field.Type.Kind != schema.KindWrapper {
			t.Errorf("Expected field %s to remain wrapper type, got %s", field.Name, field.Type.Kind)
		}
	}
}

func TestRegistry_NonWrapperTypes(t *testing.T) {
	registry := NewRegistry([]string{""})

	// Test that non-wrapper types are not detected as wrappers
	nonWrapperTypes := []string{
		"MyMessage",
		"google.protobuf.Timestamp",
		"google.protobuf.Any",
		"com.example.User",
	}

	for _, protoType := range nonWrapperTypes {
		t.Run(protoType, func(t *testing.T) {
			fieldType,err := registry.convertProtoType(protoType,make(map[string]struct{}),"")
			if err == nil {
				t.Errorf("Expected error for type resolution, got: %v", err)
			}

			if fieldType != nil && fieldType.Kind == schema.KindWrapper  {
				t.Errorf("Type %s should not be detected as wrapper type", protoType)
			}
		})
	}
}
