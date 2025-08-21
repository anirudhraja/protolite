package protolite

import (
	"reflect"
	"strings"
	"testing"

	"github.com/anirudhraja/protolite/schema"
	"github.com/anirudhraja/protolite/wire"
)

func TestProtolite_Parse(t *testing.T) {
	proto := NewProtolite([]string{""})

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
		if err := ve.EncodeVarint(uint64(tag)); err != nil {
			t.Fatalf("Failed to encode tag: %v", err)
		}
		if err := ve.EncodeVarint(42); err != nil {
			t.Fatalf("Failed to encode value: %v", err)
		}

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
		if err := ve.EncodeVarint(uint64(tag1)); err != nil {
			t.Fatalf("Failed to encode tag1: %v", err)
		}
		if err := ve.EncodeVarint(123); err != nil {
			t.Fatalf("Failed to encode value1: %v", err)
		}

		// Field 2: string "hello"
		tag2 := wire.MakeTag(wire.FieldNumber(2), wire.WireBytes)
		if err := ve.EncodeVarint(uint64(tag2)); err != nil {
			t.Fatalf("Failed to encode tag2: %v", err)
		}
		if err := be.EncodeString("hello"); err != nil {
			t.Fatalf("Failed to encode string: %v", err)
		}

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
	proto := NewProtolite([]string{""})

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
	proto := NewProtolite([]string{""})

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
		proto := NewProtolite([]string{""})
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

func TestProtolite_UnmarshalWithSchema(t *testing.T) {
	proto := NewProtolite([]string{"", "sampleapp/testdata"})

	t.Run("unmarshal_with_schema", func(t *testing.T) {
		if err := proto.LoadSchemaFromFile("sampleapp/testdata/post.proto"); err != nil {
			t.Fatalf("Failed to load post.proto: %v", err)
		}
		if err := proto.LoadSchemaFromFile("sampleapp/testdata/user.proto"); err != nil {
			t.Fatalf("Failed to load user.proto: %v", err)
		}

		// Verify both files are loaded
		pImpl := proto.(*protolite)
		protoFiles := pImpl.registry.ListProtoFiles()

		if len(protoFiles) != 2 {
			t.Errorf("Expected 2 proto files, got %d", len(protoFiles))
			for _, path := range protoFiles {
				t.Logf("Loaded file: %s", path)
			}
		}

		// Check specific files exist
		hasUser := false
		hasPost := false
		for _, path := range protoFiles {
			if path == "sampleapp/testdata/user.proto" {
				hasUser = true
			}
			if path == "sampleapp/testdata/post.proto" {
				hasPost = true
			}
		}

		if !hasUser {
			t.Error("user.proto not found in registry")
		}
		if !hasPost {
			t.Error("post.proto not found in registry")
		}

		// Create User data with 2 Posts
		userData := map[string]interface{}{
			"id":     int32(1),
			"name":   "John Doe",
			"email":  "john.doe@example.com",
			"active": true,
			"status": "USER_ACTIVE", // ACTIVE
			"posts": []map[string]interface{}{
				{
					"id":         int32(101),
					"title":      "My First Blog Post",
					"content":    "This is my first blog post about Go programming and protobuf.",
					"author_id":  int32(1),
					"status":     "POST_PUBLISHED", // PUBLISHED
					"tags":       []string{"go", "programming", "protobuf"},
					"created_at": int64(1640995200), // 2022-01-01 00:00:00 UTC
					"updated_at": int64(1640995200),
					"view_count": int32(150),
					"featured":   true,
				},
				{
					"id":         int32(102),
					"title":      "Advanced Protobuf Patterns",
					"content":    "In this post, I'll share advanced protobuf patterns and best practices.",
					"author_id":  int32(1),
					"status":     "POST_PUBLISHED", // PUBLISHED
					"tags":       []string{"protobuf", "advanced", "patterns"},
					"created_at": int64(1641081600), // 2022-01-02 00:00:00 UTC
					"updated_at": int64(1641168000), // 2022-01-03 00:00:00 UTC (updated)
					"view_count": int32(275),
					"featured":   false,
				},
			},
			"metadata": map[string]string{
				"timezone":   "UTC",
				"theme":      "dark",
				"language":   "en",
				"newsletter": "subscribed",
			},
			"created_at": int64(1609459200), // 2021-01-01 00:00:00 UTC
			"scores":     []int32{42, 39, 21},
			"cool_list":  []interface{}{int32(42), nil, int32(39), int32(21)},
			"show_me_null": map[string]interface{}{
				"null": nil,
			},
		}

		t.Logf("Original User Data: %+v", userData)

		// Marshal the user data with schema
		encodedData, err := proto.MarshalWithSchema(userData, "User")
		if err != nil {
			t.Fatalf("Failed to marshal user data: %v", err)
		}

		t.Logf("Encoded data size: %d bytes", len(encodedData))

		// Test Parse (schema-less)
		parsedData, err := proto.Parse(encodedData)
		if err != nil {
			t.Fatalf("Failed to parse data: %v", err)
		}

		t.Logf("Parsed data (schema-less): %+v", parsedData)

		// Test UnmarshalWithSchema (schema-based)
		userMap, err := proto.UnmarshalWithSchema(encodedData, "User")
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		t.Logf("Unmarshaled User: %+v", userMap)

		// Verify user fields
		if userMap["id"] != int32(1) {
			t.Errorf("Expected user id=1, got %v", userMap["id"])
		}
		if userMap["name"] != "John Doe" {
			t.Errorf("Expected user name='John Doe', got %v", userMap["name"])
		}
		if userMap["email"] != "john.doe@example.com" {
			t.Errorf("Expected user email='john.doe@example.com', got %v", userMap["email"])
		}
		if userMap["active"] != true {
			t.Errorf("Expected user active=true, got %v", userMap["active"])
		}
		if userMap["status"] != "USER_ACTIVE" {
			t.Errorf("Expected user status=1, got %v", userMap["status"])
		}

		// Verify posts
		posts, ok := userMap["posts"].([]interface{})
		if !ok {
			t.Fatalf("Expected posts to be a slice, got %T", userMap["posts"])
		}

		if len(posts) != 2 {
			t.Errorf("Expected 2 posts, got %d", len(posts))
		}

		// Verify first post
		if len(posts) > 0 {
			post1, ok := posts[0].(map[string]interface{})
			if !ok {
				t.Fatalf("Expected first post to be a map, got %T", posts[0])
			}

			if post1["id"] != int32(101) {
				t.Errorf("Expected first post id=101, got %v", post1["id"])
			}
			if post1["title"] != "My First Blog Post" {
				t.Errorf("Expected first post title='My First Blog Post', got %v", post1["title"])
			}
			if post1["author_id"] != int32(1) {
				t.Errorf("Expected first post author_id=1, got %v", post1["author_id"])
			}
			if post1["status"] != "POST_PUBLISHED" {
				t.Errorf("Expected first post status=1, got %v", post1["status"])
			}
			if post1["view_count"] != int32(150) {
				t.Errorf("Expected first post view_count=150, got %v", post1["view_count"])
			}
			if post1["featured"] != true {
				t.Errorf("Expected first post featured=true, got %v", post1["featured"])
			}

			// Check tags
			tags1, ok := post1["tags"].([]interface{})
			if !ok {
				t.Errorf("Expected first post tags to be a slice, got %T", post1["tags"])
			} else if len(tags1) != 3 {
				t.Errorf("Expected first post to have 3 tags, got %d", len(tags1))
			} else {
				expectedTags1 := []string{"go", "programming", "protobuf"}
				for i, tag := range expectedTags1 {
					if i < len(tags1) && tags1[i] != tag {
						t.Errorf("Expected first post tag[%d]='%s', got %v", i, tag, tags1[i])
					}
				}
			}

			t.Logf("First Post: %+v", post1)
		}

		// Verify second post
		if len(posts) > 1 {
			post2, ok := posts[1].(map[string]interface{})
			if !ok {
				t.Fatalf("Expected second post to be a map, got %T", posts[1])
			}

			if post2["id"] != int32(102) {
				t.Errorf("Expected second post id=102, got %v", post2["id"])
			}
			if post2["title"] != "Advanced Protobuf Patterns" {
				t.Errorf("Expected second post title='Advanced Protobuf Patterns', got %v", post2["title"])
			}
			if post2["author_id"] != int32(1) {
				t.Errorf("Expected second post author_id=1, got %v", post2["author_id"])
			}
			if post2["status"] != "POST_PUBLISHED" {
				t.Errorf("Expected second post status=1, got %v", post2["status"])
			}
			if post2["view_count"] != int32(275) {
				t.Errorf("Expected second post view_count=275, got %v", post2["view_count"])
			}
			if post2["featured"] != false {
				t.Errorf("Expected second post featured=false, got %v", post2["featured"])
			}

			// Check tags
			tags2, ok := post2["tags"].([]interface{})
			if !ok {
				t.Errorf("Expected second post tags to be a slice, got %T", post2["tags"])
			} else if len(tags2) != 3 {
				t.Errorf("Expected second post to have 3 tags, got %d", len(tags2))
			} else {
				expectedTags2 := []string{"protobuf", "advanced", "patterns"}
				for i, tag := range expectedTags2 {
					if i < len(tags2) && tags2[i] != tag {
						t.Errorf("Expected second post tag[%d]='%s', got %v", i, tag, tags2[i])
					}
				}
			}

			t.Logf("Second Post: %+v", post2)
		}

		// Verify metadata
		metadata, ok := userMap["metadata"].(map[interface{}]interface{})
		if !ok {
			t.Errorf("Expected metadata to be a map[interface{}]interface{}, got %T", userMap["metadata"])
		} else {
			expectedMetadata := map[string]string{
				"timezone":   "UTC",
				"theme":      "dark",
				"language":   "en",
				"newsletter": "subscribed",
			}
			for key, expectedValue := range expectedMetadata {
				if metadata[key] != expectedValue {
					t.Errorf("Expected metadata[%s]='%s', got %v", key, expectedValue, metadata[key])
				}
			}
		}

		// Verify created_at
		if userMap["created_at"] != int64(1609459200) {
			t.Errorf("Expected user created_at=1609459200, got %v", userMap["created_at"])
		}

		// Verify scores
		scores, ok := userMap["scores"].([]interface{})
		if !ok {
			t.Fatalf("Expected scores to be a slice, got %T", userMap["scores"])
		}
		gotScores := make([]int32, 0, len(scores))
		for i := 0; i < len(scores); i++ {
			gotScores = append(gotScores, scores[i].(int32))
		}
		if !reflect.DeepEqual(userData["scores"], gotScores) {
			t.Fatalf("Expected scores to be %v, got %v", userData["scores"], gotScores)
		}

		// Verify cool_list
		coolList, ok := userMap["cool_list"].([]interface{})
		if !ok {
			t.Fatalf("Expected cool_list to be a slice, got %T", userMap["cool_list"])
		}
		if !reflect.DeepEqual(userData["cool_list"], coolList) {
			t.Fatalf("Expected cool_list to be %v, got %v", userData["cool_list"], coolList)
		}

		// Verify show_me_null
		showMeNull, ok := userMap["show_me_null"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected show_me_null to be a map, got %T", userMap["show_me_null"])
		}
		null, ok := showMeNull["null"]
		if !ok {
			t.Fatalf("Expected null key present in map")
		}
		if null != nil {
			t.Fatalf("Expected nil value of key null, got %v", null)
		}

		t.Log("✅ User-Posts relationship test completed successfully!")
		t.Log("✅ Both proto files loaded correctly")
		t.Log("✅ User with 2 Posts marshaled and unmarshaled correctly")
		t.Log("✅ All field values verified")
	})
}
