package registry

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/protolite/schema"
)

// Registry allows us to store the schema of the protobuf messages. We look this up when we need to parse or marshal a message.
type Registry struct {
	repo     *schema.ProtoRepo
	messages map[string]*schema.Message // fully qualified name -> message
	enums    map[string]*schema.Enum    // fully qualified name -> enum
	services map[string]*schema.Service // fully qualified name -> service

}

func NewRegistry() *Registry {
	return &Registry{}
}

// LoadSchema Given a path it will recursively scan all *proto files inside it and return schema.ProtoRepo
func (r *Registry) LoadSchema(protoPath string) error {
	// Initialize the registry maps if not already done
	if r.messages == nil {
		r.messages = make(map[string]*schema.Message)
	}
	if r.enums == nil {
		r.enums = make(map[string]*schema.Enum)
	}
	if r.services == nil {
		r.services = make(map[string]*schema.Service)
	}

	// Initialize the repo if not already done
	if r.repo == nil {
		r.repo = &schema.ProtoRepo{
			ProtoFiles: make(map[string]*schema.ProtoFile),
		}
	}

	// Check if the path exists
	info, err := os.Stat(protoPath)
	if err != nil {
		return fmt.Errorf("path does not exist: %w", err)
	}

	// If it's a single file, process it directly
	if !info.IsDir() {
		if strings.HasSuffix(protoPath, ".proto") {
			if err := r.loadSingleProtoFile(protoPath); err != nil {
				return fmt.Errorf("failed to load proto file: %w", err)
			}
		} else {
			return fmt.Errorf("file %s is not a .proto file", protoPath)
		}
	} else {
		// If it's a directory, walk through it recursively
		err = filepath.WalkDir(protoPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Skip directories and non-proto files
			if d.IsDir() || !strings.HasSuffix(path, ".proto") {
				return nil
			}

			// Load the proto file
			if err := r.loadSingleProtoFile(path); err != nil {
				return fmt.Errorf("failed to load proto file %s: %w", path, err)
			}

			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to walk directory: %w", err)
		}
	}

	// After loading all files, populate the registry maps
	if err := r.buildSymbolTable(); err != nil {
		return fmt.Errorf("failed to build symbol table: %w", err)
	}

	return nil
}

// loadSingleProtoFile loads and parses a single .proto file
func (r *Registry) loadSingleProtoFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	protoFile := &schema.ProtoFile{
		Name:     filepath.Base(filePath),
		Syntax:   "proto3", // Default
		Package:  "",
		Imports:  []*schema.Import{},
		Messages: []*schema.Message{},
		Enums:    []*schema.Enum{},
		Services: []*schema.Service{},
	}

	// Parse the proto file content
	if err := r.parseProtoContent(string(content), protoFile); err != nil {
		return fmt.Errorf("failed to parse proto content: %w", err)
	}

	// Store in the ProtoRepo
	r.repo.ProtoFiles[filePath] = protoFile
	return nil
}

// parseProtoContent parses the content of a proto file and populates the ProtoFile structure
func (r *Registry) parseProtoContent(content string, protoFile *schema.ProtoFile) error {
	lines := strings.Split(content, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Parse package
		if strings.HasPrefix(line, "package ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				protoFile.Package = strings.TrimSuffix(parts[1], ";")
			}
			continue
		}

		// Parse syntax
		if strings.HasPrefix(line, "syntax ") {
			if strings.Contains(line, "proto2") {
				protoFile.Syntax = "proto2"
			}
			continue
		}

		// Parse imports
		if strings.HasPrefix(line, "import ") {
			// Extract import path - simple implementation
			start := strings.Index(line, "\"")
			end := strings.LastIndex(line, "\"")
			if start != -1 && end != -1 && start != end {
				importPath := line[start+1 : end]
				protoFile.Imports = append(protoFile.Imports, &schema.Import{Path: importPath})
			}
			continue
		}

		// Parse messages
		if strings.HasPrefix(line, "message ") {
			message, newIndex, err := r.parseMessage(lines, i)
			if err != nil {
				return err
			}
			protoFile.Messages = append(protoFile.Messages, message)
			i = newIndex
			continue
		}

		// Parse enums
		if strings.HasPrefix(line, "enum ") {
			enum, newIndex, err := r.parseEnum(lines, i)
			if err != nil {
				return err
			}
			protoFile.Enums = append(protoFile.Enums, enum)
			i = newIndex
			continue
		}
	}

	return nil
}

