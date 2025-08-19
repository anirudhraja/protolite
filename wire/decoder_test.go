package wire

import (
	"math"
	"reflect"
	"testing"

	"github.com/anirudhraja/protolite/registry"
	"github.com/anirudhraja/protolite/schema"
)

func TestDecoder_AllTypes(t *testing.T) {
	// Define the main comprehensive message - focusing on primitive types first
	mainMessage := &schema.Message{
		Name: "ComprehensiveMessage",
		Fields: []*schema.Field{
			// Primitive types - varint wire type
			{
				Name:   "test_int32",
				Number: 1,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeInt32,
				},
			},
			{
				Name:   "test_int64",
				Number: 2,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeInt64,
				},
			},
			{
				Name:   "test_uint32",
				Number: 3,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeUint32,
				},
			},
			{
				Name:   "test_uint64",
				Number: 4,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeUint64,
				},
			},

			{
				Name:   "test_bool",
				Number: 5,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeBool,
				},
			},
			// Fixed width types
			{
				Name:   "test_float",
				Number: 6,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeFloat,
				},
			},
			{
				Name:   "test_double",
				Number: 7,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeDouble,
				},
			},
			// Bytes types
			{
				Name:   "test_string",
				Number: 8,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeString,
				},
			},
			{
				Name:   "test_bytes",
				Number: 9,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeBytes,
				},
			},
		},
	}

	// Create test data (without map for now, as it requires registry)
	testData := map[string]interface{}{
		"test_int32":  int32(-123),
		"test_int64":  int64(-456789),
		"test_uint32": uint32(123),
		"test_uint64": uint64(456789),
		"test_bool":   true,
		"test_float":  float32(3.14),
		"test_double": float64(2.718281828),
		"test_string": "Hello, protolite!",
		"test_bytes":  []byte("binary data"),
	}

	// Encode the message (without registry for primitive types)
	encodedData, err := EncodeMessage(testData, mainMessage, nil)
	if err != nil {
		t.Fatalf("Failed to encode message: %v", err)
	}

	// Decode the message
	decodedDataI, err := DecodeMessage(encodedData, mainMessage, nil)
	if err != nil {
		t.Fatalf("Failed to decode message: %v", err)
	}
	decodedData, ok := decodedDataI.(map[string]interface{})
	if !ok {
		t.Fatalf("decoded data must be of type map[string]interface{} , got: %T", decodedDataI)
	}
	// Verify the results
	tests := []struct {
		field    string
		expected interface{}
	}{
		{"test_int32", int32(-123)},
		{"test_int64", int64(-456789)},
		{"test_uint32", uint32(123)},
		{"test_uint64", uint64(456789)},
		{"test_bool", true},
		{"test_float", float32(3.14)},
		{"test_double", float64(2.718281828)},
		{"test_string", "Hello, protolite!"},
	}

	for _, test := range tests {
		actual, exists := decodedData[test.field]
		if !exists {
			t.Errorf("Field %s not found in decoded data", test.field)
			continue
		}

		if !reflect.DeepEqual(actual, test.expected) {
			t.Errorf("Field %s: expected %v (%T), got %v (%T)",
				test.field, test.expected, test.expected, actual, actual)
		}
	}

	// Verify bytes field separately
	expectedBytes := []byte("binary data")
	actualBytes, exists := decodedData["test_bytes"]
	if !exists {
		t.Error("Field test_bytes not found in decoded data")
	} else if !reflect.DeepEqual(actualBytes, expectedBytes) {
		t.Errorf("Field test_bytes: expected %v, got %v", expectedBytes, actualBytes)
	}
}

