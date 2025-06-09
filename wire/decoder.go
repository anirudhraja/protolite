package wire

import (
	"encoding/binary"
	"fmt"
	"math"

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

// Entry point functions for API layer

// DecodeMessage decodes protobuf bytes using schema - main entry point
func DecodeMessage(data []byte, msg *schema.Message, registry *registry.Registry) (map[string]interface{}, error) {
	decoder := NewDecoderWithRegistry(data, registry)
	return decoder.DecodeWithSchema(msg)
}

// EncodeMessage encodes a map to protobuf bytes using schema - main entry point
func EncodeMessage(data map[string]interface{}, msg *schema.Message, registry *registry.Registry) ([]byte, error) {
	encoder := NewEncoder()

	for fieldName, value := range data {
		field := findFieldByName(msg, fieldName)
		if field == nil {
			continue // Skip unknown fields
		}

		err := encodeFieldWithSchema(encoder, field, value, registry)
		if err != nil {
			return nil, err
		}
	}

	return encoder.Bytes(), nil
}

// DecodeWithSchema decodes a message using schema information
func (d *Decoder) DecodeWithSchema(msg *schema.Message) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	mapCollector := make(map[string]map[interface{}]interface{})

	for d.pos < len(d.buf) {
		// Read field tag
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

		// Decode based on field type
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

// DecodeTypedField decodes a field based on its schema type
func (d *Decoder) DecodeTypedField(fieldType *schema.FieldType, wireType WireType) (interface{}, error) {
	switch fieldType.Kind {
	case schema.KindPrimitive:
		return d.decodePrimitive(fieldType.PrimitiveType, wireType)
	case schema.KindMessage:
		return d.decodeMessage(fieldType.MessageType, wireType)
	case schema.KindEnum:
		return d.decodeEnum(fieldType.EnumType, wireType)
	case schema.KindMap:
		return d.decodeMap(fieldType.MapKey, fieldType.MapValue, wireType)
	default:
		return d.decodeRawValue(wireType)
	}
}

// decodePrimitive decodes a primitive type
func (d *Decoder) decodePrimitive(primitiveType schema.PrimitiveType, wireType WireType) (interface{}, error) {
	switch wireType {
	case WireVarint:
		rawValue, err := d.DecodeVarint()
		if err != nil {
			return nil, err
		}
		return d.convertPrimitiveValue(primitiveType, rawValue), nil
	case WireFixed32:
		rawValue, err := d.DecodeFixed32()
		if err != nil {
			return nil, err
		}
		if primitiveType == schema.TypeFloat {
			return DecodeFloat32(rawValue), nil
		}
		return rawValue, nil
	case WireFixed64:
		rawValue, err := d.DecodeFixed64()
		if err != nil {
			return nil, err
		}
		if primitiveType == schema.TypeDouble {
			return DecodeFloat64(rawValue), nil
		}
		return rawValue, nil
	case WireBytes:
		rawValue, err := d.DecodeBytes()
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

// decodeMessage decodes a nested message
func (d *Decoder) decodeMessage(messageType string, wireType WireType) (interface{}, error) {
	if wireType != WireBytes {
		return nil, fmt.Errorf("message must use wire type bytes")
	}

	messageBytes, err := d.DecodeBytes()
	if err != nil {
		return nil, err
	}

	if d.registry == nil {
		return messageBytes, nil
	}

	msg, err := d.registry.GetMessage(messageType)
	if err != nil {
		return messageBytes, nil
	}

	// Recursively decode nested message
	nestedDecoder := NewDecoderWithRegistry(messageBytes, d.registry)
	return nestedDecoder.DecodeWithSchema(msg)
}

// decodeEnum decodes an enum value
func (d *Decoder) decodeEnum(enumType string, wireType WireType) (interface{}, error) {
	if wireType != WireVarint {
		return nil, fmt.Errorf("enum must use wire type varint")
	}

	enumNumber, err := d.DecodeVarint()
	if err != nil {
		return nil, err
	}

	return int32(enumNumber), nil
}

// decodeMap decodes a map entry
func (d *Decoder) decodeMap(keyType, valueType *schema.FieldType, wireType WireType) (interface{}, error) {
	if wireType != WireBytes {
		return nil, fmt.Errorf("map entry must use wire type bytes")
	}

	entryBytes, err := d.DecodeBytes()
	if err != nil {
		return nil, err
	}

	entryDecoder := NewDecoderWithRegistry(entryBytes, d.registry)

	var key, value interface{}
	for entryDecoder.pos < len(entryDecoder.buf) {
		tag, err := entryDecoder.DecodeVarint()
		if err != nil {
			return nil, err
		}

		fieldNumber, wireType := ParseTag(Tag(tag))
		switch fieldNumber {
		case 1: // key field
			key, err = entryDecoder.DecodeTypedField(keyType, wireType)
			if err != nil {
				return nil, fmt.Errorf("failed to decode map key: %v", err)
			}
		case 2: // value field
			value, err = entryDecoder.DecodeTypedField(valueType, wireType)
			if err != nil {
				return nil, fmt.Errorf("failed to decode map value: %v", err)
			}
		default:
			err = entryDecoder.skipField(wireType)
			if err != nil {
				return nil, err
			}
		}
	}

	return map[string]interface{}{
		"key":   key,
		"value": value,
	}, nil
}

// convertPrimitiveValue converts raw varint to proper primitive type
func (d *Decoder) convertPrimitiveValue(primitiveType schema.PrimitiveType, rawValue uint64) interface{} {
	switch primitiveType {
	case schema.TypeInt32:
		return int32(rawValue)
	case schema.TypeInt64:
		return int64(rawValue)
	case schema.TypeUint32:
		return uint32(rawValue)
	case schema.TypeUint64:
		return rawValue
	case schema.TypeSint32:
		return int32(DecodeSignedVarint(rawValue))
	case schema.TypeSint64:
		return DecodeSignedVarint(rawValue)
	case schema.TypeBool:
		return rawValue != 0
	default:
		return rawValue
	}
}

// decodeRawValue decodes without type information
func (d *Decoder) decodeRawValue(wireType WireType) (interface{}, error) {
	switch wireType {
	case WireVarint:
		return d.DecodeVarint()
	case WireFixed64:
		return d.DecodeFixed64()
	case WireBytes:
		return d.DecodeBytes()
	case WireFixed32:
		return d.DecodeFixed32()
	default:
		return nil, fmt.Errorf("unknown wire type: %d", wireType)
	}
}

// skipField skips a field based on wire type
func (d *Decoder) skipField(wireType WireType) error {
	switch wireType {
	case WireVarint:
		_, err := d.DecodeVarint()
		return err
	case WireFixed64:
		_, err := d.DecodeFixed64()
		return err
	case WireBytes:
		_, err := d.DecodeBytes()
		return err
	case WireFixed32:
		_, err := d.DecodeFixed32()
		return err
	default:
		return fmt.Errorf("unknown wire type: %d", wireType)
	}
}

// Helper functions

func findFieldByName(msg *schema.Message, fieldName string) *schema.Field {
	for _, field := range msg.Fields {
		if field.Name == fieldName {
			return field
		}
	}
	return nil
}

func encodeFieldWithSchema(encoder *Encoder, field *schema.Field, value interface{}, registry *registry.Registry) error {
	fieldNumber := FieldNumber(field.Number)
	fieldType := &field.Type

	switch fieldType.Kind {
	case schema.KindPrimitive:
		return encodePrimitiveWithType(encoder, fieldNumber, fieldType.PrimitiveType, value)
	case schema.KindMessage:
		return encodeMessageWithType(encoder, fieldNumber, fieldType.MessageType, value, registry)
	case schema.KindEnum:
		return encodeEnumWithType(encoder, fieldNumber, fieldType.EnumType, value, registry)
	case schema.KindMap:
		return encodeMapWithType(encoder, fieldNumber, fieldType.MapKey, fieldType.MapValue, value, registry)
	default:
		return fmt.Errorf("unsupported field type: %s", fieldType.Kind)
	}
}

func encodePrimitiveWithType(encoder *Encoder, fieldNumber FieldNumber, primitiveType schema.PrimitiveType, value interface{}) error {
	switch primitiveType {
	case schema.TypeString:
		return encoder.EncodeField(fieldNumber, WireBytes, value)
	case schema.TypeBytes:
		return encoder.EncodeField(fieldNumber, WireBytes, value)
	case schema.TypeInt32, schema.TypeInt64, schema.TypeUint32, schema.TypeUint64:
		return encoder.EncodeField(fieldNumber, WireVarint, value)
	case schema.TypeSint32, schema.TypeSint64:
		var encodedValue uint64
		switch v := value.(type) {
		case int32:
			encodedValue = EncodeSignedVarint(int64(v))
		case int64:
			encodedValue = EncodeSignedVarint(v)
		default:
			return fmt.Errorf("invalid signed int value: %T", value)
		}
		return encoder.EncodeField(fieldNumber, WireVarint, encodedValue)
	case schema.TypeFixed32, schema.TypeSfixed32:
		return encoder.EncodeField(fieldNumber, WireFixed32, value)
	case schema.TypeFixed64, schema.TypeSfixed64:
		return encoder.EncodeField(fieldNumber, WireFixed64, value)
	case schema.TypeFloat:
		return encoder.EncodeField(fieldNumber, WireFixed32, value)
	case schema.TypeDouble:
		return encoder.EncodeField(fieldNumber, WireFixed64, value)
	case schema.TypeBool:
		return encoder.EncodeField(fieldNumber, WireVarint, value)
	default:
		return fmt.Errorf("unsupported primitive type: %s", primitiveType)
	}
}

func encodeMessageWithType(encoder *Encoder, fieldNumber FieldNumber, messageType string, value interface{}, registry *registry.Registry) error {
	nestedData, ok := value.(map[string]interface{})
	if !ok {
		return fmt.Errorf("nested message must be map[string]interface{}, got %T", value)
	}

	msg, err := registry.GetMessage(messageType)
	if err != nil {
		return err
	}

	nestedBytes, err := EncodeMessage(nestedData, msg, registry)
	if err != nil {
		return err
	}

	return encoder.EncodeField(fieldNumber, WireBytes, nestedBytes)
}

func encodeEnumWithType(encoder *Encoder, fieldNumber FieldNumber, enumType string, value interface{}, registry *registry.Registry) error {
	_, err := registry.GetEnum(enumType)
	if err != nil {
		return err
	}

	var enumNumber int32
	switch v := value.(type) {
	case int32:
		enumNumber = v
	case int:
		enumNumber = int32(v)
	default:
		return fmt.Errorf("enum encoding not fully implemented, got %T", value)
	}

	return encoder.EncodeField(fieldNumber, WireVarint, enumNumber)
}

func encodeMapWithType(encoder *Encoder, fieldNumber FieldNumber, keyType, valueType *schema.FieldType, value interface{}, registry *registry.Registry) error {
	mapData := normalizeMapData(value)
	if mapData == nil {
		return fmt.Errorf("map field must be a map type, got %T", value)
	}

	// Encode each map entry as a message with key=1, value=2
	for key, val := range mapData {
		entryEncoder := NewEncoder()

		// Encode key (field 1)
		err := encodeFieldWithType(entryEncoder, FieldNumber(1), keyType, key, registry)
		if err != nil {
			return fmt.Errorf("failed to encode map key: %v", err)
		}

		// Encode value (field 2)
		err = encodeFieldWithType(entryEncoder, FieldNumber(2), valueType, val, registry)
		if err != nil {
			return fmt.Errorf("failed to encode map value: %v", err)
		}

		// Encode the complete entry
		entryBytes := entryEncoder.Bytes()
		err = encoder.EncodeField(fieldNumber, WireBytes, entryBytes)
		if err != nil {
			return err
		}
	}

	return nil
}

func encodeFieldWithType(encoder *Encoder, fieldNumber FieldNumber, fieldType *schema.FieldType, value interface{}, registry *registry.Registry) error {
	switch fieldType.Kind {
	case schema.KindPrimitive:
		return encodePrimitiveWithType(encoder, fieldNumber, fieldType.PrimitiveType, value)
	case schema.KindMessage:
		return encodeMessageWithType(encoder, fieldNumber, fieldType.MessageType, value, registry)
	case schema.KindEnum:
		return encodeEnumWithType(encoder, fieldNumber, fieldType.EnumType, value, registry)
	default:
		return fmt.Errorf("unsupported field type in map: %s", fieldType.Kind)
	}
}

func normalizeMapData(value interface{}) map[interface{}]interface{} {
	switch v := value.(type) {
	case map[interface{}]interface{}:
		return v
	case map[string]interface{}:
		result := make(map[interface{}]interface{})
		for k, val := range v {
			result[k] = val
		}
		return result
	case map[string]string:
		result := make(map[interface{}]interface{})
		for k, val := range v {
			result[k] = val
		}
		return result
	case map[string]int32:
		result := make(map[interface{}]interface{})
		for k, val := range v {
			result[k] = val
		}
		return result
	case map[int32]interface{}:
		result := make(map[interface{}]interface{})
		for k, val := range v {
			result[k] = val
		}
		return result
	default:
		return nil
	}
}

// Low-level decoder methods

// DecodeMessage decodes all fields in a protobuf message (backward compatibility)
func (d *Decoder) DecodeMessage() ([]*Value, error) {
	var values []*Value

	for d.pos < len(d.buf) {
		value, err := d.DecodeField()
		if err != nil {
			return nil, err
		}
		if value == nil {
			break
		}
		values = append(values, value)
	}

	return values, nil
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

// DecodeVarint decodes a varint from the current position
func (d *Decoder) DecodeVarint() (uint64, error) {
	if d.pos >= len(d.buf) {
		return 0, fmt.Errorf("unexpected end of data")
	}

	var result uint64
	var shift uint

	for i := 0; i < 10; i++ {
		if d.pos >= len(d.buf) {
			return 0, fmt.Errorf("varint truncated")
		}

		b := d.buf[d.pos]
		d.pos++

		result |= uint64(b&0x7F) << shift
		if b&0x80 == 0 {
			return result, nil
		}
		shift += 7
	}

	return 0, fmt.Errorf("varint too long")
}

// DecodeFixed32 decodes a 32-bit fixed-width value
func (d *Decoder) DecodeFixed32() (uint32, error) {
	if d.pos+4 > len(d.buf) {
		return 0, fmt.Errorf("not enough data for fixed32")
	}

	value := binary.LittleEndian.Uint32(d.buf[d.pos:])
	d.pos += 4
	return value, nil
}

// DecodeFixed64 decodes a 64-bit fixed-width value
func (d *Decoder) DecodeFixed64() (uint64, error) {
	if d.pos+8 > len(d.buf) {
		return 0, fmt.Errorf("not enough data for fixed64")
	}

	value := binary.LittleEndian.Uint64(d.buf[d.pos:])
	d.pos += 8
	return value, nil
}

// DecodeBytes decodes a length-delimited byte array
func (d *Decoder) DecodeBytes() ([]byte, error) {
	length, err := d.DecodeVarint()
	if err != nil {
		return nil, err
	}

	if d.pos+int(length) > len(d.buf) {
		return nil, fmt.Errorf("bytes truncated")
	}

	data := make([]byte, length)
	copy(data, d.buf[d.pos:d.pos+int(length)])
	d.pos += int(length)

	return data, nil
}

// DecodeFloat32 decodes a 32-bit float from fixed32 data
func DecodeFloat32(data uint32) float32 {
	return math.Float32frombits(data)
}

// DecodeFloat64 decodes a 64-bit float from fixed64 data
func DecodeFloat64(data uint64) float64 {
	return math.Float64frombits(data)
}

// DecodeSignedVarint decodes a signed varint (zigzag encoding)
func DecodeSignedVarint(data uint64) int64 {
	return int64(data>>1) ^ -int64(data&1)
}
