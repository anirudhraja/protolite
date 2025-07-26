package registry

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/anirudhraja/protolite/schema"
	protoparserparser "github.com/yoheimuta/go-protoparser/v4/parser"
)

// Registry allows us to store the schema of the protobuf messages. We look this up when we need to parse or marshal a message.
type Registry struct {
	repo             *schema.ProtoRepo
	messages         map[string]*schema.Message          // fully qualified name -> message
	enums            map[string]*schema.Enum             // fully qualified name -> enum
	services         map[string]*schema.Service          // fully qualified name -> service
	protoEntities    map[string]*protoFileEntity         // for each proto store the entities so its easy to refer
	parsedProtoBody  map[string]*protoparserparser.Proto // just a cache to avoid parsing proto body
	ProtoDirectories []string                            // list of directories to search for the imported protos
}

// preprocessing the proto file to store the proto entities
type protoFileEntity struct {
	entities []string
	imports  []string
}

func NewRegistry(ProtoDirectories []string) *Registry {
	return &Registry{
		ProtoDirectories: ProtoDirectories,
	}
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
	if r.protoEntities == nil {
		r.protoEntities = map[string]*protoFileEntity{}
	}
	// Initialize the repo if not already done
	if r.parsedProtoBody == nil {
		r.parsedProtoBody = map[string]*protoparserparser.Proto{}
	}

	// Initialize the repo if not already done
	if r.repo == nil {
		r.repo = &schema.ProtoRepo{
			ProtoFiles: make(map[string]*schema.ProtoFile),
		}
	}
	allProtoFiles, err := r.getAllProtoInfo(protoPath)
	if err != nil {
		return err
	}

	for _, protoPath := range allProtoFiles {
		// resolves and stores the type in proto file
		protoFileEntity, err := r.resolveProtoFile(protoPath)
		if err != nil {
			return fmt.Errorf("failed to load proto file: %w", err)
		}
		ent, ok := r.protoEntities[protoPath]
		if !ok {
			return fmt.Errorf("failed to get proto entities for file %s:", protoPath)
		}
		ent.entities = protoFileEntity
	}
	for _, protoPath := range allProtoFiles {
		// loadSingleProtoFile processes the proto file
		if err := r.loadSingleProtoFile(protoPath); err != nil {
			return fmt.Errorf("failed to load proto file: %w", err)
		}
	}
	// After loading all files, populate the registry maps
	if err := r.buildSymbolTable(); err != nil {
		return fmt.Errorf("failed to build symbol table: %w", err)
	}

	return nil
}

func (r *Registry) resolveProtoFile(protoPath string) ([]string, error) {
	parsedProtoBody, ok := r.parsedProtoBody[protoPath]
	if !ok {
		return nil, fmt.Errorf("cannot find parsed proto body for: %s", protoPath)
	}
	allEntities := make([]string, 0)
	packageName := ""
	for _, entity := range parsedProtoBody.ProtoBody {
		switch b := entity.(type) {
		case *protoparserparser.Package:
			packageName = b.Name
		}
		if packageName != "" {
			break
		}
	}
	allEntities = append(allEntities, addNestedEntities(parsedProtoBody.ProtoBody, packageName, packageName)...)
	return allEntities, nil
}
func addNestedEntities(entites []protoparserparser.Visitee, prefix, packageName string) []string {
	result := []string{}
	if prefix != packageName {
		result = append(result, prefix)
	}
	prefix = prefix + "."
	for _, entity := range entites {
		switch b := entity.(type) {
		case *protoparserparser.Message:
			result = append(result, addNestedEntities(b.MessageBody, prefix+b.MessageName, packageName)...)

		case *protoparserparser.Enum:
			result = append(result, prefix+b.EnumName)
		}
	}
	return result

}

func (r *Registry) getAllEntities(filePath string) map[string]struct{} {
	res := make(map[string]struct{})
	allEntities := r.protoEntities[filePath]
	for _, file := range allEntities.imports {
		for _, entities := range r.protoEntities[file].entities {
			res[entities] = struct{}{}
		}
	}
	for _, entities := range allEntities.entities {
		res[entities] = struct{}{}
	}
	return res
}