func TestDecoder_PrimitiveTypes(t *testing.T) {
	tests := []struct {
		name          string
		primitiveType schema.PrimitiveType
		testValue     interface{}
		wireType      WireType
	}{
		{"int32", schema.TypeInt32, int32(42), WireVarint},
		{"int64", schema.TypeInt64, int64(1234567890), WireVarint},
		{"uint32", schema.TypeUint32, uint32(42), WireVarint},
		{"uint64", schema.TypeUint64, uint64(1234567890), WireVarint},

		{"bool_true", schema.TypeBool, true, WireVarint},
		{"bool_false", schema.TypeBool, false, WireVarint},
		{"float", schema.TypeFloat, float32(3.14), WireFixed32},
		{"double", schema.TypeDouble, float64(2.718281828), WireFixed64},
		{"string", schema.TypeString, "test string", WireBytes},
		{"bytes", schema.TypeBytes, []byte("test bytes"), WireBytes},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a simple message with one field
			msg := &schema.Message{
				Name: "TestMessage",
				Fields: []*schema.Field{
					{
						Name:   "test_field",
						Number: 1,
						Type: schema.FieldType{
							Kind:          schema.KindPrimitive,
							PrimitiveType: test.primitiveType,
						},
					},
				},
			}

			// Create test data
			data := map[string]interface{}{
				"test_field": test.testValue,
			}

			// Encode
			encoder := NewEncoder()
			me := NewMessageEncoder(encoder)
			err := me.EncodeMessage(data, msg)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			// Decode
			decodedDataI, err := DecodeMessage(encoder.Bytes(), msg, nil)
			if err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
			decodedData, ok := decodedDataI.(map[string]interface{})
			if !ok {
				t.Fatalf("decoded data must be of type map[string]interface{} , got: %T", decodedDataI)
			}
			// Verify
			actual := decodedData["test_field"]
			if !reflect.DeepEqual(actual, test.testValue) {
				t.Errorf("Expected %v (%T), got %v (%T)",
					test.testValue, test.testValue, actual, actual)
			}
		})
	}
}

