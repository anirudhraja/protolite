package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	"github.com/protolite"
	"github.com/protolite/schema"
)

func main() {
	fmt.Println("🚀 Testing protolite with nested User -> Posts structure...")

	// Create the protolite instance
	p := protolite.New()

	// Load schema with nested structures
	err := p.LoadRepo(createUserPostsSchema())
	if err != nil {
		log.Fatal("Failed to load schema:", err)
	}

	fmt.Printf("✅ Loaded messages: %v\n", p.ListMessages())

	// Test 1: Marshal nested data
	fmt.Println("\n📤 Test 1: Marshal nested User with Posts...")
	userData := createSampleUserData()
	fmt.Printf("Input data: %+v\n", userData)

	protobufBytes, err := p.Marshal(userData, "User")
	if err != nil {
		log.Fatal("Failed to marshal:", err)
	}

	fmt.Printf("✅ Marshaled %d bytes: %s\n", len(protobufBytes), hex.EncodeToString(protobufBytes))

	// Test 2: Parse the marshaled data back
	fmt.Println("\n📥 Test 2: Parse protobuf bytes back to map...")
	result, err := p.Parse(protobufBytes, "User")
	if err != nil {
		log.Fatal("Failed to parse:", err)
	}

	fmt.Printf("✅ Parsed result:\n")
	printParsedUser(result)

	// Test 3: Test with hand-crafted protobuf bytes
	fmt.Println("\n📥 Test 3: Parse hand-crafted protobuf bytes...")
	handCraftedBytes := createHandCraftedProtobuf()
	fmt.Printf("Hand-crafted bytes: %s\n", hex.EncodeToString(handCraftedBytes))

	result2, err := p.Parse(handCraftedBytes, "User")
	if err != nil {
		log.Fatal("Failed to parse hand-crafted bytes:", err)
	}

	fmt.Printf("✅ Parsed hand-crafted result:\n")
	printParsedUser(result2)

	// Test 4: Round-trip test
	fmt.Println("\n🔄 Test 4: Round-trip test...")
	roundTripBytes, err := p.Marshal(result, "User")
	if err != nil {
		log.Fatal("Failed round-trip marshal:", err)
	}

	fmt.Printf("✅ Round-trip comparison:\n")
	fmt.Printf("   Original:   %s\n", hex.EncodeToString(protobufBytes))
	fmt.Printf("   Round-trip: %s\n", hex.EncodeToString(roundTripBytes))

	// Test 5: Test with missing schema (should handle gracefully)
	fmt.Println("\n⚠️  Test 5: Parse with unknown message type...")
	_, err = p.Parse(protobufBytes, "UnknownMessage")
	if err != nil {
		fmt.Printf("✅ Expected error for unknown message: %v\n", err)
	}

	fmt.Println("\n🎉 All basic tests completed!")

	// Test 6: Refactored Map Support Demo
	fmt.Println("\n" + strings.Repeat("=", 60))
	testRefactoredMaps()
}

