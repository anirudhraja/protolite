package protolite

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/anirudhraja/protolite/registry"
	"github.com/anirudhraja/protolite/wire"
)

// Protolite is the main interface for the library.
type Protolite interface {
	// Parse parses the given data into a map of string to interface. This is used when schema is not known.
	Parse(data []byte) (map[string]interface{}, error)

	// MarshalWithSchema marshals data using a specific message schema
	MarshalWithSchema(data map[string]interface{}, messageName string) ([]byte, error)

	// UnmarshalWithSchema unmarshals data using a specific message schema
	UnmarshalWithSchema(data []byte, messageName string) (map[string]interface{}, error)

	// UnmarshalToStruct unmarshals protobuf data into a Go struct using reflection
	UnmarshalToStruct(data []byte, messageName string, v interface{}) error

	// LoadSchemaFromFile loads schema definitions from a .proto file
	LoadSchemaFromFile(protoPath string) error
}

type protolite struct {
	registry *registry.Registry
}

// Parse implements Protolite - parses protobuf data without schema knowledge.
func (p *protolite) Parse(data []byte) (map[string]interface{}, error) {
	if len(data) == 0 {
		return make(map[string]interface{}), nil
	}

	decoder := wire.NewDecoder(data)
	result := make(map[string]interface{})

	for {
		field, err := decoder.DecodeField()
		if err != nil {
			return nil, fmt.Errorf("failed to decode field: %v", err)
		}

		if field == nil {
			// End of data
			break
		}

		// Use field number as key since we don't have schema
		fieldKey := fmt.Sprintf("field_%d", field.FieldNumber)

		// Convert wire type to more readable format
		switch field.WireType {
		case wire.WireVarint:
			result[fieldKey] = map[string]interface{}{
				"type":  "varint",
				"value": field.Data,
			}
		case wire.WireFixed64:
			result[fieldKey] = map[string]interface{}{
				"type":  "fixed64",
				"value": field.Data,
			}
		case wire.WireBytes:
			result[fieldKey] = map[string]interface{}{
				"type":  "bytes",
				"value": field.Data,
			}
		case wire.WireFixed32:
			result[fieldKey] = map[string]interface{}{
				"type":  "fixed32",
				"value": field.Data,
			}
		default:
			result[fieldKey] = map[string]interface{}{
				"type":  "unknown",
				"value": field.Data,
			}
		}
	}

	return result, nil
}

// LoadSchemaFromFile loads schema definitions from a .proto file
func (p *protolite) LoadSchemaFromFile(protoPath string) error {
	return p.registry.LoadSchema(protoPath)
}

// Additional helper methods that require schema

// MarshalWithSchema marshals data using a specific message schema
func (p *protolite) MarshalWithSchema(data map[string]interface{}, messageName string) ([]byte, error) {
	message, err := p.registry.GetMessage(messageName)
	if err != nil {
		return nil, fmt.Errorf("message schema not found: %v", err)
	}

	return wire.EncodeMessage(data, message, p.registry)
}

// UnmarshalWithSchema unmarshals data using a specific message schema
func (p *protolite) UnmarshalWithSchema(data []byte, messageName string) (map[string]interface{}, error) {
	message, err := p.registry.GetMessage(messageName)
	if err != nil {
		return nil, fmt.Errorf("message schema not found: %v", err)
	}

	return wire.DecodeMessage(data, message, p.registry)
}

// UnmarshalToStruct unmarshals protobuf data into a Go struct using reflection
func (p *protolite) UnmarshalToStruct(data []byte, messageName string, v interface{}) error {
	// First unmarshal to map
	result, err := p.UnmarshalWithSchema(data, messageName)
	if err != nil {
		return err
	}

	// Use reflection to populate the struct
	return p.mapToStruct(result, v)
}

