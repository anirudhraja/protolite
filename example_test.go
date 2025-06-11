package protolite

import (
	"fmt"
	"log"

	"github.com/protolite/schema"
	"github.com/protolite/wire"
)

// Example demonstrates the Protolite API usage
func ExampleProtolite() {
	// Create a new Protolite instance
	proto := NewProtolite()

	// Example 1: Schema-less parsing
	fmt.Println("=== Schema-less Parsing ===")

	// Create some protobuf data manually for demonstration
	encoder := wire.NewEncoder()
	ve := wire.NewVarintEncoder(encoder)
	be := wire.NewBytesEncoder(encoder)

	// Encode: field 1 = varint 123, field 2 = string "hello"
	tag1 := wire.MakeTag(wire.FieldNumber(1), wire.WireVarint)
	ve.EncodeVarint(uint64(tag1))
	ve.EncodeVarint(123)

	tag2 := wire.MakeTag(wire.FieldNumber(2), wire.WireBytes)
	ve.EncodeVarint(uint64(tag2))
	be.EncodeString("hello")

	// Parse without schema
	result, err := proto.Parse(encoder.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Parsed fields: %+v\n", result)
	// Output: field_1: {type: varint, value: 123}, field_2: {type: bytes, value: [104 101 108 108 111]}

	// Example 2: Struct unmarshaling with reflection
	fmt.Println("\n=== Struct Unmarshaling ===")

	type User struct {
		ID     int32  `json:"id"`
		Name   string `json:"name"`
		Email  string `json:"email"`
		Active bool   `json:"active"`
	}

	// Create test data
	userData := map[string]interface{}{
		"id":     int32(12345),
		"name":   "John Doe",
		"email":  "john.doe@example.com",
		"active": true,
	}

	// Use reflection to populate a struct
	protoImpl := &protolite{}
	var user User
	err = protoImpl.mapToStruct(userData, &user)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("User struct: %+v\n", user)
	// Output: {ID:12345 Name:John Doe Email:john.doe@example.com Active:true}

	// Example 3: Schema-based operations with proper methods
	fmt.Println("\n=== Schema-based Operations ===")

	// Note: These require schemas to be loaded first
	fmt.Println("Schema-based methods require LoadSchemaFromFile() first")
	fmt.Println("Available methods:")
	fmt.Println("- proto.MarshalWithSchema(data, messageName)")
	fmt.Println("- proto.UnmarshalWithSchema(data, messageName)")
	fmt.Println("- proto.UnmarshalToStruct(data, messageName, &struct)")
	fmt.Println("- proto.LoadSchemaFromFile(protoPath)")

	fmt.Println("\n=== Protolite API Demo Complete ===")

	// Output:
	// === Schema-less Parsing ===
	// Parsed fields: map[field_1:map[type:varint value:123] field_2:map[type:bytes value:[104 101 108 108 111]]]
	//
	// === Struct Unmarshaling ===
	// User struct: {ID:12345 Name:John Doe Email:john.doe@example.com Active:true}
	//
	// === Schema-based Operations ===
	// Schema-based methods require LoadSchemaFromFile() first
	// Available methods:
	// - proto.MarshalWithSchema(data, messageName)
	// - proto.UnmarshalWithSchema(data, messageName)
	// - proto.UnmarshalToStruct(data, messageName, &struct)
	// - proto.LoadSchemaFromFile(protoPath)
	//
	// === Protolite API Demo Complete ===
}

// Example demonstrates encoding and parsing with schemas
func ExampleProtolite_withSchema() {
	fmt.Println("=== Schema-based Operations ===")

	// Define a message schema
	userMessage := &schema.Message{
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

	// Create test data
	userData := map[string]interface{}{
		"id":     int32(999),
		"name":   "Jane Smith",
		"active": true,
	}

	// Encode using wire functions (direct schema-based encoding)
	encodedData, err := wire.EncodeMessage(userData, userMessage, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Encoded data: %v bytes\n", len(encodedData))

	// Parse using schema-less parsing
	proto := NewProtolite()
	parsedData, err := proto.Parse(encodedData)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Parsed data: %+v\n", parsedData)

	// Decode back using schema
	decodedData, err := wire.DecodeMessage(encodedData, userMessage, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Decoded with schema: %+v\n", decodedData)

	fmt.Println("=== Schema-based Demo Complete ===")

	// Output:
	// === Schema-based Operations ===
	// Encoded data: 17 bytes
	// Parsed data: map[field_1:map[type:varint value:999] field_2:map[type:bytes value:[74 97 110 101 32 83 109 105 116 104]] field_3:map[type:varint value:1]]
	// Decoded with schema: map[active:true id:999 name:Jane Smith]
	// === Schema-based Demo Complete ===
}
