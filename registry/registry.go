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

	r.repo = &schema.ProtoRepo{
		ProtoFiles: make(map[string]*schema.ProtoFile),
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

	// Basic parsing - extract package name
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				protoFile.Package = strings.TrimSuffix(parts[1], ";")
			}
		}
		if strings.HasPrefix(line, "syntax ") {
			if strings.Contains(line, "proto2") {
				protoFile.Syntax = "proto2"
			}
		}
	}

	// Store in the ProtoRepo
	r.repo.ProtoFiles[filePath] = protoFile
	return nil
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
	// TODO: Validate field types, resolve references, etc.
	return nil
}

// buildServices builds service definitions (placeholder for now)
func (r *Registry) buildServices(protoFile *schema.ProtoFile) error {
	// TODO: Build service method handlers
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