// parseMessage parses a message definition starting from the given line index
func (r *Registry) parseMessage(lines []string, startIndex int) (*schema.Message, int, error) {
	line := strings.TrimSpace(lines[startIndex])

	// Extract message name
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil, startIndex, fmt.Errorf("invalid message declaration: %s", line)
	}

	messageName := parts[1]
	if strings.HasSuffix(messageName, "{") {
		messageName = strings.TrimSuffix(messageName, "{")
	}
	messageName = strings.TrimSpace(messageName)

	message := &schema.Message{
		Name:        messageName,
		Fields:      []*schema.Field{},
		NestedTypes: []*schema.Message{},
		NestedEnums: []*schema.Enum{},
	}

	// Find opening brace if not on same line
	i := startIndex
	for i < len(lines) && !strings.Contains(lines[i], "{") {
		i++
	}
	i++ // Move past opening brace line

	// Parse fields until closing brace
	for i < len(lines) {
		fieldLine := strings.TrimSpace(lines[i])

		// Skip empty lines and comments
		if fieldLine == "" || strings.HasPrefix(fieldLine, "//") {
			i++
			continue
		}

		// Check for closing brace
		if strings.HasPrefix(fieldLine, "}") {
			break
		}

		// Parse field
		field, err := r.parseField(fieldLine)
		if err != nil {
			return nil, i, fmt.Errorf("failed to parse field in message %s: %w", messageName, err)
		}

		if field != nil {
			message.Fields = append(message.Fields, field)
		}

		i++
	}

	return message, i, nil
}

// parseField parses a field definition line
func (r *Registry) parseField(line string) (*schema.Field, error) {
	// Remove trailing semicolon and comments
	line = strings.TrimSuffix(line, ";")
	if commentIndex := strings.Index(line, "//"); commentIndex != -1 {
		line = strings.TrimSpace(line[:commentIndex])
	}

	parts := strings.Fields(line)
	if len(parts) < 4 {
		return nil, nil // Skip invalid field lines
	}

	// Handle repeated, map, and optional keywords
	fieldIndex := 0
	label := schema.LabelOptional

	if parts[0] == "repeated" {
		label = schema.LabelRepeated
		fieldIndex = 1
	} else if parts[0] == "map" || strings.HasPrefix(line, "map<") {
		// Handle map<key, value> field_name = number;
		return r.parseMapField(line)
	}

	fieldType := parts[fieldIndex]
	fieldName := parts[fieldIndex+1]

	// Extract field number
	numberPart := ""
	for i := fieldIndex + 2; i < len(parts); i++ {
		if parts[i] == "=" && i+1 < len(parts) {
			numberPart = parts[i+1]
			break
		}
		if strings.Contains(parts[i], "=") {
			numberPart = strings.Split(parts[i], "=")[1]
			break
		}
	}

	if numberPart == "" {
		return nil, fmt.Errorf("no field number found in: %s", line)
	}

	// Parse field number
	var fieldNumber int32
	if _, err := fmt.Sscanf(numberPart, "%d", &fieldNumber); err != nil {
		return nil, fmt.Errorf("invalid field number: %s", numberPart)
	}

	// Convert type to FieldType
	protoFieldType, err := r.convertProtoType(fieldType)
	if err != nil {
		return nil, err
	}

	return &schema.Field{
		Name:   fieldName,
		Number: fieldNumber,
		Label:  label,
		Type:   *protoFieldType,
	}, nil
}

