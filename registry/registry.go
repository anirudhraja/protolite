package registry

import (
	"fmt"
	"strings"

	"github.com/protolite/schema"
)

// Registry manages protobuf schemas and provides schema-aware operations
type Registry struct {
	repo     *schema.ProtoRepo
	messages map[string]*schema.Message // fully qualified name -> message
	enums    map[string]*schema.Enum    // fully qualified name -> enum
	services map[string]*schema.Service // fully qualified name -> service
}

// NewRegistry creates a new schema registry
func NewRegistry() *Registry {
	return &Registry{
		messages: make(map[string]*schema.Message),
		enums:    make(map[string]*schema.Enum),
		services: make(map[string]*schema.Service),
	}
}

// LoadRepo loads a protobuf repository into the registry
func (r *Registry) LoadRepo(repo *schema.ProtoRepo) error {
	if r.repo != nil {
		return fmt.Errorf("registry already has a repo loaded")
	}

	r.repo = repo

	// Multi-pass build process (inspired by tzero)
	return r.buildSymbolTable()
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

// getFullName constructs the fully qualified name
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