func TestDecoder_EdgeCases(t *testing.T) {
	t.Run("empty_message", func(t *testing.T) {
		msg := &schema.Message{
			Name:   "EmptyMessage",
			Fields: []*schema.Field{},
		}

		data := map[string]interface{}{}

		// Encode
		encodedData, err := EncodeMessage(data, msg, nil)
		if err != nil {
			t.Fatalf("Failed to encode empty message: %v", err)
		}

		// Decode
		decodedDataI, err := DecodeMessage(encodedData, msg, nil)
		if err != nil {
			t.Fatalf("Failed to decode empty message: %v", err)
		}
		decodedData, ok := decodedDataI.(map[string]interface{})
		if !ok {
			t.Fatalf("decoded data must be of type map[string]interface{} , got: %T", decodedDataI)
		}

		if len(decodedData) != 0 {
			t.Errorf("Expected empty map, got %v", decodedData)
		}
	})

	t.Run("zero_values", func(t *testing.T) {
		msg := &schema.Message{
			Name: "ZeroMessage",
			Fields: []*schema.Field{
				{
					Name:   "zero_int",
					Number: 1,
					Type: schema.FieldType{
						Kind:          schema.KindPrimitive,
						PrimitiveType: schema.TypeInt32,
					},
				},
				{
					Name:   "zero_string",
					Number: 2,
					Type: schema.FieldType{
						Kind:          schema.KindPrimitive,
						PrimitiveType: schema.TypeString,
					},
				},
				{
					Name:   "zero_bool",
					Number: 3,
					Type: schema.FieldType{
						Kind:          schema.KindPrimitive,
						PrimitiveType: schema.TypeBool,
					},
				},
			},
		}

		data := map[string]interface{}{
			"zero_int":    int32(0),
			"zero_string": "",
			"zero_bool":   false,
		}

		// Encode
		encodedData, err := EncodeMessage(data, msg, nil)
		if err != nil {
			t.Fatalf("Failed to encode zero values: %v", err)
		}

		// Decode
		decodedDataI, err := DecodeMessage(encodedData, msg, nil)
		if err != nil {
			t.Fatalf("Failed to decode zero values: %v", err)
		}

		decodedData, ok := decodedDataI.(map[string]interface{})
		if !ok {
			t.Fatalf("decoded data must be of type map[string]interface{} , got: %T", decodedDataI)
		}
		// Verify
		if decodedData["zero_int"] != int32(0) {
			t.Errorf("Expected zero_int=0, got %v", decodedData["zero_int"])
		}
		if decodedData["zero_string"] != "" {
			t.Errorf("Expected zero_string='', got %v", decodedData["zero_string"])
		}
		if decodedData["zero_bool"] != false {
			t.Errorf("Expected zero_bool=false, got %v", decodedData["zero_bool"])
		}
	})

	t.Run("extreme_values", func(t *testing.T) {
		msg := &schema.Message{
			Name: "ExtremeMessage",
			Fields: []*schema.Field{
				{
					Name:   "max_int32",
					Number: 1,
					Type: schema.FieldType{
						Kind:          schema.KindPrimitive,
						PrimitiveType: schema.TypeInt32,
					},
				},
				{
					Name:   "min_int32",
					Number: 2,
					Type: schema.FieldType{
						Kind:          schema.KindPrimitive,
						PrimitiveType: schema.TypeInt32,
					},
				},
				{
					Name:   "max_float",
					Number: 3,
					Type: schema.FieldType{
						Kind:          schema.KindPrimitive,
						PrimitiveType: schema.TypeFloat,
					},
				},
				{
					Name:   "inf_double",
					Number: 4,
					Type: schema.FieldType{
						Kind:          schema.KindPrimitive,
						PrimitiveType: schema.TypeDouble,
					},
				},
			},
		}

		data := map[string]interface{}{
			"max_int32":  int32(math.MaxInt32),
			"min_int32":  int32(math.MinInt32),
			"max_float":  float32(math.MaxFloat32),
			"inf_double": math.Inf(1),
		}

		// Encode
		encodedData, err := EncodeMessage(data, msg, nil)
		if err != nil {
			t.Fatalf("Failed to encode extreme values: %v", err)
		}

		// Decode
		decodedDataI, err := DecodeMessage(encodedData, msg, nil)
		if err != nil {
			t.Fatalf("Failed to decode extreme values: %v", err)
		}
		decodedData, ok := decodedDataI.(map[string]interface{})
		if !ok {
			t.Fatalf("decoded data must be of type map[string]interface{} , got: %T", decodedDataI)
		}

		// Verify
		if decodedData["max_int32"] != int32(math.MaxInt32) {
			t.Errorf("Expected max_int32=%v, got %v", int32(math.MaxInt32), decodedData["max_int32"])
		}
		if decodedData["min_int32"] != int32(math.MinInt32) {
			t.Errorf("Expected min_int32=%v, got %v", int32(math.MinInt32), decodedData["min_int32"])
		}
		if decodedData["max_float"] != float32(math.MaxFloat32) {
			t.Errorf("Expected max_float=%v, got %v", float32(math.MaxFloat32), decodedData["max_float"])
		}
		if infDouble, exists := decodedData["inf_double"]; !exists {
			t.Error("Field inf_double not found in decoded data")
		} else if !math.IsInf(infDouble.(float64), 1) {
			t.Errorf("Expected inf_double=+Inf, got %v", infDouble)
		}
	})
}