// parseMapField parses a map field definition
func (r *Registry) parseMapField(line string) (*schema.Field, error) {
	// Example: map<string, string> metadata = 7;
	mapStart := strings.Index(line, "<")
	mapEnd := strings.Index(line, ">")
	if mapStart == -1 || mapEnd == -1 {
		return nil, fmt.Errorf("invalid map field: %s", line)
	}

	mapTypes := strings.TrimSpace(line[mapStart+1 : mapEnd])
	typeParts := strings.Split(mapTypes, ",")
	if len(typeParts) != 2 {
		return nil, fmt.Errorf("invalid map types: %s", mapTypes)
	}

	keyType := strings.TrimSpace(typeParts[0])
	valueType := strings.TrimSpace(typeParts[1])

	// Extract field name and number
	afterMap := strings.TrimSpace(line[mapEnd+1:])
	parts := strings.Fields(afterMap)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid map field format: %s", line)
	}

	fieldName := parts[0]

	// Find field number
	var fieldNumber int32
	for i := 1; i < len(parts); i++ {
		if parts[i] == "=" && i+1 < len(parts) {
			if _, err := fmt.Sscanf(parts[i+1], "%d", &fieldNumber); err != nil {
				return nil, fmt.Errorf("invalid field number: %s", parts[i+1])
			}
			break
		}
	}

	// Convert types
	keyFieldType, err := r.convertProtoType(keyType)
	if err != nil {
		return nil, err
	}

	valueFieldType, err := r.convertProtoType(valueType)
	if err != nil {
		return nil, err
	}

	return &schema.Field{
		Name:   fieldName,
		Number: fieldNumber,
		Label:  schema.LabelOptional,
		Type: schema.FieldType{
			Kind:     schema.KindMap,
			MapKey:   keyFieldType,
			MapValue: valueFieldType,
		},
	}, nil
}

// parseEnum parses an enum definition starting from the given line index
func (r *Registry) parseEnum(lines []string, startIndex int) (*schema.Enum, int, error) {
	line := strings.TrimSpace(lines[startIndex])

	// Extract enum name
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil, startIndex, fmt.Errorf("invalid enum declaration: %s", line)
	}

	enumName := parts[1]
	if strings.HasSuffix(enumName, "{") {
		enumName = strings.TrimSuffix(enumName, "{")
	}
	enumName = strings.TrimSpace(enumName)

	enum := &schema.Enum{
		Name:   enumName,
		Values: []*schema.EnumValue{},
	}

	// Find opening brace if not on same line
	i := startIndex
	for i < len(lines) && !strings.Contains(lines[i], "{") {
		i++
	}
	i++ // Move past opening brace line

	// Parse enum values until closing brace
	for i < len(lines) {
		valueLine := strings.TrimSpace(lines[i])

		// Skip empty lines and comments
		if valueLine == "" || strings.HasPrefix(valueLine, "//") {
			i++
			continue
		}

		// Check for closing brace
		if strings.HasPrefix(valueLine, "}") {
			break
		}

		// Parse enum value
		enumValue, err := r.parseEnumValue(valueLine)
		if err != nil {
			return nil, i, fmt.Errorf("failed to parse enum value in enum %s: %w", enumName, err)
		}

		if enumValue != nil {
			enum.Values = append(enum.Values, enumValue)
		}

		i++
	}

	return enum, i, nil
}

// parseEnumValue parses an enum value line
func (r *Registry) parseEnumValue(line string) (*schema.EnumValue, error) {
	// Remove trailing semicolon and comments
	line = strings.TrimSuffix(line, ";")
	if commentIndex := strings.Index(line, "//"); commentIndex != -1 {
		line = strings.TrimSpace(line[:commentIndex])
	}

	parts := strings.Fields(line)
	if len(parts) < 3 {
		return nil, nil // Skip invalid enum value lines
	}

	enumName := parts[0]

	// Find the number after "="
	var enumNumber int32
	for i := 1; i < len(parts); i++ {
		if parts[i] == "=" && i+1 < len(parts) {
			if _, err := fmt.Sscanf(parts[i+1], "%d", &enumNumber); err != nil {
				return nil, fmt.Errorf("invalid enum number: %s", parts[i+1])
			}
			break
		}
	}

	return &schema.EnumValue{
		Name:   enumName,
		Number: enumNumber,
	}, nil
}

// convertProtoType converts a protobuf type string to a FieldType
func (r *Registry) convertProtoType(protoType string) (*schema.FieldType, error) {
	switch protoType {
	case "int32":
		return &schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeInt32}, nil
	case "int64":
		return &schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeInt64}, nil
	case "uint32":
		return &schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeUint32}, nil
	case "uint64":
		return &schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeUint64}, nil
	case "string":
		return &schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeString}, nil
	case "bytes":
		return &schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeBytes}, nil
	case "bool":
		return &schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeBool}, nil
	case "float":
		return &schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeFloat}, nil
	case "double":
		return &schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeDouble}, nil
	default:
		// For non-primitive types, we need to determine if it's an enum or message
		// This will be resolved later in buildDefinitions after all types are registered
		return &schema.FieldType{Kind: schema.KindMessage, MessageType: protoType}, nil
	}
}