// mapToStruct uses reflection to copy map values to struct fields
func (p *protolite) mapToStruct(data map[string]interface{}, v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return errors.New("v must be a pointer to a struct")
	}

	rv = rv.Elem()
	rt := rv.Type()

	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		fieldType := rt.Field(i)

		if !field.CanSet() {
			continue
		}

		// Try to find matching data by field name with multiple strategies
		var value interface{}
		var found bool

		// Strategy 1: Check exact match
		if val, ok := data[fieldType.Name]; ok {
			value = val
			found = true
		}

		// Strategy 2: Check lowercase version
		if !found {
			lowerName := strings.ToLower(fieldType.Name)
			if val, ok := data[lowerName]; ok {
				value = val
				found = true
			}
		}

		// Strategy 3: Check snake_case conversion
		if !found {
			snakeName := toSnakeCase(fieldType.Name)
			if val, ok := data[snakeName]; ok {
				value = val
				found = true
			}
		}

		if !found {
			continue
		}

		// Set the field value with type conversion
		if err := p.setFieldValue(field, value); err != nil {
			return fmt.Errorf("failed to set field %s: %v", fieldType.Name, err)
		}
	}

	return nil
}

// setFieldValue sets a struct field value with appropriate type conversion
func (p *protolite) setFieldValue(field reflect.Value, value interface{}) error {
	if value == nil {
		return nil
	}

	rv := reflect.ValueOf(value)

	// Handle type conversions
	switch field.Kind() {
	case reflect.String:
		if rv.Kind() == reflect.String {
			field.SetString(rv.String())
		} else {
			return fmt.Errorf("cannot convert %T to string", value)
		}
	case reflect.Int, reflect.Int32, reflect.Int64:
		switch rv.Kind() {
		case reflect.Int, reflect.Int32, reflect.Int64:
			field.SetInt(rv.Int())
		default:
			return fmt.Errorf("cannot convert %T to int", value)
		}
	case reflect.Uint, reflect.Uint32, reflect.Uint64:
		switch rv.Kind() {
		case reflect.Uint, reflect.Uint32, reflect.Uint64:
			field.SetUint(rv.Uint())
		default:
			return fmt.Errorf("cannot convert %T to uint", value)
		}
	case reflect.Float32, reflect.Float64:
		switch rv.Kind() {
		case reflect.Float32, reflect.Float64:
			field.SetFloat(rv.Float())
		default:
			return fmt.Errorf("cannot convert %T to float", value)
		}
	case reflect.Bool:
		if rv.Kind() == reflect.Bool {
			field.SetBool(rv.Bool())
		} else {
			return fmt.Errorf("cannot convert %T to bool", value)
		}
	case reflect.Slice:
		if rv.Kind() == reflect.Slice {
			field.Set(rv)
		} else {
			return fmt.Errorf("cannot convert %T to slice", value)
		}
	default:
		// Try direct assignment
		if rv.Type().AssignableTo(field.Type()) {
			field.Set(rv)
		} else {
			return fmt.Errorf("cannot assign %T to %s", value, field.Type())
		}
	}

	return nil
}

// toSnakeCase converts CamelCase to snake_case
func toSnakeCase(s string) string {
	if len(s) == 0 {
		return s
	}

	var result []rune
	runes := []rune(s)

	for i, r := range runes {
		if 'A' <= r && r <= 'Z' {
			// Add underscore before uppercase letters when:
			// 1. Not at the beginning, AND
			// 2. Previous char is lowercase, OR
			// 3. Previous char is uppercase AND next char is lowercase (end of acronym)
			if i > 0 {
				prev := runes[i-1]
				needUnderscore := false

				// Case 1: Previous char is lowercase
				if 'a' <= prev && prev <= 'z' {
					needUnderscore = true
				}

				// Case 2: Previous char is uppercase AND next char is lowercase (acronym boundary)
				if 'A' <= prev && prev <= 'Z' && i+1 < len(runes) && 'a' <= runes[i+1] && runes[i+1] <= 'z' {
					needUnderscore = true
				}

				if needUnderscore {
					result = append(result, '_')
				}
			}
			// Convert to lowercase
			result = append(result, r-'A'+'a')
		} else {
			result = append(result, r)
		}
	}

	return string(result)
}

func NewProtolite() Protolite {
	return &protolite{
		registry: registry.NewRegistry(),
	}
}
