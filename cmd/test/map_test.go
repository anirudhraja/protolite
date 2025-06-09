package main

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/protolite"
	"github.com/protolite/schema"
)

// Demo: Map Type Support in protolite

func testMapSupport() {
	fmt.Println("🗺️  Testing Map Support in protolite...")

	// Create protolite instance
	p := protolite.New()

	// Load schema with various map types
	err := p.LoadRepo(createMapSchema())
	if err != nil {
		log.Fatal("Failed to load schema:", err)
	}

	fmt.Printf("✅ Loaded messages: %v\n", p.ListMessages())

	// =============================================================
	// TEST 1: String -> String Map
	// =============================================================
	fmt.Println("\n📋 TEST 1: String -> String Map")

	stringMapData := map[string]interface{}{
		"id":   int32(1),
		"name": "John Doe",
		"attributes": map[string]string{
			"country":    "USA",
			"language":   "English",
			"timezone":   "UTC-5",
			"department": "Engineering",
		},
	}

	fmt.Printf("Input data: %+v\n", stringMapData)

	// Marshal
	bytes1, err := p.Marshal(stringMapData, "UserProfile")
	if err != nil {
		log.Fatal("Failed to marshal string map:", err)
	}
	fmt.Printf("✅ Marshaled %d bytes: %s\n", len(bytes1), hex.EncodeToString(bytes1))

	// Parse back
	result1, err := p.Parse(bytes1, "UserProfile")
	if err != nil {
		log.Fatal("Failed to parse string map:", err)
	}
	fmt.Printf("✅ Parsed result:\n")
	printMapResult(result1)

	// =============================================================
	// TEST 2: String -> Int32 Map
	// =============================================================
	fmt.Println("\n📋 TEST 2: String -> Int32 Map")

	intMapData := map[string]interface{}{
		"game_id": int32(42),
		"scores": map[string]int32{
			"player1": 1500,
			"player2": 1200,
			"player3": 1800,
			"player4": 950,
		},
	}

	fmt.Printf("Input data: %+v\n", intMapData)

	// Marshal
	bytes2, err := p.Marshal(intMapData, "GameSession")
	if err != nil {
		log.Fatal("Failed to marshal int map:", err)
	}
	fmt.Printf("✅ Marshaled %d bytes: %s\n", len(bytes2), hex.EncodeToString(bytes2))

	// Parse back
	result2, err := p.Parse(bytes2, "GameSession")
	if err != nil {
		log.Fatal("Failed to parse int map:", err)
	}
	fmt.Printf("✅ Parsed result:\n")
	printMapResult(result2)

	// =============================================================
	// TEST 3: Int32 -> String Map
	// =============================================================
	fmt.Println("\n📋 TEST 3: Int32 -> String Map")

	idMapData := map[string]interface{}{
		"service": "UserService",
		"user_names": map[int32]interface{}{
			1001: "Alice Johnson",
			1002: "Bob Smith",
			1003: "Charlie Brown",
			1004: "Diana Prince",
		},
	}

	fmt.Printf("Input data: %+v\n", idMapData)

	// Marshal
	bytes3, err := p.Marshal(idMapData, "UserDirectory")
	if err != nil {
		log.Fatal("Failed to marshal id map:", err)
	}
	fmt.Printf("✅ Marshaled %d bytes: %s\n", len(bytes3), hex.EncodeToString(bytes3))

	// Parse back
	result3, err := p.Parse(bytes3, "UserDirectory")
	if err != nil {
		log.Fatal("Failed to parse id map:", err)
	}
	fmt.Printf("✅ Parsed result:\n")
	printMapResult(result3)

	// =============================================================
	// TEST 4: Round-trip Verification
	// =============================================================
	fmt.Println("\n📋 TEST 4: Round-trip Verification")

	// Re-marshal the parsed result
	roundTripBytes, err := p.Marshal(result1, "UserProfile")
	if err != nil {
		log.Fatal("Failed round-trip marshal:", err)
	}

	fmt.Printf("Original bytes:   %s\n", hex.EncodeToString(bytes1))
	fmt.Printf("Round-trip bytes: %s\n", hex.EncodeToString(roundTripBytes))

	// Parse round-trip bytes
	roundTripResult, err := p.Parse(roundTripBytes, "UserProfile")
	if err != nil {
		log.Fatal("Failed round-trip parse:", err)
	}

	fmt.Printf("✅ Round-trip verification:\n")
	printMapResult(roundTripResult)

	fmt.Println("\n🎉 Map functionality tests completed!")
}

func createMapSchema() *schema.ProtoRepo {
	// UserProfile message with string->string map
	userProfileMessage := &schema.Message{
		Name: "UserProfile",
		Fields: []*schema.Field{
			{Name: "id", Number: 1, Label: schema.LabelOptional,
				Type: schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeInt32}},
			{Name: "name", Number: 2, Label: schema.LabelOptional,
				Type: schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeString}},
			{Name: "attributes", Number: 3, Label: schema.LabelRepeated,
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
				}},
		},
	}

	// GameSession message with string->int32 map
	gameSessionMessage := &schema.Message{
		Name: "GameSession",
		Fields: []*schema.Field{
			{Name: "game_id", Number: 1, Label: schema.LabelOptional,
				Type: schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeInt32}},
			{Name: "scores", Number: 2, Label: schema.LabelRepeated,
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
				}},
		},
	}

	// UserDirectory message with int32->string map
	userDirectoryMessage := &schema.Message{
		Name: "UserDirectory",
		Fields: []*schema.Field{
			{Name: "service", Number: 1, Label: schema.LabelOptional,
				Type: schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeString}},
			{Name: "user_names", Number: 2, Label: schema.LabelRepeated,
				Type: schema.FieldType{
					Kind: schema.KindMap,
					MapKey: &schema.FieldType{
						Kind:          schema.KindPrimitive,
						PrimitiveType: schema.TypeInt32,
					},
					MapValue: &schema.FieldType{
						Kind:          schema.KindPrimitive,
						PrimitiveType: schema.TypeString,
					},
				}},
		},
	}

	return &schema.ProtoRepo{
		ProtoFiles: map[string]*schema.ProtoFile{
			"maps.proto": {
				Name:     "maps.proto",
				Package:  "example",
				Syntax:   "proto3",
				Messages: []*schema.Message{userProfileMessage, gameSessionMessage, userDirectoryMessage},
			},
		},
	}
}

func printMapResult(result map[string]interface{}) {
	for key, value := range result {
		if mapValue, ok := value.(map[interface{}]interface{}); ok {
			fmt.Printf("  %s (map): {\n", key)
			for k, v := range mapValue {
				fmt.Printf("    %v: %v (types: %T -> %T)\n", k, v, k, v)
			}
			fmt.Printf("  }\n")
		} else {
			fmt.Printf("  %s: %v (type: %T)\n", key, value, value)
		}
	}
}
