package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/protolite/schema"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Fatal("NewRegistry() returned nil")
	}

	if registry.messages != nil {
		t.Error("Expected messages map to be nil initially")
	}

	if registry.enums != nil {
		t.Error("Expected enums map to be nil initially")
	}

	if registry.services != nil {
		t.Error("Expected services map to be nil initially")
	}
}

func TestLoadSchema_NonExistentPath(t *testing.T) {
	registry := NewRegistry()

	err := registry.LoadSchema("/nonexistent/path")
	if err == nil {
		t.Error("Expected error for non-existent path")
	}

	if err != nil && !contains(err.Error(), "path does not exist") {
		t.Errorf("Expected 'path does not exist' error, got: %v", err)
	}
}

func TestLoadSchema_NonProtoFile(t *testing.T) {
	// Create a temporary non-proto file
	tmpFile, err := os.CreateTemp("", "test*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	registry := NewRegistry()
	err = registry.LoadSchema(tmpFile.Name())

	if err == nil {
		t.Error("Expected error for non-proto file")
	}

	if err != nil && !contains(err.Error(), "is not a .proto file") {
		t.Errorf("Expected 'is not a .proto file' error, got: %v", err)
	}
}

func TestLoadSchema_SingleProtoFile(t *testing.T) {
	// Create a temporary proto file
	tmpDir, err := os.MkdirTemp("", "proto_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	protoContent := `syntax = "proto3";
package test.package;

message TestMessage {
  string name = 1;
  int32 id = 2;
}

enum TestEnum {
  UNKNOWN = 0;
  ACTIVE = 1;
}

service TestService {
  rpc GetTest(TestMessage) returns (TestMessage);
}
`

	protoFile := filepath.Join(tmpDir, "test.proto")
	err = os.WriteFile(protoFile, []byte(protoContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	registry := NewRegistry()
	err = registry.LoadSchema(protoFile)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// Verify the proto file was loaded
	if registry.repo == nil {
		t.Fatal("ProtoRepo is nil")
	}

	if len(registry.repo.ProtoFiles) != 1 {
		t.Errorf("Expected 1 proto file, got %d", len(registry.repo.ProtoFiles))
	}

	protoFileData := registry.repo.ProtoFiles[protoFile]
	if protoFileData == nil {
		t.Fatal("Proto file data is nil")
	}

	if protoFileData.Package != "test.package" {
		t.Errorf("Expected package 'test.package', got '%s'", protoFileData.Package)
	}

	if protoFileData.Syntax != "proto3" {
		t.Errorf("Expected syntax 'proto3', got '%s'", protoFileData.Syntax)
	}
}

func TestLoadSchema_Directory(t *testing.T) {
	// Create a temporary directory with multiple proto files
	tmpDir, err := os.MkdirTemp("", "proto_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create proto files
	protoFiles := map[string]string{
		filepath.Join(tmpDir, "file1.proto"): `syntax = "proto3";
package pkg1;`,
		filepath.Join(subDir, "file2.proto"): `syntax = "proto2";
package pkg2;`,
		filepath.Join(tmpDir, "notproto.txt"): "not a proto file",
	}

	for path, content := range protoFiles {
		err = os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	registry := NewRegistry()
	err = registry.LoadSchema(tmpDir)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// Should have loaded 2 proto files, ignoring the .txt file
	if len(registry.repo.ProtoFiles) != 2 {
		t.Errorf("Expected 2 proto files, got %d", len(registry.repo.ProtoFiles))
	}
}

func TestGetFullName(t *testing.T) {
	registry := NewRegistry()

	tests := []struct {
		pkg      string
		name     string
		expected string
	}{
		{"", "Message", "Message"},
		{"pkg", "Message", "pkg.Message"},
		{"com.example", "Message", "com.example.Message"},
	}

	for _, test := range tests {
		result := registry.getFullName(test.pkg, test.name)
		if result != test.expected {
			t.Errorf("getFullName(%q, %q) = %q, expected %q",
				test.pkg, test.name, result, test.expected)
		}
	}
}

func TestGetMessage_NotFound(t *testing.T) {
	registry := NewRegistry()
	registry.messages = make(map[string]*schema.Message)

	_, err := registry.GetMessage("NonExistent")
	if err == nil {
		t.Error("Expected error for non-existent message")
	}

	if !contains(err.Error(), "message not found") {
		t.Errorf("Expected 'message not found' error, got: %v", err)
	}
}

func TestGetMessage_Found(t *testing.T) {
	registry := NewRegistry()
	registry.messages = make(map[string]*schema.Message)

	testMessage := &schema.Message{
		Name: "TestMessage",
		Fields: []*schema.Field{
			{Name: "field1", Number: 1},
		},
	}

	registry.messages["pkg.TestMessage"] = testMessage

	// Test exact match
	msg, err := registry.GetMessage("pkg.TestMessage")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if msg != testMessage {
		t.Error("Got wrong message")
	}

	// Test suffix match
	msg, err = registry.GetMessage("TestMessage")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if msg != testMessage {
		t.Error("Got wrong message")
	}
}

func TestGetEnum_NotFound(t *testing.T) {
	registry := NewRegistry()
	registry.enums = make(map[string]*schema.Enum)

	_, err := registry.GetEnum("NonExistent")
	if err == nil {
		t.Error("Expected error for non-existent enum")
	}

	if !contains(err.Error(), "enum not found") {
		t.Errorf("Expected 'enum not found' error, got: %v", err)
	}
}

func TestGetEnum_Found(t *testing.T) {
	registry := NewRegistry()
	registry.enums = make(map[string]*schema.Enum)

	testEnum := &schema.Enum{
		Name: "TestEnum",
		Values: []*schema.EnumValue{
			{Name: "VALUE1", Number: 0},
		},
	}

	registry.enums["pkg.TestEnum"] = testEnum

	// Test exact match
	enum, err := registry.GetEnum("pkg.TestEnum")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if enum != testEnum {
		t.Error("Got wrong enum")
	}

	// Test suffix match
	enum, err = registry.GetEnum("TestEnum")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if enum != testEnum {
		t.Error("Got wrong enum")
	}
}

func TestGetService_NotFound(t *testing.T) {
	registry := NewRegistry()
	registry.services = make(map[string]*schema.Service)

	_, err := registry.GetService("NonExistent")
	if err == nil {
		t.Error("Expected error for non-existent service")
	}

	if !contains(err.Error(), "service not found") {
		t.Errorf("Expected 'service not found' error, got: %v", err)
	}
}

func TestGetService_Found(t *testing.T) {
	registry := NewRegistry()
	registry.services = make(map[string]*schema.Service)

	testService := &schema.Service{
		Name: "TestService",
		Methods: []*schema.Method{
			{Name: "Method1", InputType: "Input", OutputType: "Output"},
		},
	}

	registry.services["pkg.TestService"] = testService

	// Test exact match
	service, err := registry.GetService("pkg.TestService")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if service != testService {
		t.Error("Got wrong service")
	}

	// Test suffix match
	service, err = registry.GetService("TestService")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if service != testService {
		t.Error("Got wrong service")
	}
}

func TestListMessages(t *testing.T) {
	registry := NewRegistry()
	registry.messages = make(map[string]*schema.Message)

	registry.messages["pkg1.Message1"] = &schema.Message{Name: "Message1"}
	registry.messages["pkg2.Message2"] = &schema.Message{Name: "Message2"}

	names := registry.ListMessages()
	if len(names) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(names))
	}

	// Check that all names are present
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}

	if !nameSet["pkg1.Message1"] || !nameSet["pkg2.Message2"] {
		t.Error("Missing expected message names")
	}
}

func TestListEnums(t *testing.T) {
	registry := NewRegistry()
	registry.enums = make(map[string]*schema.Enum)

	registry.enums["pkg1.Enum1"] = &schema.Enum{Name: "Enum1"}
	registry.enums["pkg2.Enum2"] = &schema.Enum{Name: "Enum2"}

	names := registry.ListEnums()
	if len(names) != 2 {
		t.Errorf("Expected 2 enums, got %d", len(names))
	}

	// Check that all names are present
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}

	if !nameSet["pkg1.Enum1"] || !nameSet["pkg2.Enum2"] {
		t.Error("Missing expected enum names")
	}
}

func TestListServices(t *testing.T) {
	registry := NewRegistry()
	registry.services = make(map[string]*schema.Service)

	registry.services["pkg1.Service1"] = &schema.Service{Name: "Service1"}
	registry.services["pkg2.Service2"] = &schema.Service{Name: "Service2"}

	names := registry.ListServices()
	if len(names) != 2 {
		t.Errorf("Expected 2 services, got %d", len(names))
	}

	// Check that all names are present
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}

	if !nameSet["pkg1.Service1"] || !nameSet["pkg2.Service2"] {
		t.Error("Missing expected service names")
	}
}

func TestGetOrCreateMapEntryMessage_Create(t *testing.T) {
	registry := NewRegistry()
	registry.messages = make(map[string]*schema.Message)

	keyType := &schema.FieldType{
		Kind:          schema.KindPrimitive,
		PrimitiveType: schema.TypeString,
	}
	valueType := &schema.FieldType{
		Kind:          schema.KindPrimitive,
		PrimitiveType: schema.TypeInt32,
	}

	msg, err := registry.GetOrCreateMapEntryMessage("TestMap", keyType, valueType)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if msg.Name != "TestMapEntry" {
		t.Errorf("Expected name 'TestMapEntry', got '%s'", msg.Name)
	}

	if !msg.MapEntry {
		t.Error("Expected MapEntry to be true")
	}

	if len(msg.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(msg.Fields))
	}

	// Check key field
	if msg.Fields[0].Name != "key" || msg.Fields[0].Number != 1 {
		t.Error("Invalid key field")
	}

	// Check value field
	if msg.Fields[1].Name != "value" || msg.Fields[1].Number != 2 {
		t.Error("Invalid value field")
	}

	// Verify it was registered
	if registry.messages["TestMapEntry"] != msg {
		t.Error("Map entry message was not registered")
	}
}

func TestGetOrCreateMapEntryMessage_Existing(t *testing.T) {
	registry := NewRegistry()
	registry.messages = make(map[string]*schema.Message)

	existingMsg := &schema.Message{
		Name:     "TestMapEntry",
		MapEntry: true,
	}
	registry.messages["TestMapEntry"] = existingMsg

	keyType := &schema.FieldType{
		Kind:          schema.KindPrimitive,
		PrimitiveType: schema.TypeString,
	}
	valueType := &schema.FieldType{
		Kind:          schema.KindPrimitive,
		PrimitiveType: schema.TypeInt32,
	}

	msg, err := registry.GetOrCreateMapEntryMessage("TestMap", keyType, valueType)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if msg != existingMsg {
		t.Error("Should have returned existing message")
	}
}

func TestRegisterNames(t *testing.T) {
	registry := NewRegistry()
	registry.messages = make(map[string]*schema.Message)
	registry.enums = make(map[string]*schema.Enum)
	registry.services = make(map[string]*schema.Service)

	protoFile := &schema.ProtoFile{
		Package: "test.pkg",
		Messages: []*schema.Message{
			{Name: "Message1"},
			{
				Name: "Message2",
				NestedTypes: []*schema.Message{
					{Name: "NestedMessage"},
				},
				NestedEnums: []*schema.Enum{
					{Name: "NestedEnum"},
				},
			},
		},
		Enums: []*schema.Enum{
			{Name: "Enum1"},
		},
		Services: []*schema.Service{
			{Name: "Service1"},
		},
	}

	err := registry.registerNames(protoFile)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check messages were registered
	expectedMessages := []string{
		"test.pkg.Message1",
		"test.pkg.Message2",
		"test.pkg.Message2.NestedMessage",
	}

	for _, name := range expectedMessages {
		if _, exists := registry.messages[name]; !exists {
			t.Errorf("Message %s was not registered", name)
		}
	}

	// Check enums were registered
	expectedEnums := []string{
		"test.pkg.Enum1",
		"test.pkg.Message2.NestedEnum",
	}

	for _, name := range expectedEnums {
		if _, exists := registry.enums[name]; !exists {
			t.Errorf("Enum %s was not registered", name)
		}
	}

	// Check services were registered
	if _, exists := registry.services["test.pkg.Service1"]; !exists {
		t.Error("Service1 was not registered")
	}
}

func TestBuildDefinitions(t *testing.T) {
	registry := NewRegistry()

	// This is currently a placeholder, so just test it doesn't error
	err := registry.buildDefinitions(&schema.ProtoFile{})
	if err != nil {
		t.Errorf("buildDefinitions failed: %v", err)
	}
}

func TestBuildServices(t *testing.T) {
	registry := NewRegistry()

	// This is currently a placeholder, so just test it doesn't error
	err := registry.buildServices(&schema.ProtoFile{})
	if err != nil {
		t.Errorf("buildServices failed: %v", err)
	}
}

func TestLoadSingleProtoFile_Proto2(t *testing.T) {
	// Create a temporary proto2 file
	tmpFile, err := os.CreateTemp("", "test*.proto")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := `syntax = "proto2";
package test.proto2;

message TestMessage {
  required string name = 1;
}
`

	_, err = tmpFile.WriteString(content)
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	registry := NewRegistry()
	registry.repo = &schema.ProtoRepo{
		ProtoFiles: make(map[string]*schema.ProtoFile),
	}

	err = registry.loadSingleProtoFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("loadSingleProtoFile failed: %v", err)
	}

	protoFile := registry.repo.ProtoFiles[tmpFile.Name()]
	if protoFile == nil {
		t.Fatal("Proto file not loaded")
	}

	if protoFile.Syntax != "proto2" {
		t.Errorf("Expected syntax 'proto2', got '%s'", protoFile.Syntax)
	}

	if protoFile.Package != "test.proto2" {
		t.Errorf("Expected package 'test.proto2', got '%s'", protoFile.Package)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsMiddle(s, substr))))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestBuildDefinitions_Success(t *testing.T) {
	registry := NewRegistry()
	registry.messages = make(map[string]*schema.Message)
	registry.enums = make(map[string]*schema.Enum)

	// Add some test messages and enums to the registry
	testMessage := &schema.Message{Name: "TestMessage"}
	testEnum := &schema.Enum{Name: "TestEnum"}
	registry.messages["test.pkg.TestMessage"] = testMessage
	registry.enums["test.pkg.TestEnum"] = testEnum

	protoFile := &schema.ProtoFile{
		Package: "test.pkg",
		Messages: []*schema.Message{
			{
				Name: "Message1",
				Fields: []*schema.Field{
					{
						Name:   "field1",
						Number: 1,
						Type: schema.FieldType{
							Kind:        schema.KindMessage,
							MessageType: "TestMessage",
						},
					},
					{
						Name:   "field2",
						Number: 2,
						Type: schema.FieldType{
							Kind:     schema.KindEnum,
							EnumType: "TestEnum",
						},
					},
				},
			},
		},
		Enums: []*schema.Enum{
			{Name: "TestEnum"},
		},
	}

	err := registry.buildDefinitions(protoFile)
	if err != nil {
		t.Errorf("buildDefinitions failed: %v", err)
	}
}

func TestBuildDefinitions_InvalidMessageType(t *testing.T) {
	registry := NewRegistry()
	registry.messages = make(map[string]*schema.Message)
	registry.enums = make(map[string]*schema.Enum)

	protoFile := &schema.ProtoFile{
		Package: "test.pkg",
		Messages: []*schema.Message{
			{
				Name: "Message1",
				Fields: []*schema.Field{
					{
						Name:   "field1",
						Number: 1,
						Type: schema.FieldType{
							Kind:        schema.KindMessage,
							MessageType: "NonExistentMessage",
						},
					},
				},
			},
		},
	}

	err := registry.buildDefinitions(protoFile)
	if err == nil {
		t.Error("Expected error for invalid message type")
	}

	if !contains(err.Error(), "unknown type") {
		t.Errorf("Expected 'unknown type' error, got: %v", err)
	}
}

func TestBuildDefinitions_InvalidEnumType(t *testing.T) {
	registry := NewRegistry()
	registry.messages = make(map[string]*schema.Message)
	registry.enums = make(map[string]*schema.Enum)

	protoFile := &schema.ProtoFile{
		Package: "test.pkg",
		Messages: []*schema.Message{
			{
				Name: "Message1",
				Fields: []*schema.Field{
					{
						Name:   "field1",
						Number: 1,
						Type: schema.FieldType{
							Kind:     schema.KindEnum,
							EnumType: "NonExistentEnum",
						},
					},
				},
			},
		},
	}

	err := registry.buildDefinitions(protoFile)
	if err == nil {
		t.Error("Expected error for invalid enum type")
	}

	if !contains(err.Error(), "enum type") && !contains(err.Error(), "not found") {
		t.Errorf("Expected 'enum type not found' error, got: %v", err)
	}
}

func TestBuildDefinitions_NestedMessages(t *testing.T) {
	registry := NewRegistry()
	registry.messages = make(map[string]*schema.Message)
	registry.enums = make(map[string]*schema.Enum)

	// Add a test message to reference
	testMessage := &schema.Message{Name: "TestMessage"}
	registry.messages["test.pkg.TestMessage"] = testMessage

	protoFile := &schema.ProtoFile{
		Package: "test.pkg",
		Messages: []*schema.Message{
			{
				Name: "Message1",
				NestedTypes: []*schema.Message{
					{
						Name: "NestedMessage",
						Fields: []*schema.Field{
							{
								Name:   "field1",
								Number: 1,
								Type: schema.FieldType{
									Kind:        schema.KindMessage,
									MessageType: "TestMessage",
								},
							},
						},
					},
				},
			},
		},
	}

	err := registry.buildDefinitions(protoFile)
	if err != nil {
		t.Errorf("buildDefinitions failed for nested messages: %v", err)
	}
}

func TestBuildServices_Success(t *testing.T) {
	registry := NewRegistry()
	registry.messages = make(map[string]*schema.Message)

	// Add test messages for input/output types
	inputMessage := &schema.Message{Name: "InputMessage"}
	outputMessage := &schema.Message{Name: "OutputMessage"}
	registry.messages["test.pkg.InputMessage"] = inputMessage
	registry.messages["test.pkg.OutputMessage"] = outputMessage

	protoFile := &schema.ProtoFile{
		Package: "test.pkg",
		Services: []*schema.Service{
			{
				Name: "TestService",
				Methods: []*schema.Method{
					{
						Name:       "TestMethod",
						InputType:  "InputMessage",
						OutputType: "OutputMessage",
					},
				},
			},
		},
	}

	err := registry.buildServices(protoFile)
	if err != nil {
		t.Errorf("buildServices failed: %v", err)
	}
}

func TestBuildServices_InvalidInputType(t *testing.T) {
	registry := NewRegistry()
	registry.messages = make(map[string]*schema.Message)

	// Add only output message, missing input message
	outputMessage := &schema.Message{Name: "OutputMessage"}
	registry.messages["test.pkg.OutputMessage"] = outputMessage

	protoFile := &schema.ProtoFile{
		Package: "test.pkg",
		Services: []*schema.Service{
			{
				Name: "TestService",
				Methods: []*schema.Method{
					{
						Name:       "TestMethod",
						InputType:  "NonExistentInput",
						OutputType: "OutputMessage",
					},
				},
			},
		},
	}

	err := registry.buildServices(protoFile)
	if err == nil {
		t.Error("Expected error for invalid input type")
	}

	if !contains(err.Error(), "input type") && !contains(err.Error(), "not found") {
		t.Errorf("Expected input type error, got: %v", err)
	}
}

func TestBuildServices_InvalidOutputType(t *testing.T) {
	registry := NewRegistry()
	registry.messages = make(map[string]*schema.Message)

	// Add only input message, missing output message
	inputMessage := &schema.Message{Name: "InputMessage"}
	registry.messages["test.pkg.InputMessage"] = inputMessage

	protoFile := &schema.ProtoFile{
		Package: "test.pkg",
		Services: []*schema.Service{
			{
				Name: "TestService",
				Methods: []*schema.Method{
					{
						Name:       "TestMethod",
						InputType:  "InputMessage",
						OutputType: "NonExistentOutput",
					},
				},
			},
		},
	}

	err := registry.buildServices(protoFile)
	if err == nil {
		t.Error("Expected error for invalid output type")
	}

	if !contains(err.Error(), "output type") && !contains(err.Error(), "not found") {
		t.Errorf("Expected output type error, got: %v", err)
	}
}

func TestBuildServices_MultipleServices(t *testing.T) {
	registry := NewRegistry()
	registry.messages = make(map[string]*schema.Message)

	// Add test messages
	inputMessage := &schema.Message{Name: "InputMessage"}
	outputMessage := &schema.Message{Name: "OutputMessage"}
	registry.messages["test.pkg.InputMessage"] = inputMessage
	registry.messages["test.pkg.OutputMessage"] = outputMessage

	protoFile := &schema.ProtoFile{
		Package: "test.pkg",
		Services: []*schema.Service{
			{
				Name: "Service1",
				Methods: []*schema.Method{
					{
						Name:       "Method1",
						InputType:  "InputMessage",
						OutputType: "OutputMessage",
					},
				},
			},
			{
				Name: "Service2",
				Methods: []*schema.Method{
					{
						Name:       "Method2",
						InputType:  "InputMessage",
						OutputType: "OutputMessage",
					},
				},
			},
		},
	}

	err := registry.buildServices(protoFile)
	if err != nil {
		t.Errorf("buildServices failed for multiple services: %v", err)
	}
}

func TestResolveMessageFields_PrimitiveTypes(t *testing.T) {
	registry := NewRegistry()
	registry.messages = make(map[string]*schema.Message)

	message := &schema.Message{
		Name: "TestMessage",
		Fields: []*schema.Field{
			{
				Name:   "stringField",
				Number: 1,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeString,
				},
			},
			{
				Name:   "intField",
				Number: 2,
				Type: schema.FieldType{
					Kind:          schema.KindPrimitive,
					PrimitiveType: schema.TypeInt32,
				},
			},
		},
	}

	err := registry.resolveMessageFields(message, "test.pkg")
	if err != nil {
		t.Errorf("resolveMessageFields failed for primitive types: %v", err)
	}
}
