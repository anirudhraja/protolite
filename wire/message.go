package wire

import (
	"fmt"

	"github.com/protolite/schema"
)

// MessageDecoder handles message decoding operations
type MessageDecoder struct {
	decoder *Decoder
}

// MessageEncoder handles message encoding operations
type MessageEncoder struct {
	encoder *Encoder
}

// NewMessageDecoder creates a new message decoder
func NewMessageDecoder(d *Decoder) *MessageDecoder {
	return &MessageDecoder{decoder: d}
}

// NewMessageEncoder creates a new message encoder
func NewMessageEncoder(e *Encoder) *MessageEncoder {
	return &MessageEncoder{encoder: e}
}

// DECODER METHODS

// DecodeMessage decodes a nested message
func (md *MessageDecoder) DecodeMessage(messageType string) (interface{}, error) {
	// Messages are encoded as length-delimited bytes
	bd := NewBytesDecoder(md.decoder)
	messageBytes, err := bd.DecodeBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to decode message bytes: %v", err)
	}

	if md.decoder.registry == nil {
		// No registry available, return raw bytes
		return messageBytes, nil
	}

	// Look up the message schema
	msg, err := md.decoder.registry.GetMessage(messageType)
	if err != nil {
		// Schema not found, return raw bytes
		return messageBytes, nil
	}

	// Recursively decode the nested message
	nestedDecoder := NewDecoderWithRegistry(messageBytes, md.decoder.registry)
	return nestedDecoder.DecodeWithSchema(msg)
}

// ENCODER METHODS

// EncodeMessage encodes a message with the given data
func (me *MessageEncoder) EncodeMessage(data map[string]interface{}, msg *schema.Message) error {
	// Create a temporary encoder for the message content
	messageEncoder := NewEncoder()
	messageEncoder.registry = me.encoder.registry

	// Encode each field
	for fieldName, fieldValue := range data {
		field := me.findFieldByName(msg, fieldName)
		if field == nil {
			continue // Skip unknown fields
		}

		// Encode field tag
		ve := NewVarintEncoder(messageEncoder)
		wireType := me.getWireType(&field.Type)
		tag := MakeTag(FieldNumber(field.Number), wireType)
		if err := ve.EncodeVarint(uint64(tag)); err != nil {
			return fmt.Errorf("failed to encode field tag for %s: %v", fieldName, err)
		}

		// Encode field value
		if err := me.encodeFieldValue(messageEncoder, fieldValue, field); err != nil {
			return fmt.Errorf("failed to encode field %s: %v", fieldName, err)
		}
	}

	// Add the message bytes to the main encoder
	me.encoder.buf = append(me.encoder.buf, messageEncoder.buf...)
	return nil
}

// encodeFieldValue encodes a field value based on its type
func (me *MessageEncoder) encodeFieldValue(encoder *Encoder, value interface{}, field *schema.Field) error {
	switch field.Type.Kind {
	case schema.KindPrimitive:
		return me.encodePrimitiveField(encoder, value, field.Type.PrimitiveType)
	case schema.KindMessage:
		return me.encodeMessageField(encoder, value)
	case schema.KindEnum:
		return me.encodeEnumField(encoder, value)
	case schema.KindMap:
		return me.encodeMapField(encoder, value, field)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Type.Kind)
	}
}

// encodePrimitiveField encodes a primitive field
func (me *MessageEncoder) encodePrimitiveField(encoder *Encoder, value interface{}, primitiveType schema.PrimitiveType) error {
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
	case schema.TypeFloat:
		fe := NewFixedEncoder(encoder)
		return fe.EncodeFloat32(value.(float32))
	case schema.TypeDouble:
		fe := NewFixedEncoder(encoder)
		return fe.EncodeFloat64(value.(float64))
	default:
		return fmt.Errorf("unsupported primitive type: %s", primitiveType)
	}
}

// encodeMessageField encodes a nested message field
func (me *MessageEncoder) encodeMessageField(encoder *Encoder, value interface{}) error {
	messageBytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("message value must be []byte")
	}

	be := NewBytesEncoder(encoder)
	return be.EncodeBytes(messageBytes)
}

// encodeEnumField encodes an enum field
func (me *MessageEncoder) encodeEnumField(encoder *Encoder, value interface{}) error {
	enumValue, ok := value.(int32)
	if !ok {
		return fmt.Errorf("enum value must be int32")
	}

	ve := NewVarintEncoder(encoder)
	return ve.EncodeEnum(enumValue)
}

// encodeMapField encodes a map field
func (me *MessageEncoder) encodeMapField(encoder *Encoder, value interface{}, field *schema.Field) error {
	mapData, ok := value.(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf("map value must be map[interface{}]interface{}")
	}

	mapEncoder := NewMapEncoder(encoder)
	for key, val := range mapData {
		if err := mapEncoder.EncodeMapEntry(key, val, field.Type.MapKey, field.Type.MapValue); err != nil {
			return err
		}
	}
	return nil
}

// UTILITY METHODS

// getWireType returns the wire type for a field type
func (me *MessageEncoder) getWireType(fieldType *schema.FieldType) WireType {
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
	case schema.KindMap:
		return WireBytes
	default:
		return WireVarint
	}
}

// findFieldByName finds a field by name in a message
func (me *MessageEncoder) findFieldByName(msg *schema.Message, fieldName string) *schema.Field {
	for _, field := range msg.Fields {
		if field.Name == fieldName {
			return field
		}
	}
	return nil
}

// Convenience methods for direct access (maintains backward compatibility)

// DecodeMessage - convenience method for main decoder
func (d *Decoder) DecodeMessage(messageType string) (interface{}, error) {
	md := NewMessageDecoder(d)
	return md.DecodeMessage(messageType)
}

// EncodeMessage - convenience method for main encoder
func (e *Encoder) EncodeMessage(data map[string]interface{}, msg *schema.Message) error {
	me := NewMessageEncoder(e)
	return me.EncodeMessage(data, msg)
}
