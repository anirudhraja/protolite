package wire

import (
	"fmt"

	"github.com/anirudhraja/protolite/schema"
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

		// Handle map fields specially
		if field.Type.Kind == schema.KindMap {
			if err := me.encodeMapField(messageEncoder, fieldValue, field); err != nil {
				return fmt.Errorf("failed to encode map field %s: %v", fieldName, err)
			}
			continue
		}

		// For repeated fields, encodeFieldValue handles the field tags
		if field.Label == schema.LabelRepeated {
			if err := me.encodeFieldValue(messageEncoder, fieldValue, field); err != nil {
				return fmt.Errorf("failed to encode field %s: %v", fieldName, err)
			}
		} else {
			// For non-repeated fields, encode field tag first
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
	}

	// Add the message bytes to the main encoder
	me.encoder.buf = append(me.encoder.buf, messageEncoder.buf...)
	return nil
}

// encodeFieldValue encodes a field value based on its type
func (me *MessageEncoder) encodeFieldValue(encoder *Encoder, value interface{}, field *schema.Field) error {
	// Handle repeated fields
	if field.Label == schema.LabelRepeated {
		return me.encodeRepeatedField(encoder, value, field)
	}

	switch field.Type.Kind {
	case schema.KindPrimitive:
		return me.encodePrimitiveField(encoder, value, field.Type.PrimitiveType)
	case schema.KindMessage:
		return me.encodeMessageField(encoder, value, field.Type.MessageType)
	case schema.KindEnum:
		return me.encodeEnumField(encoder, value)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Type.Kind)
	}
}

// encodeRepeatedField encodes a repeated field
func (me *MessageEncoder) encodeRepeatedField(encoder *Encoder, value interface{}, field *schema.Field) error {
	slice, ok := value.([]interface{})
	if !ok {
		// Try to convert []map[string]interface{} to []interface{}
		if mapSlice, ok := value.([]map[string]interface{}); ok {
			slice = make([]interface{}, len(mapSlice))
			for i, v := range mapSlice {
				slice[i] = v
			}
		} else if stringSlice, ok := value.([]string); ok {
			slice = make([]interface{}, len(stringSlice))
			for i, v := range stringSlice {
				slice[i] = v
			}
		} else {
			return fmt.Errorf("repeated field value must be a slice, got %T", value)
		}
	}

	// For each element in the slice, encode field tag + value
	for _, element := range slice {
		// Encode field tag for each element
		ve := NewVarintEncoder(encoder)
		wireType := me.getWireType(&field.Type)
		tag := MakeTag(FieldNumber(field.Number), wireType)
		if err := ve.EncodeVarint(uint64(tag)); err != nil {
			return fmt.Errorf("failed to encode repeated field tag: %v", err)
		}

		// Encode the element value
		switch field.Type.Kind {
		case schema.KindPrimitive:
			if err := me.encodePrimitiveField(encoder, element, field.Type.PrimitiveType); err != nil {
				return fmt.Errorf("failed to encode repeated primitive element: %v", err)
			}
		case schema.KindMessage:
			if err := me.encodeMessageField(encoder, element, field.Type.MessageType); err != nil {
				return fmt.Errorf("failed to encode repeated message element: %v", err)
			}
		case schema.KindEnum:
			if err := me.encodeEnumField(encoder, element); err != nil {
				return fmt.Errorf("failed to encode repeated enum element: %v", err)
			}
		default:
			return fmt.Errorf("unsupported repeated field type: %s", field.Type.Kind)
		}
	}

	return nil
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
func (me *MessageEncoder) encodeMessageField(encoder *Encoder, value interface{}, messageTypeName string) error {
	// If it's already bytes, encode directly
	if messageBytes, ok := value.([]byte); ok {
		be := NewBytesEncoder(encoder)
		return be.EncodeBytes(messageBytes)
	}

	// If it's a map, we need to encode it as a message
	messageData, ok := value.(map[string]interface{})
	if !ok {
		return fmt.Errorf("message value must be map[string]interface{} or []byte, got %T", value)
	}

	// Look up the message schema
	if me.encoder.registry == nil {
		return fmt.Errorf("registry is required to encode message fields")
	}

	messageSchema, err := me.encoder.registry.GetMessage(messageTypeName)
	if err != nil {
		return fmt.Errorf("failed to get message schema for %s: %v", messageTypeName, err)
	}

	// Create a temporary encoder for the nested message
	nestedEncoder := NewEncoder()
	nestedEncoder.registry = me.encoder.registry

	nestedMessageEncoder := NewMessageEncoder(nestedEncoder)
	if err := nestedMessageEncoder.EncodeMessage(messageData, messageSchema); err != nil {
		return fmt.Errorf("failed to encode nested message: %v", err)
	}

	// Encode the nested message bytes
	be := NewBytesEncoder(encoder)
	return be.EncodeBytes(nestedEncoder.Bytes())
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
	var mapData map[interface{}]interface{}

	// Handle different map types
	switch v := value.(type) {
	case map[interface{}]interface{}:
		mapData = v
	case map[string]interface{}:
		mapData = make(map[interface{}]interface{})
		for k, val := range v {
			mapData[k] = val
		}
	case map[string]string:
		mapData = make(map[interface{}]interface{})
		for k, val := range v {
			mapData[k] = val
		}
	default:
		return fmt.Errorf("map value must be map[string]string, map[string]interface{}, or map[interface{}]interface{}, got %T", value)
	}

	// Use the map encoder to encode the entire map with field tags
	mapEncoder := NewMapEncoder(encoder)
	return mapEncoder.EncodeMap(mapData, field.Type.MapKey, field.Type.MapValue, field.Number)
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

// Convenience methods for direct access (main maintains backward compatibility)

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