// buildSymbolTable builds the symbol table from the loaded repository
func (r *Registry) buildSymbolTable() error {
	// Pass 1: Register all message and enum names
	for _, protoFile := range r.repo.ProtoFiles {
		if err := r.registerNames(protoFile); err != nil {
			return err
		}
	}

	// Pass 2: Build all message and enum definitions
	for _, protoFile := range r.repo.ProtoFiles {
		if err := r.buildDefinitions(protoFile); err != nil {
			return err
		}
	}

	// Pass 3: Build services
	for _, protoFile := range r.repo.ProtoFiles {
		if err := r.buildServices(protoFile); err != nil {
			return err
		}
	}

	return nil
}

// registerNames registers all message, enum, and service names
func (r *Registry) registerNames(protoFile *schema.ProtoFile) error {
	pkg := protoFile.Package
	// Register messages
	for _, msg := range protoFile.Messages {
		fullName := r.getFullName(pkg, msg.Name)
		r.messages[fullName] = msg

		// Register nested types
		if err := r.registerNestedNames(pkg, msg.Name, msg); err != nil {
			return err
		}
	}

	// Register enums
	for _, enum := range protoFile.Enums {
		fullName := r.getFullName(pkg, enum.Name)
		r.enums[fullName] = enum
	}

	// Register services
	for _, service := range protoFile.Services {
		fullName := r.getFullName(pkg, service.Name)
		r.services[fullName] = service
	}

	return nil
}

// registerNestedNames registers nested message and enum names
func (r *Registry) registerNestedNames(pkg, parentName string, msg *schema.Message) error {
	// Register nested messages
	for _, nestedMsg := range msg.NestedTypes {
		nestedFullName := r.getFullName(pkg, parentName+"."+nestedMsg.Name)
		r.messages[nestedFullName] = nestedMsg

		// Recursively register nested types
		if err := r.registerNestedNames(pkg, parentName+"."+nestedMsg.Name, nestedMsg); err != nil {
			return err
		}
	}

	// Register nested enums
	for _, nestedEnum := range msg.NestedEnums {
		nestedFullName := r.getFullName(pkg, parentName+"."+nestedEnum.Name)
		r.enums[nestedFullName] = nestedEnum
	}

	return nil
}

// buildDefinitions builds the complete definitions (placeholder for now)
func (r *Registry) buildDefinitions(protoFile *schema.ProtoFile) error {
	// Validate field types and resolve references
	for _, message := range protoFile.Messages {
		if err := r.resolveMessageFields(message, protoFile.Package); err != nil {
			return fmt.Errorf("failed to resolve fields in message %s: %w", message.Name, err)
		}
	}
	return nil
}

// buildServices builds service definitions (placeholder for now)
func (r *Registry) buildServices(protoFile *schema.ProtoFile) error {
	// Validate service method input/output types
	for _, service := range protoFile.Services {
		for _, method := range service.Methods {
			// Check if input type exists
			if _, err := r.GetMessage(method.InputType); err != nil {
				return fmt.Errorf("service %s method %s: input type %s not found",
					service.Name, method.Name, method.InputType)
			}

			// Check if output type exists
			if _, err := r.GetMessage(method.OutputType); err != nil {
				return fmt.Errorf("service %s method %s: output type %s not found",
					service.Name, method.Name, method.OutputType)
			}
		}
	}
	return nil
}

// resolveMessageFields resolves field type references within a message
func (r *Registry) resolveMessageFields(message *schema.Message, packageName string) error {
	for _, field := range message.Fields {
		// For map fields, resolve both key and value types
		if field.Type.Kind == schema.KindMap {
			if err := r.resolveFieldType(field.Type.MapKey, packageName); err != nil {
				return fmt.Errorf("failed to resolve map key type in field %s: %v", field.Name, err)
			}
			if err := r.resolveFieldType(field.Type.MapValue, packageName); err != nil {
				return fmt.Errorf("failed to resolve map value type in field %s: %v", field.Name, err)
			}
			continue
		}

		// For regular fields, resolve the field type
		if err := r.resolveFieldType(&field.Type, packageName); err != nil {
			return fmt.Errorf("failed to resolve field %s: %v", field.Name, err)
		}
	}

	// Recursively process nested messages
	for _, nestedMsg := range message.NestedTypes {
		if err := r.resolveMessageFields(nestedMsg, packageName); err != nil {
			return err
		}
	}

	return nil
}