// loadSingleProtoFile loads and parses a single .proto file
func (r *Registry) loadSingleProtoFile(filePath string) error {
	parsedProtoBody, ok := r.parsedProtoBody[filePath]
	if !ok {
		return fmt.Errorf("cannot find parsed proto body for: %s", filePath)
	}
	allResolvedEntities := r.getAllEntities(filePath)

	protoFile := &schema.ProtoFile{
		Name:     filepath.Base(filePath),
		Syntax:   "proto3", // Default
		Imports:  []*schema.Import{},
		Messages: []*schema.Message{},
		Enums:    []*schema.Enum{},
		Services: []*schema.Service{},
	}

	// preprocess the imports first and add package name to each entity
	for _, body := range parsedProtoBody.ProtoBody {
		switch b := body.(type) {

		case *protoparserparser.Package:
			protoFile.Package = b.Name

		case *protoparserparser.Import: // resolve relation for each imports
			singleImport := &schema.Import{
				Path: b.Location,
			}
			if b.Modifier == protoparserparser.ImportModifierPublic {
				singleImport.Public = true
			}

			if b.Modifier == protoparserparser.ImportModifierWeak {
				singleImport.Weak = true
			}
			protoFile.Imports = append(protoFile.Imports, singleImport)
		case *protoparserparser.Message:
			msg, err := r.processMessage(b, allResolvedEntities, protoFile.Package)
			if err != nil {
				return fmt.Errorf("Unable to process message: %v", b.MessageName)
			}
			protoFile.Messages = append(protoFile.Messages, msg)
		case *protoparserparser.Enum:
			enum, err := r.processEnum(b)
			if err != nil {
				return fmt.Errorf("Unable to process enum: %v", b.EnumName)
			}
			protoFile.Enums = append(protoFile.Enums, enum)
		case *protoparserparser.Service:
			service, err := r.processService(b)
			if err != nil {
				return fmt.Errorf("Unable to process service: %v", b.ServiceName)
			}
			protoFile.Services = append(protoFile.Services, service)

		}
	}
	// Store in the ProtoRepo
	r.repo.ProtoFiles[filePath] = protoFile
	return nil
}

// parseMessage parses a message definition starting from the given line index
func (r *Registry) processMessage(message *protoparserparser.Message, allResolvedEntities map[string]struct{}, prefix string) (*schema.Message, error) {
	msg := &schema.Message{
		Name: message.MessageName,
	}
	prefix = prefix + "." + message.MessageName
	nestedEnums := make([]*schema.Enum, 0)
	nestedTypes := make([]*schema.Message, 0)
	fields := make([]*schema.Field, 0)
	oneOfGroups := make([]*schema.Oneof, 0)
	for _, m := range message.MessageBody {
		switch b := m.(type) {
		case *protoparserparser.Enum:
			enum, err := r.processEnum(b)
			if err != nil {
				return nil, err
			}
			nestedEnums = append(nestedEnums, enum)
		case *protoparserparser.Message:
			msg, err := r.processMessage(b, allResolvedEntities, prefix)
			if err != nil {
				return nil, err
			}
			nestedTypes = append(nestedTypes, msg)
		case *protoparserparser.Field:
			field, err := r.processField(b, allResolvedEntities, prefix)
			if err != nil {
				return nil, err
			}
			fields = append(fields, field)
		case *protoparserparser.MapField:
			field, err := r.processMapField(b, allResolvedEntities, prefix)
			if err != nil {
				return nil, err
			}
			fields = append(fields, field)
		case *protoparserparser.Oneof:
			oneOfFields := make([]*schema.Field, 0)
			for _, field := range b.OneofFields {
				fieldNumber, err := strconv.ParseInt(field.FieldNumber, 10, 32)
				if err != nil {
					return nil, err
				}
				fieldType, err := r.convertProtoType(field.Type, allResolvedEntities, prefix)
				if err != nil {
					return nil, err
				}
				fieldLabel := schema.LabelOptional
				f := &schema.Field{
					Name:     field.FieldName,
					Number:   int32(fieldNumber),
					Label:    fieldLabel,
					Type:     *fieldType,
					JsonName: findJSONName(field.FieldOptions),
				}
				oneOfFields = append(oneOfFields, f)
			}
			oneOfGroups = append(oneOfGroups, &schema.Oneof{
				Name:   b.OneofName,
				Fields: oneOfFields,
			})
		}
	}
	msg.NestedTypes = nestedTypes
	msg.Fields = fields
	msg.NestedEnums = nestedEnums
	msg.OneofGroups = oneOfGroups

	return msg, nil
}

