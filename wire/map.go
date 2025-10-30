package wire

import (
	"fmt"

	"github.com/anirudhraja/protolite/schema"
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
			key, _, err = entryDecoder.DecodeTypedField(&schema.Field{Type: *keyType}, wireType)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to decode map key: %v", err)
			}
		case 2: // Value field
			value, _, err = entryDecoder.DecodeTypedField(&schema.Field{Type: *valueType}, wireType)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to decode map value: %v", err)
			}
		default:
			// Skip unknown fields
			if err := entryDecoder.skipField(wireType); err != nil {
				return nil, nil, err
			}
		}
	}

	return key, value, nil
}

// ENCODER METHODS

// EncodeMapEntry encodes a map entry (key-value pair)
func (me *MapEncoder) EncodeMapEntry(key, value interface{}, keyType, valueType *schema.FieldType) error {
	// Create a temporary encoder for the entry
	entryEncoder := NewEncoder()

	// Encode key (field number 1)
	ve := NewVarintEncoder(entryEncoder)
	entMsg := NewMessageEncoder(entryEncoder)
	keyTag := MakeTag(FieldNumber(1), me.getWireType(keyType))
	ve.EncodeVarint(uint64(keyTag))
	if err := entMsg.encodeFieldValue(key, &schema.Field{Type: *keyType}); err != nil {
		return err
	}

	// Encode value (field number 2)
	valueTag := MakeTag(FieldNumber(2), me.getWireType(valueType))
	ve.EncodeVarint(uint64(valueTag))
	if err := entMsg.encodeFieldValue(value, &schema.Field{Type: *valueType}); err != nil {
		return err
	}

	// Encode the complete entry as length-delimited bytes
	be := NewBytesEncoder(me.encoder)
	be.EncodeBytes(entryEncoder.buf)
	return nil
}

// EncodeMap encodes a complete map
func (me *MapEncoder) EncodeMap(mapData map[interface{}]interface{}, keyType, valueType *schema.FieldType, fieldNumber int32) error {
	for key, value := range mapData {
		// Encode field tag
		ve := NewVarintEncoder(me.encoder)
		tag := MakeTag(FieldNumber(fieldNumber), WireBytes)
		ve.EncodeVarint(uint64(tag))

		// Encode map entry
		if err := me.EncodeMapEntry(key, value, keyType, valueType); err != nil {
			return err
		}
	}
	return nil
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