// createUserPostsSchema creates a schema with User containing nested Posts
func createUserPostsSchema() *schema.ProtoRepo {
	// Define Post message
	postMessage := &schema.Message{
		Name: "Post",
		Fields: []*schema.Field{
			{
				Name:   "id",
				Number: 1,
				Label:  schema.LabelOptional,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeInt64,
				},
			},
			{
				Name:   "title",
				Number: 2,
				Label:  schema.LabelOptional,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeString,
				},
			},
			{
				Name:   "content",
				Number: 3,
				Label:  schema.LabelOptional,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeString,
				},
			},
			{
				Name:   "published",
				Number: 4,
				Label:  schema.LabelOptional,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeBool,
				},
			},
		},
	}

	// Define User message with nested Posts
	userMessage := &schema.Message{
		Name: "User",
		Fields: []*schema.Field{
			{
				Name:   "id",
				Number: 1,
				Label:  schema.LabelOptional,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeInt32,
				},
			},
			{
				Name:   "name",
				Number: 2,
				Label:  schema.LabelOptional,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeString,
				},
			},
			{
				Name:   "email",
				Number: 3,
				Label:  schema.LabelOptional,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeString,
				},
			},
			{
				Name:   "profile",
				Number: 4,
				Label:  schema.LabelOptional,
				Type: schema.FieldType{
					Kind:        schema.KindMessage,
					MessageType: "UserProfile",
				},
			},
			// Note: This is a simplified representation of repeated fields
			// In a full implementation, we'd need to handle repeated fields properly
			{
				Name:   "latest_post",
				Number: 5,
				Label:  schema.LabelOptional,
				Type: schema.FieldType{
					Kind:        schema.KindMessage,
					MessageType: "Post",
				},
			},
		},
	}

	// Define UserProfile message
	userProfileMessage := &schema.Message{
		Name: "UserProfile",
		Fields: []*schema.Field{
			{
				Name:   "bio",
				Number: 1,
				Label:  schema.LabelOptional,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeString,
				},
			},
			{
				Name:   "avatar_url",
				Number: 2,
				Label:  schema.LabelOptional,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeString,
				},
			},
		},
	}

	// Create the repository
	return &schema.ProtoRepo{
		ProtoFiles: map[string]*schema.ProtoFile{
			"user.proto": {
				Name:     "user.proto",
				Package:  "example",
				Syntax:   "proto3",
				Messages: []*schema.Message{userMessage, postMessage, userProfileMessage},
			},
		},
	}
}

// createSampleUserData creates sample nested user data
func createSampleUserData() map[string]interface{} {
	return map[string]interface{}{
		"id":    int32(42),
		"name":  "Alice Johnson",
		"email": "alice@example.com",
		"profile": map[string]interface{}{
			"bio":        "Software engineer and blogger",
			"avatar_url": "https://example.com/avatars/alice.jpg",
		},
		"latest_post": map[string]interface{}{
			"id":        int64(1001),
			"title":     "Getting Started with Protobuf",
			"content":   "Protobuf is a language-neutral, platform-neutral...",
			"published": true,
		},
	}
}

// createHandCraftedProtobuf creates some hand-crafted protobuf bytes for testing
func createHandCraftedProtobuf() []byte {
	// This represents a simple User message with:
	// field 1 (id): varint 123
	// field 2 (name): string "Bob"
	// field 3 (email): string "bob@test.com"

	// Manually crafted protobuf bytes:
	// 08 7B = field 1, varint 123 (0x7B = 123)
	// 12 03 42 6F 62 = field 2, length 3, "Bob"
	// 1A 0C 62 6F 62 40 74 65 73 74 2E 63 6F 6D = field 3, length 12, "bob@test.com"

	bytes, _ := hex.DecodeString("087B1203426F621A0C626F6240746573742E636F6D")
	return bytes
}

// printParsedUser prints a parsed user in a nice format
func printParsedUser(user map[string]interface{}) {
	fmt.Printf("  User ID: %v (type: %T)\n", user["id"], user["id"])
	fmt.Printf("  Name: %v (type: %T)\n", user["name"], user["name"])
	fmt.Printf("  Email: %v (type: %T)\n", user["email"], user["email"])

	if profile, ok := user["profile"].(map[string]interface{}); ok {
		fmt.Printf("  Profile:\n")
		fmt.Printf("    Bio: %v (type: %T)\n", profile["bio"], profile["bio"])
		fmt.Printf("    Avatar: %v (type: %T)\n", profile["avatar_url"], profile["avatar_url"])
	}

	if post, ok := user["latest_post"].(map[string]interface{}); ok {
		fmt.Printf("  Latest Post:\n")
		fmt.Printf("    ID: %v (type: %T)\n", post["id"], post["id"])
		fmt.Printf("    Title: %v (type: %T)\n", post["title"], post["title"])
		fmt.Printf("    Content: %v (type: %T)\n", post["content"], post["content"])
		fmt.Printf("    Published: %v (type: %T)\n", post["published"], post["published"])
	}
}