func (r *Registry) processField(field *protoparserparser.Field, resolvedEntities map[string]struct{}, prefix string) (*schema.Field, error) {
	fieldNumber, err := strconv.ParseInt(field.FieldNumber, 10, 32)
	if err != nil {
		return nil, err
	}
	fieldLabel := schema.LabelOptional
	if field.IsRepeated {
		fieldLabel = schema.LabelRepeated
	} else if field.IsRequired {
		fieldLabel = schema.LabelRequired
	}
	fieldType, err := r.convertProtoType(field.Type, resolvedEntities, prefix)
	if err != nil {
		return nil, err
	}
	f := &schema.Field{
		Name:     field.FieldName,
		Number:   int32(fieldNumber),
		Label:    fieldLabel,
		Type:     *fieldType,
		JsonName: findJSONName(field.FieldOptions),
	}
	return f, nil
}

func (r *Registry) processMapField(field *protoparserparser.MapField, resolvedEntities map[string]struct{}, prefix string) (*schema.Field, error) {
	fieldNumber, err := strconv.ParseInt(field.FieldNumber, 10, 32)
	if err != nil {
		return nil, err
	}
	mapKeyType, err := r.convertProtoType(field.KeyType, resolvedEntities, prefix)
	if err != nil {
		return nil, err
	}
	mapValueType, err := r.convertProtoType(field.Type, resolvedEntities, prefix)
	if err != nil {
		return nil, err
	}
	f := &schema.Field{
		Name:   field.MapName,
		Number: int32(fieldNumber),
		Label:  schema.LabelOptional,
		Type: schema.FieldType{
			Kind:     schema.KindMap,
			MapKey:   mapKeyType,
			MapValue: mapValueType,
		},
		JsonName: findJSONName(field.FieldOptions),
	}
	return f, nil
}

func (r *Registry) processService(service *protoparserparser.Service) (*schema.Service, error) {
	methods := make([]*schema.Method, 0)
	for _, rpc := range service.ServiceBody {
		switch b := rpc.(type) {
		case *protoparserparser.RPC:
			method := &schema.Method{
				Name:            b.RPCName,
				InputType:       b.RPCRequest.MessageType,
				OutputType:      b.RPCResponse.MessageType,
				ClientStreaming: b.RPCRequest.IsStream,
				ServerStreaming: b.RPCResponse.IsStream,
			}
			methods = append(methods, method)
		}
	}
	return &schema.Service{
		Name:    service.ServiceName,
		Methods: methods,
	}, nil
}

// processEnum parses an enum definition starting from the given line index
func (r *Registry) processEnum(enum *protoparserparser.Enum) (*schema.Enum, error) {
	enumValues := make([]*schema.EnumValue, 0)
	for _, en := range enum.EnumBody {
		switch b := en.(type) {

		case *protoparserparser.EnumField:
			num, err := strconv.Atoi(b.Number)
			if err != nil {
				return nil, err
			}
			enumValues = append(enumValues, &schema.EnumValue{
				Name:   b.Ident,
				Number: int32(num),
			})
		}
	}
	return &schema.Enum{
		Name:   enum.EnumName,
		Values: enumValues,
	}, nil
}