func TestDecoder_NestedMessages(t *testing.T) {
	// Define a nested message
	nestedMessage := &schema.Message{
		Name: "Address",
		Fields: []*schema.Field{
			{
				Name:   "street",
				Number: 1,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeString,
				},
			},
			{
				Name:   "city",
				Number: 2,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeString,
				},
			},
			{
				Name:   "zip_code",
				Number: 3,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeInt32,
				},
			},
		},
	}

	// Define main message with nested message
	mainMessage := &schema.Message{
		Name: "Person",
		Fields: []*schema.Field{
			{
				Name:   "name",
				Number: 1,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeString,
				},
			},
			{
				Name:   "age",
				Number: 2,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeInt32,
				},
			},
			{
				Name:   "address",
				Number: 3,
				Type: schema.FieldType{
					Kind:        schema.KindMessage,
					MessageType: "Address",
				},
			},
		},
	}

	// Create a registry and populate it
	reg := registry.NewRegistry([]string{""})

	// First, encode the nested message separately
	nestedData := map[string]interface{}{
		"street":   "123 Main St",
		"city":     "Anytown",
		"zip_code": int32(12345),
	}

	nestedBytes, err := EncodeMessage(nestedData, nestedMessage, nil)
	if err != nil {
		t.Fatalf("Failed to encode nested message: %v", err)
	}

	// Create test data with the nested message as bytes
	testData := map[string]interface{}{
		"name":    "John Doe",
		"age":     int32(30),
		"address": nestedBytes,
	}

	// Encode the main message
	encodedData, err := EncodeMessage(testData, mainMessage, reg)
	if err != nil {
		t.Fatalf("Failed to encode main message: %v", err)
	}

	// Decode the main message (without registry, should get raw bytes for nested message)
	decodedDataI, err := DecodeMessage(encodedData, mainMessage, nil)
	if err != nil {
		t.Fatalf("Failed to decode main message: %v", err)
	}
	decodedData, ok := decodedDataI.(map[string]interface{})
	if !ok {
		t.Fatalf("decoded data must be of type map[string]interface{} , got: %T", decodedDataI)
	}

	// Verify primitive fields
	if decodedData["name"] != "John Doe" {
		t.Errorf("Expected name='John Doe', got %v", decodedData["name"])
	}
	if decodedData["age"] != int32(30) {
		t.Errorf("Expected age=30, got %v", decodedData["age"])
	}

	// The address should be raw bytes since we don't have registry
	addressBytes, ok := decodedData["address"].([]byte)
	if !ok {
		t.Errorf("Expected address to be []byte, got %T", decodedData["address"])
	} else {
		// Decode the nested message manually
		nestedDecodedI, err := DecodeMessage(addressBytes, nestedMessage, nil)
		if err != nil {
			t.Fatalf("Failed to decode nested message: %v", err)
		}
		nestedDecoded, ok := nestedDecodedI.(map[string]interface{})
		if !ok {
			t.Fatalf("decoded data must be of type map[string]interface{} , got: %T", decodedDataI)
		}

		if nestedDecoded["street"] != "123 Main St" {
			t.Errorf("Expected street='123 Main St', got %v", nestedDecoded["street"])
		}
		if nestedDecoded["city"] != "Anytown" {
			t.Errorf("Expected city='Anytown', got %v", nestedDecoded["city"])
		}
		if nestedDecoded["zip_code"] != int32(12345) {
			t.Errorf("Expected zip_code=12345, got %v", nestedDecoded["zip_code"])
		}
	}
}

