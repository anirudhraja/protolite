package wire

import (
	"fmt"

	"github.com/protolite/registry"
	"github.com/protolite/schema"
)

// Decoder handles low-level protobuf wire format decoding
type Decoder struct {
	buf      []byte
	pos      int
	registry *registry.Registry
}

// NewDecoder creates a new wire format decoder
func NewDecoder(data []byte) *Decoder {
	return &Decoder{
		buf: data,
		pos: 0,
	}
}

// NewDecoderWithRegistry creates a decoder with schema registry
func NewDecoderWithRegistry(data []byte, registry *registry.Registry) *Decoder {
	return &Decoder{
		buf:      data,
		pos:      0,
		registry: registry,
	}
}

// DecodeMessage decodes protobuf bytes using schema - main entry point
func DecodeMessage(data []byte, msg *schema.Message, registry *registry.Registry) (map[string]interface{}, error) {
	decoder := NewDecoderWithRegistry(data, registry)
	return decoder.DecodeWithSchema(msg)
}

// Main decoding methods that orchestrate the individual decoders
func (d *Decoder) DecodeWithSchema(msg *schema.Message) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	mapCollector := make(map[string]map[interface{}]interface{})

	for d.pos < len(d.buf) {
		// Read field tag using varint decoder
		tag, err := d.DecodeVarint()
		if err != nil {
			return nil, err
		}

		fieldNumber, wireType := ParseTag(Tag(tag))

		// Find field in schema
		var field *schema.Field
		for _, f := range msg.Fields {
			if f.Number == int32(fieldNumber) {
				field = f
				break
			}
		}

		if field == nil {
			// Unknown field - skip it
			err := d.skipField(wireType)
			if err != nil {
				return nil, err
			}
			continue
		}

		// Decode using appropriate decoder
		value, err := d.DecodeTypedField(&field.Type, wireType)
		if err != nil {
			return nil, fmt.Errorf("failed to decode field %s: %v", field.Name, err)
		}

		// Handle maps specially
		if field.Type.Kind == schema.KindMap {
			if mapCollector[field.Name] == nil {
				mapCollector[field.Name] = make(map[interface{}]interface{})
			}
			if entryMap, ok := value.(map[string]interface{}); ok {
				mapCollector[field.Name][entryMap["key"]] = entryMap["value"]
			}
		} else {
			result[field.Name] = value
		}
	}

	// Add collected maps to result
	for fieldName, mapData := range mapCollector {
		result[fieldName] = mapData
	}

	return result, nil
}

// DecodeTypedField routes to the appropriate decoder based on field type
func (d *Decoder) DecodeTypedField(fieldType *schema.FieldType, wireType WireType) (interface{}, error) {
	switch fieldType.Kind {
	case schema.KindPrimitive:
		return d.decodePrimitive(fieldType.PrimitiveType, wireType)
	case schema.KindMessage:
		md := NewMessageDecoder(d)
		return md.DecodeMessage(fieldType.MessageType)
	case schema.KindEnum:
		vd := NewVarintDecoder(d)
		return vd.DecodeEnum()
	case schema.KindMap:
		mapDecoder := NewMapDecoder(d)
		key, value, err := mapDecoder.DecodeMapEntry(fieldType.MapKey, fieldType.MapValue)
		if err != nil {
			return nil, err
		}
		// Return as a map entry object
		return map[string]interface{}{
			"key":   key,
			"value": value,
		}, nil
	default:
		return d.decodeRawValue(wireType)
	}
}

// decodePrimitive decodes a primitive type using the appropriate decoder
func (d *Decoder) decodePrimitive(primitiveType schema.PrimitiveType, wireType WireType) (interface{}, error) {
	switch wireType {
	case WireVarint:
		vd := NewVarintDecoder(d)
		rawValue, err := vd.DecodeVarint()
		if err != nil {
			return nil, err
		}
		// Convert primitive value inline
		switch primitiveType {
		case schema.TypeInt32:
			return int32(rawValue), nil
		case schema.TypeInt64:
			return int64(rawValue), nil
		case schema.TypeUint32:
			return uint32(rawValue), nil
		case schema.TypeUint64:
			return rawValue, nil
		case schema.TypeSint32:
			return DecodeZigZag32(rawValue), nil
		case schema.TypeSint64:
			return DecodeZigZag64(rawValue), nil
		case schema.TypeBool:
			return rawValue != 0, nil
		default:
			return rawValue, nil
		}
	case WireFixed32:
		fd := NewFixedDecoder(d)
		if primitiveType == schema.TypeFloat {
			return fd.DecodeFloat32()
		}
		return fd.DecodeFixed32()
	case WireFixed64:
		fd := NewFixedDecoder(d)
		if primitiveType == schema.TypeDouble {
			return fd.DecodeFloat64()
		}
		return fd.DecodeFixed64()
	case WireBytes:
		bd := NewBytesDecoder(d)
		rawValue, err := bd.DecodeBytes()
		if err != nil {
			return nil, err
		}
		if primitiveType == schema.TypeString {
			return string(rawValue), nil
		}
		return rawValue, nil
	default:
		return nil, fmt.Errorf("invalid wire type %d for primitive %s", wireType, primitiveType)
	}
}

// decodeRawValue decodes without type information
func (d *Decoder) decodeRawValue(wireType WireType) (interface{}, error) {
	switch wireType {
	case WireVarint:
		vd := NewVarintDecoder(d)
		return vd.DecodeVarint()
	case WireFixed64:
		fd := NewFixedDecoder(d)
		return fd.DecodeFixed64()
	case WireBytes:
		bd := NewBytesDecoder(d)
		return bd.DecodeBytes()
	case WireFixed32:
		fd := NewFixedDecoder(d)
		return fd.DecodeFixed32()
	default:
		return nil, fmt.Errorf("unknown wire type: %d", wireType)
	}
}

// skipField skips a field based on wire type
func (d *Decoder) skipField(wireType WireType) error {
	switch wireType {
	case WireVarint:
		vd := NewVarintDecoder(d)
		return vd.SkipVarint()
	case WireFixed64:
		if d.pos+8 > len(d.buf) {
			return fmt.Errorf("not enough data to skip fixed64")
		}
		d.pos += 8
		return nil
	case WireBytes:
		bd := NewBytesDecoder(d)
		return bd.SkipBytes()
	case WireFixed32:
		if d.pos+4 > len(d.buf) {
			return fmt.Errorf("not enough data to skip fixed32")
		}
		d.pos += 4
		return nil
	default:
		return fmt.Errorf("unknown wire type: %d", wireType)
	}
}

// DecodeField decodes a single field from the current position (backward compatibility)
func (d *Decoder) DecodeField() (*Value, error) {
	if d.pos >= len(d.buf) {
		return nil, nil
	}

	tag, err := d.DecodeVarint()
	if err != nil {
		return nil, err
	}

	fieldNumber, wireType := ParseTag(Tag(tag))

	data, err := d.decodeRawValue(wireType)
	if err != nil {
		return nil, err
	}

	return &Value{
		FieldNumber: fieldNumber,
		WireType:    wireType,
		Data:        data,
	}, nil
}
