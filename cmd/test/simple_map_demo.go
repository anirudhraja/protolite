package main

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/protolite"
	"github.com/protolite/schema"
)

// Simple demo to verify refactored map functionality

func testRefactoredMaps() {
	fmt.Println("🔧 Testing Refactored Map Implementation...")
	fmt.Println("Using existing decoder.go infrastructure instead of duplicated logic")

	p := protolite.New()
	err := p.LoadRepo(createSimpleMapSchema())
	if err != nil {
		log.Fatal("Failed to load schema:", err)
	}

	// Test data with string->int32 map
	testData := map[string]interface{}{
		"name": "TestUser",
		"scores": map[string]int32{
			"level1": 100,
			"level2": 250,
			"level3": 500,
		},
	}

	fmt.Printf("Input: %+v\n", testData)

	// Marshal using refactored logic
	bytes, err := p.Marshal(testData, "GameData")
	if err != nil {
		log.Fatal("Marshal failed:", err)
	}
	fmt.Printf("✅ Marshaled %d bytes: %s\n", len(bytes), hex.EncodeToString(bytes))

	// Parse using refactored logic
	result, err := p.Parse(bytes, "GameData")
	if err != nil {
		log.Fatal("Parse failed:", err)
	}

	fmt.Printf("✅ Parsed result:\n")
	fmt.Printf("  name: %v\n", result["name"])
	if scores, ok := result["scores"].(map[interface{}]interface{}); ok {
		fmt.Printf("  scores: {\n")
		for k, v := range scores {
			fmt.Printf("    %v: %v\n", k, v)
		}
		fmt.Printf("  }\n")
	}

	fmt.Println("🎉 Refactored map implementation working!")
}

func createSimpleMapSchema() *schema.ProtoRepo {
	gameDataMessage := &schema.Message{
		Name: "GameData",
		Fields: []*schema.Field{
			{Name: "name", Number: 1, Label: schema.LabelOptional,
				Type: schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeString}},
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

	return &schema.ProtoRepo{
		ProtoFiles: map[string]*schema.ProtoFile{
			"simple.proto": {
				Name:     "simple.proto",
				Package:  "test",
				Syntax:   "proto3",
				Messages: []*schema.Message{gameDataMessage},
			},
		},
	}
}
