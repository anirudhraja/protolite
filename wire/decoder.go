package wire

import (
	"fmt"

	"github.com/anirudhraja/protolite/registry"
	"github.com/anirudhraja/protolite/schema"
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
	repeatedCollector := make(map[string][]interface{})

	for d.pos < len(d.buf) {
		// Read field tag using varint decoder
		tag, err := d.DecodeVarint()
		if err != nil {
			return nil, fmt.Errorf("failed to decode message %s: %v",msg.Name,err)
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
		for _, f := range msg.OneofGroups {
			for _, oneOfField := range f.Fields {
				if oneOfField.Number == int32(fieldNumber) {
					field = oneOfField
					break
				}
			}
		}

		if field == nil {
			// Unknown field - skip it
			err := d.skipField(wireType)
			if err != nil {
				return nil, fmt.Errorf("failed to decode message %s: %v",msg.Name,err)
			}
			continue
		}

		// Decode using appropriate decoder
		value, err := d.DecodeTypedField(&field.Type, wireType)
		if err != nil {
			return nil, fmt.Errorf("failed to decode field %s: %v", field.Name, err)
		}

		// Handle different field types
		if field.Type.Kind == schema.KindMap {
			// Handle maps specially
			if mapCollector[field.Name] == nil {
				mapCollector[field.Name] = make(map[interface{}]interface{})
			}
			if entryMap, ok := value.(map[string]interface{}); ok {
				mapCollector[field.Name][entryMap["key"]] = entryMap["value"]
			}
		} else if field.Label == schema.LabelRepeated {
			// Handle repeated fields
			if repeatedCollector[field.Name] == nil {
				repeatedCollector[field.Name] = make([]interface{}, 0)
			}
			repeatedCollector[field.Name] = append(repeatedCollector[field.Name], value)
		} else {
			// Handle regular fields
			result[field.Name] = value
		}
	}

	// Add collected maps to result
	for fieldName, mapData := range mapCollector {
		result[fieldName] = mapData
	}

	// Add collected repeated fields to result
	for fieldName, repeatedData := range repeatedCollector {
		result[fieldName] = repeatedData
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
		enumIntVal, err := vd.DecodeEnum()
		if err != nil {
			return nil, err
		}
		enum, err := d.registry.GetEnum(fieldType.EnumType)
		if err != nil {
			return nil, err
		}
		for _, en := range enum.Values {
			if en.Number == enumIntVal {
				return en.Name, nil
			}
		}
		return nil, fmt.Errorf("unknown enum field value %d received for enum field %v", enumIntVal, fieldType)
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
	case schema.KindWrapper:
		return d.decodeWrapper(fieldType.WrapperType, wireType)
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

// decodeWrapper decodes a wrapper type
func (d *Decoder) decodeWrapper(wrapperType schema.WrapperType, wireType WireType) (interface{}, error) {
	// Wrapper types are encoded as length-delimited messages
	if wireType != WireBytes {
		return nil, fmt.Errorf("wrapper type must use wire type bytes, got %d", wireType)
	}

	// Decode the wrapper message bytes
	bd := NewBytesDecoder(d)
	wrapperBytes, err := bd.DecodeBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to decode wrapper message bytes: %v", err)
	}

	// Create a new decoder for the wrapper message content
	wrapperDecoder := NewDecoder(wrapperBytes)

	// Decode the wrapper value field (field number 1)
	if wrapperDecoder.pos >= len(wrapperDecoder.buf) {
		switch wrapperType {
		case schema.WrapperDoubleValue:
			return float64(0), nil
		case schema.WrapperFloatValue:
			return float32(0), nil
		case schema.WrapperInt64Value:
			return int64(0), nil
		case schema.WrapperUInt64Value:
			return uint64(0), nil
		case schema.WrapperInt32Value:
			return int32(0), nil
		case schema.WrapperUInt32Value:
			return uint32(0), nil
		case schema.WrapperBoolValue:
			return false, nil
		case schema.WrapperStringValue:
			return "", nil
		case schema.WrapperBytesValue:
			return []byte{}, nil
		default:
			// Empty wrapper message means nil value
			return nil, nil
		}
	}

	// Decode the field tag
	tag, err := wrapperDecoder.DecodeVarint()
	if err != nil {
		return nil, fmt.Errorf("failed to decode wrapper field tag: %v", err)
	}

	fieldNumber, valueWireType := ParseTag(Tag(tag))
	if fieldNumber != 1 {
		return nil, fmt.Errorf("expected field number 1 in wrapper, got %d", fieldNumber)
	}

	// Decode the actual value based on wrapper type
	switch wrapperType {
	case schema.WrapperDoubleValue:
		if valueWireType != WireFixed64 {
			return nil, fmt.Errorf("expected fixed64 wire type for DoubleValue, got %d", valueWireType)
		}
		fd := NewFixedDecoder(wrapperDecoder)
		return fd.DecodeFloat64()

	case schema.WrapperFloatValue:
		if valueWireType != WireFixed32 {
			return nil, fmt.Errorf("expected fixed32 wire type for FloatValue, got %d", valueWireType)
		}
		fd := NewFixedDecoder(wrapperDecoder)
		return fd.DecodeFloat32()

	case schema.WrapperInt64Value:
		if valueWireType != WireVarint {
			return nil, fmt.Errorf("expected varint wire type for Int64Value, got %d", valueWireType)
		}
		vd := NewVarintDecoder(wrapperDecoder)
		rawValue, err := vd.DecodeVarint()
		if err != nil {
			return nil, err
		}
		return int64(rawValue), nil

	case schema.WrapperUInt64Value:
		if valueWireType != WireVarint {
			return nil, fmt.Errorf("expected varint wire type for UInt64Value, got %d", valueWireType)
		}
		vd := NewVarintDecoder(wrapperDecoder)
		return vd.DecodeVarint()

	case schema.WrapperInt32Value:
		if valueWireType != WireVarint {
			return nil, fmt.Errorf("expected varint wire type for Int32Value, got %d", valueWireType)
		}
		vd := NewVarintDecoder(wrapperDecoder)
		rawValue, err := vd.DecodeVarint()
		if err != nil {
			return nil, err
		}
		return int32(rawValue), nil

	case schema.WrapperUInt32Value:
		if valueWireType != WireVarint {
			return nil, fmt.Errorf("expected varint wire type for UInt32Value, got %d", valueWireType)
		}
		vd := NewVarintDecoder(wrapperDecoder)
		rawValue, err := vd.DecodeVarint()
		if err != nil {
			return nil, err
		}
		return uint32(rawValue), nil

	case schema.WrapperBoolValue:
		if valueWireType != WireVarint {
			return nil, fmt.Errorf("expected varint wire type for BoolValue, got %d", valueWireType)
		}
		vd := NewVarintDecoder(wrapperDecoder)
		rawValue, err := vd.DecodeVarint()
		if err != nil {
			return nil, err
		}
		return rawValue != 0, nil

	case schema.WrapperStringValue:
		if valueWireType != WireBytes {
			return nil, fmt.Errorf("expected bytes wire type for StringValue, got %d", valueWireType)
		}
		bd := NewBytesDecoder(wrapperDecoder)
		stringBytes, err := bd.DecodeBytes()
		if err != nil {
			return nil, err
		}
		return string(stringBytes), nil

	case schema.WrapperBytesValue:
		if valueWireType != WireBytes {
			return nil, fmt.Errorf("expected bytes wire type for BytesValue, got %d", valueWireType)
		}
		bd := NewBytesDecoder(wrapperDecoder)
		return bd.DecodeBytes()

	default:
		return nil, fmt.Errorf("unsupported wrapper type: %s", wrapperType)
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
