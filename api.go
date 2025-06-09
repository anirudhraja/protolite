package protolite

import (
	"fmt"
	"reflect"

	"github.com/protolite/registry"
	"github.com/protolite/schema"
	"github.com/protolite/wire"
)

// ===== SCHEMA-AWARE API =====

// Protolite provides schema-aware protobuf operations without generated code
type Protolite struct {
	registry *registry.Registry
}

// New creates a new Protolite instance
func New() *Protolite {
	return &Protolite{
		registry: registry.NewRegistry(),
	}
}

// LoadRepo loads a protobuf repository (collection of .proto files)
func (p *Protolite) LoadRepo(repo *schema.ProtoRepo) error {
	return p.registry.LoadRepo(repo)
}

// Parse decodes protobuf bytes using schema-aware decoder
func (p *Protolite) Parse(data []byte, messageType string) (map[string]interface{}, error) {
	msg, err := p.registry.GetMessage(messageType)
	if err != nil {
		return nil, fmt.Errorf("message type not found: %s", messageType)
	}

	// Direct call to schema-aware decoder - that's it!
	return wire.DecodeMessage(data, msg, p.registry)
}

// Marshal encodes a map to protobuf bytes using schema information
func (p *Protolite) Marshal(data map[string]interface{}, messageType string) ([]byte, error) {
	msg, err := p.registry.GetMessage(messageType)
	if err != nil {
		return nil, fmt.Errorf("message type not found: %s", messageType)
	}

	// Direct call to schema-aware encoder - that's it!
	return wire.EncodeMessage(data, msg, p.registry)
}

// Unmarshal decodes protobuf bytes into a Go struct using reflection
func (p *Protolite) Unmarshal(data []byte, v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("unmarshal target must be a pointer to struct")
	}

	messageType := rv.Elem().Type().Name()
	result, err := p.Parse(data, messageType)
	if err != nil {
		return err
	}

	return p.mapToStruct(result, rv.Elem())
}

// mapToStruct maps parsed result to struct fields
func (p *Protolite) mapToStruct(data map[string]interface{}, rv reflect.Value) error {
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		fieldValue := rv.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		if value, ok := data[field.Name]; ok {
			if err := p.setFieldValue(fieldValue, value); err != nil {
				return fmt.Errorf("failed to set field %s: %v", field.Name, err)
			}
		}
	}
	return nil
}

// setFieldValue sets a struct field with type conversion
func (p *Protolite) setFieldValue(fieldValue reflect.Value, value interface{}) error {
	if value == nil {
		return nil
	}

	sourceValue := reflect.ValueOf(value)
	if sourceValue.Type().AssignableTo(fieldValue.Type()) {
		fieldValue.Set(sourceValue)
		return nil
	}

	if sourceValue.Type().ConvertibleTo(fieldValue.Type()) {
		fieldValue.Set(sourceValue.Convert(fieldValue.Type()))
		return nil
	}

	return fmt.Errorf("cannot convert %T to %s", value, fieldValue.Type())
}

// ===== REGISTRY ACCESS =====

func (p *Protolite) GetRegistry() *registry.Registry { return p.registry }
func (p *Protolite) ListMessages() []string          { return p.registry.ListMessages() }
func (p *Protolite) ListEnums() []string             { return p.registry.ListEnums() }
func (p *Protolite) ListServices() []string          { return p.registry.ListServices() }