func TestDecoder_MapTypes(t *testing.T) {
	// Define message with map fields
	message := &schema.Message{
		Name: "ConfigMap",
		Fields: []*schema.Field{
			{
				Name:   "string_map",
				Number: 1,
				Type: schema.FieldType{
					Kind: schema.KindMap,
					MapKey: &schema.FieldType{
						Kind:          schema.KindPrimitive,
						PrimitiveType: schema.TypeString,
					},
					MapValue: &schema.FieldType{
						Kind:          schema.KindPrimitive,
						PrimitiveType: schema.TypeString,
					},
				},
			},
			{
				Name:   "int_map",
				Number: 2,
				Type: schema.FieldType{
					Kind: schema.KindMap,
					MapKey: &schema.FieldType{
						Kind:          schema.KindPrimitive,
						PrimitiveType: schema.TypeString,
					},
					MapValue: &schema.FieldType{
						Kind:          schema.KindPrimitive,
						PrimitiveType: schema.TypeInt32,
					},
				},
			},
			{
				Name:   "bool_map",
				Number: 3,
				Type: schema.FieldType{
					Kind: schema.KindMap,
					MapKey: &schema.FieldType{
						Kind:          schema.KindPrimitive,
						PrimitiveType: schema.TypeString,
					},
					MapValue: &schema.FieldType{
						Kind:          schema.KindPrimitive,
						PrimitiveType: schema.TypeBool,
					},
				},
			},
		},
	}

	// Test encoding individual map entries (since maps are encoded as repeated map entries)
	t.Run("individual_map_entries", func(t *testing.T) {
		// Test individual map entries for string_map
		mapEncoder := NewMapEncoder(NewEncoder())

		// Test string-string map entry
		err := mapEncoder.EncodeMapEntry("key1", "value1",
			&schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeString},
			&schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeString})
		if err != nil {
			t.Fatalf("Failed to encode string map entry: %v", err)
		}

		// Test string-int map entry
		err = mapEncoder.EncodeMapEntry("count", int32(42),
			&schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeString},
			&schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeInt32})
		if err != nil {
			t.Fatalf("Failed to encode int map entry: %v", err)
		}

		// Test string-bool map entry
		err = mapEncoder.EncodeMapEntry("enabled", true,
			&schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeString},
			&schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeBool})
		if err != nil {
			t.Fatalf("Failed to encode bool map entry: %v", err)
		}
	})

	t.Run("map_entry_decoding", func(t *testing.T) {
		// Create a simple map entry and test decoding
		encoder := NewEncoder()
		mapEncoder := NewMapEncoder(encoder)

		// Encode a single map entry
		err := mapEncoder.EncodeMapEntry("test_key", "test_value",
			&schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeString},
			&schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeString})
		if err != nil {
			t.Fatalf("Failed to encode map entry: %v", err)
		}

		// Decode the map entry
		decoder := NewDecoder(encoder.Bytes())
		mapDecoder := NewMapDecoder(decoder)

		key, value, err := mapDecoder.DecodeMapEntry(
			&schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeString},
			&schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeString})
		if err != nil {
			t.Fatalf("Failed to decode map entry: %v", err)
		}

		if key != "test_key" {
			t.Errorf("Expected key='test_key', got %v", key)
		}
		if value != "test_value" {
			t.Errorf("Expected value='test_value', got %v", value)
		}
	})

	t.Run("empty_maps", func(t *testing.T) {
		// Empty maps should be omitted from encoding entirely
		emptyData := map[string]interface{}{}

		// Encode empty maps
		encodedData, err := EncodeMessage(emptyData, message, nil)
		if err != nil {
			t.Fatalf("Failed to encode empty maps: %v", err)
		}

		// Decode
		decodedDataI, err := DecodeMessage(encodedData, message, nil)
		if err != nil {
			t.Fatalf("Failed to decode empty maps: %v", err)
		}
		decodedData, ok := decodedDataI.(map[string]interface{})
		if !ok {
			t.Fatalf("decoded data must be of type map[string]interface{} , got: %T", decodedDataI)
		}

		// Should have null valued keys.
		if !reflect.DeepEqual(decodedData, map[string]interface{}{
			"bool_map":   nil,
			"int_map":    nil,
			"string_map": nil,
		}) {
			t.Errorf("Expected fields with null values, got %v", decodedData)
		}
	})
}

