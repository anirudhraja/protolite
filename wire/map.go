package wire

import (
	"fmt"

	"github.com/protolite/schema"
)

// MapDecoder handles map decoding operations
type MapDecoder struct {
	decoder *Decoder
}

// MapEncoder handles map encoding operations
type MapEncoder struct {
	encoder *Encoder
}

// NewMapDecoder creates a new map decoder
func NewMapDecoder(d *Decoder) *MapDecoder {
	return &MapDecoder{decoder: d}
}

// NewMapEncoder creates a new map encoder
func NewMapEncoder(e *Encoder) *MapEncoder {
	return &MapEncoder{encoder: e}
}

// DECODER METHODS

// DecodeMapEntry decodes a map entry (key-value pair)
func (md *MapDecoder) DecodeMapEntry(keyType, valueType *schema.FieldType) (interface{}, interface{}, error) {
	// Read the length-delimited map entry
	bd := NewBytesDecoder(md.decoder)
	entryBytes, err := bd.DecodeBytes()
	if err != nil {
		return nil, nil, err
	}

	// Create a new decoder for the entry data
	entryDecoder := NewDecoder(entryBytes)
	entryDecoder.registry = md.decoder.registry

	var key, value interface{}

	for entryDecoder.pos < len(entryDecoder.buf) {
		tag, err := entryDecoder.DecodeVarint()
		if err != nil {
			return nil, nil, err
		}

		fieldNumber, wireType := ParseTag(Tag(tag))

		switch fieldNumber {
		case 1: // Key field
			key, err = md.decodeMapField(entryDecoder, keyType, wireType)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to decode map key: %v", err)
			}
		case 2: // Value field
			value, err = md.decodeMapField(entryDecoder, valueType, wireType)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to decode map value: %v", err)
			}
		default:
			// Skip unknown fields
			if err := md.skipField(entryDecoder, wireType); err != nil {
				return nil, nil, err
			}
		}
	}

	return key, value, nil
}

// DecodeMap decodes a complete map field
func (md *MapDecoder) DecodeMap(keyType, valueType *schema.FieldType) (map[interface{}]interface{}, error) {
	result := make(map[interface{}]interface{})

	key, value, err := md.DecodeMapEntry(keyType, valueType)
	if err != nil {
		return nil, err
	}

	result[key] = value
	return result, nil
}

// decodeMapField decodes a single field within a map entry
func (md *MapDecoder) decodeMapField(decoder *Decoder, fieldType *schema.FieldType, wireType WireType) (interface{}, error) {
	switch fieldType.Kind {
	case schema.KindPrimitive:
		return md.decodePrimitiveField(decoder, fieldType.PrimitiveType, wireType)
	case schema.KindMessage:
		return md.decodeMessageField(decoder, fieldType.MessageType, wireType)
	case schema.KindEnum:
		return md.decodeEnumField(decoder, wireType)
	default:
		return nil, fmt.Errorf("unsupported map field type: %s", fieldType.Kind)
	}
}

// decodePrimitiveField decodes a primitive field
func (md *MapDecoder) decodePrimitiveField(decoder *Decoder, primitiveType schema.PrimitiveType, wireType WireType) (interface{}, error) {
	switch wireType {
	case WireVarint:
		vd := NewVarintDecoder(decoder)
		rawValue, err := vd.DecodeVarint()
		if err != nil {
			return nil, err
		}
		return md.convertPrimitiveValue(primitiveType, rawValue), nil
	case WireFixed32:
		fd := NewFixedDecoder(decoder)
		return fd.DecodeFixed32()
	case WireFixed64:
		fd := NewFixedDecoder(decoder)
		return fd.DecodeFixed64()
	case WireBytes:
		bd := NewBytesDecoder(decoder)
		data, err := bd.DecodeBytes()
		if err != nil {
			return nil, err
		}
		if primitiveType == schema.TypeString {
			return string(data), nil
		}
		return data, nil
	default:
		return nil, fmt.Errorf("invalid wire type %d for primitive %s", wireType, primitiveType)
	}
}

// decodeMessageField decodes a message field
func (md *MapDecoder) decodeMessageField(decoder *Decoder, messageType string, wireType WireType) (interface{}, error) {
	if wireType != WireBytes {
		return nil, fmt.Errorf("message must use wire type bytes")
	}

	bd := NewBytesDecoder(decoder)
	messageBytes, err := bd.DecodeBytes()
	if err != nil {
		return nil, err
	}

	if decoder.registry == nil {
		return messageBytes, nil
	}

	msg, err := decoder.registry.GetMessage(messageType)
	if err != nil {
		return messageBytes, nil
	}

	// Recursively decode nested message
	nestedDecoder := NewDecoderWithRegistry(messageBytes, decoder.registry)
	return nestedDecoder.DecodeWithSchema(msg)
}

// decodeEnumField decodes an enum field
func (md *MapDecoder) decodeEnumField(decoder *Decoder, wireType WireType) (interface{}, error) {
	if wireType != WireVarint {
		return nil, fmt.Errorf("enum must use wire type varint")
	}

	vd := NewVarintDecoder(decoder)
	enumNumber, err := vd.DecodeVarint()
	if err != nil {
		return nil, err
	}

	return int32(enumNumber), nil
}

// convertPrimitiveValue converts raw varint to proper primitive type
func (md *MapDecoder) convertPrimitiveValue(primitiveType schema.PrimitiveType, rawValue uint64) interface{} {
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
		return DecodeZigZag32(rawValue)
	case schema.TypeSint64:
		return DecodeZigZag64(rawValue)
	case schema.TypeBool:
		return rawValue != 0
	default:
		return rawValue
	}
}