// resolveFieldType resolves a single field type, determining if it's an enum or message
func (r *Registry) resolveFieldType(fieldType *schema.FieldType, packageName string) error {
	// Skip primitive types
	if fieldType.Kind == schema.KindPrimitive || fieldType.Kind == schema.KindMap {
		return nil
	}

	// For types that were initially marked as message, check if they're actually enums
	if fieldType.Kind == schema.KindMessage {
		typeName := fieldType.MessageType

		// First check if it's an enum
		if _, err := r.GetEnum(typeName); err == nil {
			// It's an enum, fix the field type
			fieldType.Kind = schema.KindEnum
			fieldType.EnumType = typeName
			fieldType.MessageType = "" // Clear the message type
			return nil
		}

		// Check if it's a message
		if _, err := r.GetMessage(typeName); err == nil {
			// It's a message, keep as is
			return nil
		}

		return fmt.Errorf("unknown type %s", typeName)
	}

	// For enum fields, verify the enum exists
	if fieldType.Kind == schema.KindEnum {
		if _, err := r.GetEnum(fieldType.EnumType); err != nil {
			return fmt.Errorf("enum type %s not found", fieldType.EnumType)
		}
	}

	return nil
}

func (r *Registry) getFullName(pkg, name string) string {
	if pkg == "" {
		return name
	}
	return pkg + "." + name
}

// GetMessage retrieves a message definition by name
func (r *Registry) GetMessage(name string) (*schema.Message, error) {
	if msg, exists := r.messages[name]; exists {
		return msg, nil
	}

	// Try without package prefix
	for fullName, msg := range r.messages {
		if strings.HasSuffix(fullName, "."+name) || fullName == name {
			return msg, nil
		}
	}

	return nil, fmt.Errorf("message not found: %s", name)
}

// GetEnum retrieves an enum definition by name
func (r *Registry) GetEnum(name string) (*schema.Enum, error) {
	if enum, exists := r.enums[name]; exists {
		return enum, nil
	}

	// Try without package prefix
	for fullName, enum := range r.enums {
		if strings.HasSuffix(fullName, "."+name) || fullName == name {
			return enum, nil
		}
	}

	return nil, fmt.Errorf("enum not found: %s", name)
}

// GetService retrieves a service definition by name
func (r *Registry) GetService(name string) (*schema.Service, error) {
	if service, exists := r.services[name]; exists {
		return service, nil
	}

	// Try without package prefix
	for fullName, service := range r.services {
		if strings.HasSuffix(fullName, "."+name) || fullName == name {
			return service, nil
		}
	}

	return nil, fmt.Errorf("service not found: %s", name)
}

// ListMessages returns all registered message names
func (r *Registry) ListMessages() []string {
	var names []string
	for name := range r.messages {
		names = append(names, name)
	}
	return names
}

// ListEnums returns all registered enum names
func (r *Registry) ListEnums() []string {
	var names []string
	for name := range r.enums {
		names = append(names, name)
	}
	return names
}

// ListServices returns all registered service names
func (r *Registry) ListServices() []string {
	var names []string
	for name := range r.services {
		names = append(names, name)
	}
	return names
}

// GetOrCreateMapEntryMessage creates a synthetic message type for map entries
func (r *Registry) GetOrCreateMapEntryMessage(mapFieldName string, keyType, valueType *schema.FieldType) (*schema.Message, error) {
	entryTypeName := mapFieldName + "Entry"

	// Check if already exists
	if msg, exists := r.messages[entryTypeName]; exists {
		return msg, nil
	}

	// Create synthetic map entry message
	mapEntryMessage := &schema.Message{
		Name:     entryTypeName,
		MapEntry: true,
		Fields: []*schema.Field{
			{
				Name:   "key",
				Number: 1,
				Label:  schema.LabelOptional,
				Type:   *keyType,
			},
			{
				Name:   "value",
				Number: 2,
				Label:  schema.LabelOptional,
				Type:   *valueType,
			},
		},
	}

	// Register it
	r.messages[entryTypeName] = mapEntryMessage
	return mapEntryMessage, nil
}

// ListProtoFiles returns all loaded proto file paths
func (r *Registry) ListProtoFiles() []string {
	if r.repo == nil {
		return nil
	}

	var paths []string
	for path := range r.repo.ProtoFiles {
		paths = append(paths, path)
	}
	return paths
}