func TestDecoder_RecursiveNestedMessages(t *testing.T) {
	// Define a recursive structure: TreeNode with children
	treeNodeMessage := &schema.Message{
		Name: "TreeNode",
		Fields: []*schema.Field{
			{
				Name:   "value",
				Number: 1,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeInt32,
				},
			},
			{
				Name:   "left_child",
				Number: 2,
				Type: schema.FieldType{
					Kind:        schema.KindMessage,
					MessageType: "TreeNode",
				},
			},
			{
				Name:   "right_child",
				Number: 3,
				Type: schema.FieldType{
					Kind:        schema.KindMessage,
					MessageType: "TreeNode",
				},
			},
		},
	}

	// Create a simple binary tree: root(1) -> left(2), right(3)
	leftChild := map[string]interface{}{
		"value": int32(2),
	}
	rightChild := map[string]interface{}{
		"value": int32(3),
	}

	// Encode children first
	leftBytes, err := EncodeMessage(leftChild, treeNodeMessage, nil)
	if err != nil {
		t.Fatalf("Failed to encode left child: %v", err)
	}

	rightBytes, err := EncodeMessage(rightChild, treeNodeMessage, nil)
	if err != nil {
		t.Fatalf("Failed to encode right child: %v", err)
	}

	// Create root with children
	root := map[string]interface{}{
		"value":       int32(1),
		"left_child":  leftBytes,
		"right_child": rightBytes,
	}

	// Encode root
	encodedData, err := EncodeMessage(root, treeNodeMessage, nil)
	if err != nil {
		t.Fatalf("Failed to encode root: %v", err)
	}

	// Decode root
	decodedDataI, err := DecodeMessage(encodedData, treeNodeMessage, nil)
	if err != nil {
		t.Fatalf("Failed to decode root: %v", err)
	}
	decodedData, ok := decodedDataI.(map[string]interface{})
	if !ok {
		t.Fatalf("decoded data must be of type map[string]interface{} , got: %T", decodedDataI)
	}

	// Verify root value
	if decodedData["value"] != int32(1) {
		t.Errorf("Expected root value=1, got %v", decodedData["value"])
	}

	// Verify left child (should be raw bytes)
	leftChildBytes, ok := decodedData["left_child"].([]byte)
	if !ok {
		t.Errorf("Expected left_child to be []byte, got %T", decodedData["left_child"])
	} else {
		// Decode left child
		leftDecodedI, err := DecodeMessage(leftChildBytes, treeNodeMessage, nil)
		if err != nil {
			t.Fatalf("Failed to decode left child: %v", err)
		}
		leftDecoded, ok := leftDecodedI.(map[string]interface{})
		if !ok {
			t.Fatalf("decoded data must be of type map[string]interface{} , got: %T", decodedDataI)
		}

		if leftDecoded["value"] != int32(2) {
			t.Errorf("Expected left child value=2, got %v", leftDecoded["value"])
		}
	}

	// Verify right child (should be raw bytes)
	rightChildBytes, ok := decodedData["right_child"].([]byte)
	if !ok {
		t.Errorf("Expected right_child to be []byte, got %T", decodedData["right_child"])
	} else {
		// Decode right child
		rightDecodedI, err := DecodeMessage(rightChildBytes, treeNodeMessage, nil)
		if err != nil {
			t.Fatalf("Failed to decode right child: %v", err)
		}
		rightDecoded, ok := rightDecodedI.(map[string]interface{})
		if !ok {
			t.Fatalf("decoded data must be of type map[string]interface{} , got: %T", decodedDataI)
		}

		if rightDecoded["value"] != int32(3) {
			t.Errorf("Expected right child value=3, got %v", rightDecoded["value"])
		}
	}
}

func TestDecoder_ComplexMixed(t *testing.T) {
	// Define a complex message that combines all types
	complexMessage := &schema.Message{
		Name: "ComplexMessage",
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
				Name:   "metadata",
				Number: 4,
				Type: schema.FieldType{
					Kind:        schema.KindMessage,
					MessageType: "Metadata",
				},
			},
		},
	}

	metadataMessage := &schema.Message{
		Name: "Metadata",
		Fields: []*schema.Field{
			{
				Name:   "created_at",
				Number: 1,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeString,
				},
			},
			{
				Name:   "version",
				Number: 2,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeInt32,
				},
			},
		},
	}

	// Create nested metadata
	metadata := map[string]interface{}{
		"created_at": "2023-01-01T00:00:00Z",
		"version":    int32(1),
	}
	metadataBytes, err := EncodeMessage(metadata, metadataMessage, nil)
	if err != nil {
		t.Fatalf("Failed to encode metadata: %v", err)
	}

	// Create complex test data
	testData := map[string]interface{}{
		"id":       int32(12345),
		"name":     "Complex Test",
		"metadata": metadataBytes,
	}

	// Encode
	encodedData, err := EncodeMessage(testData, complexMessage, nil)
	if err != nil {
		t.Fatalf("Failed to encode complex message: %v", err)
	}

	// Decode
	decodedDataI, err := DecodeMessage(encodedData, complexMessage, nil)
	if err != nil {
		t.Fatalf("Failed to decode complex message: %v", err)
	}
	decodedData, ok := decodedDataI.(map[string]interface{})
	if !ok {
		t.Fatalf("decoded data must be of type map[string]interface{} , got: %T", decodedDataI)
	}

	// Verify primitive fields
	if decodedData["id"] != int32(12345) {
		t.Errorf("Expected id=12345, got %v", decodedData["id"])
	}
	if decodedData["name"] != "Complex Test" {
		t.Errorf("Expected name='Complex Test', got %v", decodedData["name"])
	}

	// Verify nested message
	metadataBytes2, ok := decodedData["metadata"].([]byte)
	if !ok {
		t.Errorf("Expected metadata to be []byte, got %T", decodedData["metadata"])
	} else {
		metadataDecodedI, err := DecodeMessage(metadataBytes2, metadataMessage, nil)
		if err != nil {
			t.Fatalf("Failed to decode metadata: %v", err)
		}
		metadataDecoded, ok := metadataDecodedI.(map[string]interface{})
		if !ok {
			t.Fatalf("decoded data must be of type map[string]interface{} , got: %T", decodedDataI)
		}

		if metadataDecoded["created_at"] != "2023-01-01T00:00:00Z" {
			t.Errorf("Expected created_at='2023-01-01T00:00:00Z', got %v", metadataDecoded["created_at"])
		}
		if metadataDecoded["version"] != int32(1) {
			t.Errorf("Expected version=1, got %v", metadataDecoded["version"])
		}
	}
}