// skipField skips a field based on wire type
func (md *MapDecoder) skipField(decoder *Decoder, wireType WireType) error {
	switch wireType {
	case WireVarint:
		vd := NewVarintDecoder(decoder)
		return vd.SkipVarint()
	case WireFixed64:
		decoder.pos += 8
		return nil
	case WireBytes:
		bd := NewBytesDecoder(decoder)
		return bd.SkipBytes()
	case WireFixed32:
		decoder.pos += 4
		return nil
	default:
		return fmt.Errorf("unknown wire type: %d", wireType)
	}
}

// ENCODER METHODS

// EncodeMapEntry encodes a map entry (key-value pair)
func (me *MapEncoder) EncodeMapEntry(key, value interface{}, keyType, valueType *schema.FieldType) error {
	// Create a temporary encoder for the entry
	entryEncoder := NewEncoder()

	// Encode key (field number 1)
	ve := NewVarintEncoder(entryEncoder)
	keyTag := MakeTag(FieldNumber(1), me.getWireType(keyType))
	if err := ve.EncodeVarint(uint64(keyTag)); err != nil {
		return err
	}
	if err := me.encodeMapField(entryEncoder, key, keyType); err != nil {
		return err
	}

	// Encode value (field number 2)
	valueTag := MakeTag(FieldNumber(2), me.getWireType(valueType))
	if err := ve.EncodeVarint(uint64(valueTag)); err != nil {
		return err
	}
	if err := me.encodeMapField(entryEncoder, value, valueType); err != nil {
		return err
	}

	// Encode the complete entry as length-delimited bytes
	be := NewBytesEncoder(me.encoder)
	return be.EncodeBytes(entryEncoder.buf)
}

// EncodeMap encodes a complete map
func (me *MapEncoder) EncodeMap(mapData map[interface{}]interface{}, keyType, valueType *schema.FieldType, fieldNumber int32) error {
	for key, value := range mapData {
		// Encode field tag
		ve := NewVarintEncoder(me.encoder)
		tag := MakeTag(FieldNumber(fieldNumber), WireBytes)
		if err := ve.EncodeVarint(uint64(tag)); err != nil {
			return err
		}

		// Encode map entry
		if err := me.EncodeMapEntry(key, value, keyType, valueType); err != nil {
			return err
		}
	}
	return nil
}

// encodeMapField encodes a single field within a map entry
func (me *MapEncoder) encodeMapField(encoder *Encoder, fieldValue interface{}, fieldType *schema.FieldType) error {
	switch fieldType.Kind {
	case schema.KindPrimitive:
		return me.encodePrimitiveField(encoder, fieldValue, fieldType.PrimitiveType)
	case schema.KindMessage:
		return me.encodeMessageField(encoder, fieldValue)
	case schema.KindEnum:
		return me.encodeEnumField(encoder, fieldValue)
	default:
		return fmt.Errorf("unsupported map field type: %s", fieldType.Kind)
	}
}

// encodePrimitiveField encodes a primitive field
func (me *MapEncoder) encodePrimitiveField(encoder *Encoder, value interface{}, primitiveType schema.PrimitiveType) error {
	switch primitiveType {
	case schema.TypeString:
		be := NewBytesEncoder(encoder)
		return be.EncodeString(value.(string))
	case schema.TypeBytes:
		be := NewBytesEncoder(encoder)
		return be.EncodeBytes(value.([]byte))
	case schema.TypeInt32:
		ve := NewVarintEncoder(encoder)
		return ve.EncodeInt32(value.(int32))
	case schema.TypeInt64:
		ve := NewVarintEncoder(encoder)
		return ve.EncodeInt64(value.(int64))
	case schema.TypeUint32:
		ve := NewVarintEncoder(encoder)
		return ve.EncodeUint32(value.(uint32))
	case schema.TypeUint64:
		ve := NewVarintEncoder(encoder)
		return ve.EncodeUint64(value.(uint64))
	case schema.TypeBool:
		ve := NewVarintEncoder(encoder)
		return ve.EncodeBool(value.(bool))
	default:
		return fmt.Errorf("unsupported primitive type: %s", primitiveType)
	}
}

// encodeMessageField encodes a message field
func (me *MapEncoder) encodeMessageField(encoder *Encoder, value interface{}) error {
	messageBytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("message value must be []byte")
	}

	be := NewBytesEncoder(encoder)
	return be.EncodeBytes(messageBytes)
}

// encodeEnumField encodes an enum field
func (me *MapEncoder) encodeEnumField(encoder *Encoder, value interface{}) error {
	enumValue, ok := value.(int32)
	if !ok {
		return fmt.Errorf("enum value must be int32")
	}

	ve := NewVarintEncoder(encoder)
	return ve.EncodeEnum(enumValue)
}

// getWireType returns the wire type for a field type
func (me *MapEncoder) getWireType(fieldType *schema.FieldType) WireType {
	switch fieldType.Kind {
	case schema.KindPrimitive:
		switch fieldType.PrimitiveType {
		case schema.TypeString, schema.TypeBytes:
			return WireBytes
		case schema.TypeFloat, schema.TypeFixed32, schema.TypeSfixed32:
			return WireFixed32
		case schema.TypeDouble, schema.TypeFixed64, schema.TypeSfixed64:
			return WireFixed64
		default:
			return WireVarint
		}
	case schema.KindMessage:
		return WireBytes
	case schema.KindEnum:
		return WireVarint
	default:
		return WireVarint
	}
}