// convertProtoType converts a protobuf type string to a FieldType
func (r *Registry) convertProtoType(protoType string, allResolvedEntities map[string]struct{}, prefix string) (*schema.FieldType, error) {
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
	case "fixed32":
		return &schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeFixed32}, nil
	case "fixed64":
		return &schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeFixed64}, nil
	case "sfixed32":
		return &schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeSfixed32}, nil
	case "sfixed64":
		return &schema.FieldType{Kind: schema.KindPrimitive, PrimitiveType: schema.TypeSfixed64}, nil
	// Google protobuf wrapper types
	case "google.protobuf.DoubleValue":
		return &schema.FieldType{Kind: schema.KindWrapper, WrapperType: schema.WrapperDoubleValue}, nil
	case "google.protobuf.FloatValue":
		return &schema.FieldType{Kind: schema.KindWrapper, WrapperType: schema.WrapperFloatValue}, nil
	case "google.protobuf.Int64Value":
		return &schema.FieldType{Kind: schema.KindWrapper, WrapperType: schema.WrapperInt64Value}, nil
	case "google.protobuf.UInt64Value":
		return &schema.FieldType{Kind: schema.KindWrapper, WrapperType: schema.WrapperUInt64Value}, nil
	case "google.protobuf.Int32Value":
		return &schema.FieldType{Kind: schema.KindWrapper, WrapperType: schema.WrapperInt32Value}, nil
	case "google.protobuf.UInt32Value":
		return &schema.FieldType{Kind: schema.KindWrapper, WrapperType: schema.WrapperUInt32Value}, nil
	case "google.protobuf.BoolValue":
		return &schema.FieldType{Kind: schema.KindWrapper, WrapperType: schema.WrapperBoolValue}, nil
	case "google.protobuf.StringValue":
		return &schema.FieldType{Kind: schema.KindWrapper, WrapperType: schema.WrapperStringValue}, nil
	case "google.protobuf.BytesValue":
		return &schema.FieldType{Kind: schema.KindWrapper, WrapperType: schema.WrapperBytesValue}, nil
	default:
		// For non-primitive types, we need to determine if it's an enum or message
		// This will be resolved later in buildDefinitions after all types are registered
		// TODO handle error
		fullResolvedType, err := getReferencedType(protoType, prefix, allResolvedEntities)
		if err != nil {
			return nil, err
		}
		return &schema.FieldType{Kind: schema.KindMessage, MessageType: fullResolvedType}, nil
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
	// Skip primitive types and wrapper types
	if fieldType.Kind == schema.KindPrimitive || fieldType.Kind == schema.KindMap || fieldType.Kind == schema.KindWrapper {
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
// Supports both fully qualified names (com.example.User) and short names (User)
// For short names, returns error if multiple matches found (ambiguous)
func (r *Registry) GetMessage(name string) (*schema.Message, error) {
	// First: try exact match (fully qualified or already unique)
	if msg, exists := r.messages[name]; exists {
		return msg, nil
	}

	// If name contains a dot, it's a fully qualified name that doesn't exist
	if strings.Contains(name, ".") {
		return nil, fmt.Errorf("message not found: %s", name)
	}

	// For short names, collect all matches
	var matches []*schema.Message
	var matchedNames []string

	for fullName, msg := range r.messages {
		// Check if the name matches the last component of the full name
		if strings.HasSuffix(fullName, "."+name) || fullName == name {
			matches = append(matches, msg)
			matchedNames = append(matchedNames, fullName)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("message not found: %s", name)
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("ambiguous message name '%s' matches multiple: %v. Use fully qualified name",
			name, matchedNames)
	}
}

// GetEnum retrieves an enum definition by name
// Supports both fully qualified names and short names with ambiguity detection
func (r *Registry) GetEnum(name string) (*schema.Enum, error) {
	// First: try exact match (fully qualified or already unique)
	if enum, exists := r.enums[name]; exists {
		return enum, nil
	}

	// Second: try short name resolution with ambiguity detection
	var matches []*schema.Enum
	var matchedNames []string

	for fullName, enum := range r.enums {
		if strings.HasSuffix(fullName, "."+name) {
			matches = append(matches, enum)
			matchedNames = append(matchedNames, fullName)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("enum not found: %s", name)
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("ambiguous enum name '%s' matches multiple: %v. Use fully qualified name",
			name, matchedNames)
	}
}

// GetService retrieves a service definition by name
// Supports both fully qualified names and short names with ambiguity detection
func (r *Registry) GetService(name string) (*schema.Service, error) {
	// First: try exact match (fully qualified or already unique)
	if service, exists := r.services[name]; exists {
		return service, nil
	}

	// Second: try short name resolution with ambiguity detection
	var matches []*schema.Service
	var matchedNames []string

	for fullName, service := range r.services {
		if strings.HasSuffix(fullName, "."+name) {
			matches = append(matches, service)
			matchedNames = append(matchedNames, fullName)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("service not found: %s", name)
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("ambiguous service name '%s' matches multiple: %v. Use fully qualified name",
			name, matchedNames)
	}
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
