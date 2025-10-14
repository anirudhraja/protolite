package wire

import (
	"encoding/json"
	"fmt"

	"github.com/anirudhraja/protolite/registry"
	"github.com/anirudhraja/protolite/schema"
)

const gqlTypeNameField = "__typename"

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
func DecodeMessage(data []byte, msg *schema.Message, registry *registry.Registry) (interface{}, error) {
	decoder := NewDecoderWithRegistry(data, registry)
	return decoder.DecodeWithSchema(msg)
}

// Main decoding methods that orchestrate the individual decoders
func (d *Decoder) DecodeWithSchema(msg *schema.Message) (interface{}, error) {
	result := make(map[string]interface{})
	mapCollector := make(map[string]map[interface{}]interface{})
	repeatedCollector := make(map[string][]interface{})

	initNull(result, msg)

	for d.pos < len(d.buf) {
		// Read field tag using varint decoder
		tag, err := d.DecodeVarint()
		if err != nil {
			return nil, fmt.Errorf("failed to decode message %s: %v", msg.Name, err)
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
		// attempt to find it in oneof fields
		if field == nil {
			for _, f := range msg.OneofGroups {
				for _, oneOfField := range f.Fields {
					if oneOfField.Number == int32(fieldNumber) {
						field = oneOfField
						break
					}
				}
			}
		}
		// Unknown field - skip it
		if field == nil {
			err := d.skipField(wireType)
			if err != nil {
				return nil, fmt.Errorf("failed to decode message %s: %v", msg.Name, err)
			}
			continue
		}
		fieldName := getFieldName(field)
		// Decode using appropriate decoder
		value, isPackedType, err := d.DecodeTypedField(field, wireType)
		if err != nil {
			return nil, fmt.Errorf("failed to decode field %s: %v", field.Name, err)
		}

		// Handle different field types
		if field.Type.Kind == schema.KindMap {
			// Handle maps specially
			if mapCollector[fieldName] == nil {
				mapCollector[fieldName] = make(map[interface{}]interface{})
			}
			if entryMap, ok := value.(map[string]interface{}); ok {
				mapCollector[fieldName][entryMap["key"]] = entryMap["value"]
			}
		} else if field.Label == schema.LabelRepeated && !isPackedType {
			// Handle repeated fields
			if repeatedCollector[fieldName] == nil {
				repeatedCollector[fieldName] = make([]interface{}, 0)
			}
			repeatedCollector[fieldName] = append(repeatedCollector[fieldName], value)
		} else {
			// Handle regular fields
			result[fieldName] = value
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
	// if its primitive type , add all default values to the message
	for _, field := range msg.Fields {
		if field.Label == schema.LabelRepeated {
			continue
		}

		fieldName := getFieldName(field)
		// add default values only when its not present in result
		if _, ok := result[fieldName]; !ok {
			if field.Type.Kind == schema.KindPrimitive { // add default for primitive types except bytes
				result[fieldName] = getDefaultValue(field.Type.PrimitiveType)
			} else if field.Type.Kind == schema.KindEnum { // add default value 0 for enum cases
				enum, err := d.registry.GetEnum(field.Type.EnumType)
				if err != nil {
					return nil, err
				}
				enumDefaultStringVal, err := d.findEnumValue(enum, 0)
				if err != nil {
					return nil, err
				}
				result[fieldName] = enumDefaultStringVal
			}
		}
	}
	// when message is wrapper, empty message on wire means
	// two different values based on wrapped item type. If
	// wrapped item is of repeated type, it means empty list,
	// otherwise null.
	if msg.IsWrapper {
		var field *schema.Field
		if len(msg.Fields) > 0 {
			field = msg.Fields[0]
		}
		if len(msg.OneofGroups) > 0 {
			typeName := msg.OneofGroups[0].Fields[0].JsonName
			for k := range result {
				typeName = k
				break
			}
			if oneOfField := getOneOfField(msg, typeName); oneOfField != nil {
				field = oneOfField
			}
			if result[typeName] == nil {
				result[typeName] = make(map[string]interface{})
			}
			result[typeName].(map[string]interface{})[gqlTypeNameField] = typeName
		}
		wrappedVal := result[getFieldName(field)]
		if wrappedVal == nil {
			if msg.Fields[0].Label == schema.LabelRepeated {
				return []interface{}{}, nil
			}
			return nil, nil
		}
		return wrappedVal, nil
	}
	return result, nil
}

func initNull(result map[string]interface{}, msg *schema.Message) {
	if !msg.ShowNull {
		return
	}
	for _, field := range msg.Fields {
		result[getFieldName(field)] = nil
	}
}

// DecodeTypedField routes to the appropriate decoder based on field type
func (d *Decoder) DecodeTypedField(field *schema.Field, wireType WireType) (interface{}, bool, error) {
	fieldType := field.Type
	switch fieldType.Kind {
	case schema.KindPrimitive:
		return d.decodePrimitive(field, wireType)
	case schema.KindMessage:
		md := NewMessageDecoder(d)
		value, err := md.DecodeMessage(fieldType.MessageType)
		return value, false, err
	case schema.KindEnum:
		len := uint64(1)
		var err error
		result := make([]interface{}, 0)
		// first check if the enum is registered
		enum, err := d.registry.GetEnum(fieldType.EnumType)
		if err != nil {
			return nil, false, err
		}
		// get the length first if its repeated enum
		if field.Label == schema.LabelRepeated {
			vd := NewVarintDecoder(d)
			len, err = vd.DecodeVarint()
			if err != nil {
				return nil, false, err
			}
		}
		// one by one read the bytes and find the relevant field name for it.
		for i := 0; i < int(len); i++ {
			vd := NewVarintDecoder(d)
			enumIntVal, err := vd.DecodeEnum()
			if err != nil {
				return nil, false, err
			}
			enumStringVal, err := d.findEnumValue(enum, enumIntVal)
			if err != nil {
				return nil, false, err
			}
			result = append(result, enumStringVal)
		}
		// if its not repeated, return a single value
		if field.Label != schema.LabelRepeated {
			return result[0], false, nil // we are guaranteed atleast one value in the slice if we reach here
		}
		// otherwise return the slice and the whole list gets appended
		return result, true, nil

	case schema.KindMap:
		mapDecoder := NewMapDecoder(d)
		key, value, err := mapDecoder.DecodeMapEntry(fieldType.MapKey, fieldType.MapValue)
		if err != nil {
			return nil, false, err
		}
		// Return as a map entry object
		return map[string]interface{}{
			"key":   key,
			"value": value,
		}, false, nil
	case schema.KindWrapper:
		value, err := d.decodeWrapper(fieldType.WrapperType, wireType, field.JSONString)
		return value, false, err
	default:
		value, err := d.decodeRawValue(wireType)
		return value, false, err
	}
}

// decodePrimitive decodes a primitive type using the appropriate decoder
func (d *Decoder) decodePrimitive(field *schema.Field, wireType WireType) (interface{}, bool, error) {
	primitiveType := field.Type.PrimitiveType
	if wireType == WireBytes {
		if schema.IsPackedType(primitiveType) {
			// double check to ensure field is repeated
			if field.Label != schema.LabelRepeated {
				return nil, false, fmt.Errorf("wire type (2) for primitive scalars has to be repeated")
			}
			vd := NewVarintDecoder(d)
			length, err := vd.DecodeVarint()
			if err != nil {
				return nil, false, err
			}
			res := make([]interface{}, 0)
			for i := 0; i < int(length); i++ {
				val, err := d.decodePrimitiveHelper(primitiveType)
				if err != nil {
					return nil, false, err
				}
				res = append(res, val)
			}
			return res, true, nil
		} else {
			// for string and bytes , its never packed even its repeated so decode and return
			bd := NewBytesDecoder(d)
			rawValue, err := bd.DecodeBytes()
			if err != nil {
				return nil, false, err
			}
			if primitiveType == schema.TypeString {
				return string(rawValue), false, nil
			}
			return rawValue, false, nil
		}
	}
	// reached here means its a single value encoded i.e its not packed
	value, err := d.decodePrimitiveHelper(primitiveType)
	return value, false, err
}

func (d *Decoder) decodePrimitiveHelper(primitiveType schema.PrimitiveType) (any, error) {
	switch primitiveType {
	case schema.TypeInt32, schema.TypeInt64, schema.TypeUint32, schema.TypeUint64,
		schema.TypeSint32, schema.TypeSint64, schema.TypeBool:
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
		}

	case schema.TypeFixed32, schema.TypeFixed64,
		schema.TypeSfixed32, schema.TypeSfixed64,
		schema.TypeFloat, schema.TypeDouble:

		fd := NewFixedDecoder(d)

		switch primitiveType {
		case schema.TypeFixed32:
			return fd.DecodeFixed32()
		case schema.TypeFixed64:
			return fd.DecodeFixed64()
		case schema.TypeSfixed32:
			return fd.DecodeSfixed32()
		case schema.TypeSfixed64:
			return fd.DecodeSfixed64()
		case schema.TypeFloat:
			return fd.DecodeFloat32()
		case schema.TypeDouble:
			return fd.DecodeFloat64()
		}
	}
	return nil, fmt.Errorf("unsupported primitive type: %v", primitiveType)
}

// decodeWrapper decodes a wrapper type
func (d *Decoder) decodeWrapper(wrapperType schema.WrapperType, wireType WireType, jsonString bool) (interface{}, error) {
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
		if jsonString {
			data := make(map[string]interface{})
			_ = json.Unmarshal(stringBytes, &data)
			return data, nil
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

func getFieldName(field *schema.Field) string {
	if field.JsonName != "" {
		return field.JsonName
	}
	return field.Name
}

func getDefaultValue(pt schema.PrimitiveType) interface{} {
	switch pt {
	case schema.TypeDouble:
		return float64(0)
	case schema.TypeFloat:
		return float32(0)
	case schema.TypeInt64, schema.TypeSint64, schema.TypeSfixed64:
		return int64(0)
	case schema.TypeInt32, schema.TypeSint32, schema.TypeSfixed32:
		return int32(0)
	case schema.TypeUint64, schema.TypeFixed64:
		return uint64(0)
	case schema.TypeUint32, schema.TypeFixed32:
		return uint32(0)
	case schema.TypeBool:
		return false
	case schema.TypeString:
		return ""
	default:
		return nil // unknown type
	}
}

func (d *Decoder) findEnumValue(enum *schema.Enum, enumIntVal int32) (string, error) {
	for _, en := range enum.Values {
		if en.Number == enumIntVal {
			if en.JsonName != "" {
				return en.JsonName, nil
			}
			return en.Name, nil
		}
	}
	return "", fmt.Errorf("unknown enum field value %d received for enum field %#v", enumIntVal, enum)

}