func TestDecoder_JSONNames(t *testing.T) {
	// Define a complex message that combines all types
	complexMessage := &schema.Message{
		Name: "ComplexMessage",
		Fields: []*schema.Field{
			{
				Name:   "ID",
				Number: 1,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeInt32,
				},
				Label:    schema.LabelOptional,
				JsonName: "id",
			},
			{
				Name:   "NAME",
				Number: 2,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeString,
				},
				JsonName: "name",
			},
			{
				Name:   "METADATA",
				Number: 4,
				Type: schema.FieldType{
					Kind:        schema.KindMessage,
					MessageType: "Metadata",
				},
				JsonName: "metadata",
			},
		},
	}

	metadataMessage := &schema.Message{
		Name: "Metadata",
		Fields: []*schema.Field{
			{
				Name:   "CREATED_AT",
				Number: 1,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeString,
				},
				JsonName: "created_at",
			},
			{
				Name:   "VERSION",
				Number: 2,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeInt32,
				},
				JsonName: "version",
			},
		},
	}

	// Create nested metadata
	metadata := map[string]interface{}{
		"created_at": "2023-01-01T00:00:00Z",
		"version":    int32(1),
	}
	metadataBytes, err := EncodeMessage(metadata, metadataMessage, nil)
	if err != nil {
		t.Fatalf("Failed to encode metadata: %v", err)
	}

	// Create complex test data
	testData := map[string]interface{}{
		"id":       int32(12345),
		"name":     "Complex Test",
		"metadata": metadataBytes,
	}

	// Encode
	encodedData, err := EncodeMessage(testData, complexMessage, nil)
	if err != nil {
		t.Fatalf("Failed to encode complex message: %v", err)
	}

	// Decode
	decodedDataI, err := DecodeMessage(encodedData, complexMessage, nil)
	if err != nil {
		t.Fatalf("Failed to decode complex message: %v", err)
	}
	decodedData, ok := decodedDataI.(map[string]interface{})
	if !ok {
		t.Fatalf("decoded data must be of type map[string]interface{} , got: %T", decodedDataI)
	}
	// Verify primitive fields
	if decodedData["id"] != int32(12345) {
		t.Errorf("Expected id=12345, got %v", decodedData["id"])
	}
	if decodedData["name"] != "Complex Test" {
		t.Errorf("Expected name='Complex Test', got %v", decodedData["name"])
	}

	// Verify nested message
	metadataBytes2, ok := decodedData["metadata"].([]byte)
	if !ok {
		t.Errorf("Expected metadata to be []byte, got %T", decodedData["metadata"])
	} else {
		metadataDecodedI, err := DecodeMessage(metadataBytes2, metadataMessage, nil)
		if err != nil {
			t.Fatalf("Failed to decode metadata: %v", err)
		}
		metadataDecoded, ok := metadataDecodedI.(map[string]interface{})
		if !ok {
			t.Fatalf("decoded data must be of type map[string]interface{} , got: %T", decodedDataI)
		}
		if metadataDecoded["created_at"] != "2023-01-01T00:00:00Z" {
			t.Errorf("Expected created_at='2023-01-01T00:00:00Z', got %v", metadataDecoded["created_at"])
		}
		if metadataDecoded["version"] != int32(1) {
			t.Errorf("Expected version=1, got %v", metadataDecoded["version"])
		}
	}
}
