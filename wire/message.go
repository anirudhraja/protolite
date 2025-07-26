package wire

import (
	"fmt"
	"sort"

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
	// To iterate over data in a sorted manner by field number, collect valid fields first.
	type fieldEntry struct {
		name   string
		value  interface{}
		number int32
		field  *schema.Field
	}
	var entries []fieldEntry
	for fieldName, fieldValue := range data {
		field := me.findFieldByName(msg, fieldName)
		if field == nil {
			continue // Skip unknown fields
		}
		entries = append(entries, fieldEntry{
			name:   fieldName,
			value:  fieldValue,
			number: field.Number,
			field:  field,
		})
	}
	// Sort entries by field number in increasing order.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].number < entries[j].number
	})

	for _, entry := range entries {
		fieldName := entry.name
		fieldValue := entry.value
		field := entry.field

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
		return me.encodeEnumField(encoder, value, field.Type)
	case schema.KindWrapper:
		return me.encodeWrapperField(encoder, value, field.Type.WrapperType)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Type.Kind)
	}
}

// encodeRepeatedField encodes a repeated field
func (me *MessageEncoder) encodeRepeatedField(encoder *Encoder, value interface{}, field *schema.Field) error {
	slice, ok := value.([]interface{})
	if !ok {
		// Try to convert different slice types to []interface{}
		switch v := value.(type) {
		case []map[string]interface{}:
			slice = make([]interface{}, len(v))
			for i, val := range v {
				slice[i] = val
			}
		case []string:
			slice = make([]interface{}, len(v))
			for i, val := range v {
				slice[i] = val
			}
		case []int32:
			slice = make([]interface{}, len(v))
			for i, val := range v {
				slice[i] = val
			}
		case []int64:
			slice = make([]interface{}, len(v))
			for i, val := range v {
				slice[i] = val
			}
		case []uint32:
			slice = make([]interface{}, len(v))
			for i, val := range v {
				slice[i] = val
			}
		case []uint64:
			slice = make([]interface{}, len(v))
			for i, val := range v {
				slice[i] = val
			}
		case []bool:
			slice = make([]interface{}, len(v))
			for i, val := range v {
				slice[i] = val
			}
		case []float32:
			slice = make([]interface{}, len(v))
			for i, val := range v {
				slice[i] = val
			}
		case []float64:
			slice = make([]interface{}, len(v))
			for i, val := range v {
				slice[i] = val
			}
		default:
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
			if err := me.encodeEnumField(encoder, element, field.Type); err != nil {
				return fmt.Errorf("failed to encode repeated enum element: %v", err)
			}
		case schema.KindWrapper:
			if err := me.encodeWrapperField(encoder, element, field.Type.WrapperType); err != nil {
				return fmt.Errorf("failed to encode repeated wrapper element: %v", err)
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
		v, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
		return NewBytesEncoder(encoder).EncodeString(v)
	case schema.TypeBytes:
		v, ok := value.([]byte)
		if !ok {
			return fmt.Errorf("expected []byte, got %T", value)
		}
		return NewBytesEncoder(encoder).EncodeBytes(v)
	case schema.TypeInt32:
		v, ok := value.(int32)
		if !ok {
			return fmt.Errorf("expected int32, got %T", value)
		}
		return NewVarintEncoder(encoder).EncodeInt32(v)
	case schema.TypeInt64:
		v, ok := value.(int64)
		if !ok {
			return fmt.Errorf("expected int64, got %T", value)
		}
		return NewVarintEncoder(encoder).EncodeInt64(v)
	case schema.TypeUint32:
		v, ok := value.(uint32)
		if !ok {
			return fmt.Errorf("expected uint32, got %T", value)
		}
		return NewVarintEncoder(encoder).EncodeUint32(v)
	case schema.TypeUint64:
		v, ok := value.(uint64)
		if !ok {
			return fmt.Errorf("expected uint64, got %T", value)
		}
		return NewVarintEncoder(encoder).EncodeUint64(v)
	case schema.TypeBool:
		v, ok := value.(bool)
		if !ok {
			return fmt.Errorf("expected bool, got %T", value)
		}
		return NewVarintEncoder(encoder).EncodeBool(v)
	case schema.TypeFloat:
		v, ok := value.(float32)
		if !ok {
			return fmt.Errorf("expected float32, got %T", value)
		}
		return NewFixedEncoder(encoder).EncodeFloat32(v)
	case schema.TypeDouble:
		v, ok := value.(float64)
		if !ok {
			return fmt.Errorf("expected float64, got %T", value)
		}
		return NewFixedEncoder(encoder).EncodeFloat64(v)
	case schema.TypeFixed32:
		v, ok := value.(uint32)
		if !ok {
			return fmt.Errorf("expected uint32, got %T", value)
		}
		return NewFixedEncoder(encoder).EncodeFixed32(v)
	case schema.TypeFixed64:
		v, ok := value.(uint64)
		if !ok {
			return fmt.Errorf("expected uint64, got %T", value)
		}
		return NewFixedEncoder(encoder).EncodeFixed64(v)
	case schema.TypeSfixed32:
		v, ok := value.(int32)
		if !ok {
			return fmt.Errorf("expected int32, got %T", value)
		}
		return NewFixedEncoder(encoder).EncodeSfixed32(v)
	case schema.TypeSfixed64:
		v, ok := value.(int64)
		if !ok {
			return fmt.Errorf("expected int64, got %T", value)
		}
		return NewFixedEncoder(encoder).EncodeSfixed64(v)
	case schema.TypeSint32:
		v, ok := value.(int32)
		if !ok {
			return fmt.Errorf("expected int32, got %T", value)
		}
		return NewFixedEncoder(encoder).EncodeSfixed32(v)
	case schema.TypeSint64:
		v, ok := value.(int64)
		if !ok {
			return fmt.Errorf("expected int64, got %T", value)
		}
		return NewFixedEncoder(encoder).EncodeSfixed64(v)
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
func (me *MessageEncoder) encodeEnumField(encoder *Encoder, value interface{}, fieldType schema.FieldType) error {
	enumValue, ok := value.(string)
	if !ok {
		return fmt.Errorf("enum value must be string for %s field", fieldType.EnumType)
	}
	enum, err := me.encoder.registry.GetEnum(fieldType.EnumType)
	if err != nil {
		return fmt.Errorf("unknown enum %s received for enum , with value %v", fieldType.EnumType, value)
	}
	for _, en := range enum.Values {
		if en.Name == enumValue {
			ve := NewVarintEncoder(encoder)
			return ve.EncodeEnum(en.Number)
		}
	}
	return fmt.Errorf("cannot find field value %s in the enum %v", enumValue, enum.Values)

}

// encodeWrapperField encodes a wrapper field
func (me *MessageEncoder) encodeWrapperField(encoder *Encoder, value interface{}, wrapperType schema.WrapperType) error {
	// If wrapper value is nil, don't encode anything (optional semantics)
	if value == nil {
		return nil
	}

	// Create a temporary encoder for the wrapper message
	wrapperEncoder := NewEncoder()
	wrapperEncoder.registry = me.encoder.registry

	// Encode the wrapper value field (field number 1)
	ve := NewVarintEncoder(wrapperEncoder)

	// Helper function to extract actual value from wrapper structure or primitive
	extractWrapperValue := func(v interface{}, expectedType string) (interface{}, error) {
		// If it's already a map with "value" key, extract the value
		if mapVal, ok := v.(map[string]interface{}); ok {
			if actualValue, exists := mapVal["value"]; exists {
				if isTypeMatch(actualValue, expectedType) {
					return actualValue, nil
				} else {
					return nil, fmt.Errorf("expected %s, got %T", expectedType, actualValue)
				}
			}
			return nil, fmt.Errorf("wrapper map must contain 'value' field")
		}
		// Otherwise, assume it's the primitive value directly
		if isTypeMatch(v, expectedType) {
			return v, nil
		}
		return nil, fmt.Errorf("expected %s, got %T", expectedType, v)

	}

	// Determine the wire type and encode the value based on wrapper type
	switch wrapperType {
	case schema.WrapperDoubleValue:
		actualValue, err := extractWrapperValue(value, "float64")
		if err != nil {
			return err
		}
		// Wire type: fixed64
		tag := MakeTag(FieldNumber(1), WireFixed64)
		if err := ve.EncodeVarint(uint64(tag)); err != nil {
			return fmt.Errorf("failed to encode wrapper field tag: %v", err)
		}
		fe := NewFixedEncoder(wrapperEncoder)
		if err := fe.EncodeFloat64(actualValue.(float64)); err != nil {
			return fmt.Errorf("failed to encode wrapper value: %v", err)
		}

	case schema.WrapperFloatValue:
		actualValue, err := extractWrapperValue(value, "float32")
		if err != nil {
			return err
		}
		// Wire type: fixed32
		tag := MakeTag(FieldNumber(1), WireFixed32)
		if err := ve.EncodeVarint(uint64(tag)); err != nil {
			return fmt.Errorf("failed to encode wrapper field tag: %v", err)
		}
		fe := NewFixedEncoder(wrapperEncoder)
		if err := fe.EncodeFloat32(actualValue.(float32)); err != nil {
			return fmt.Errorf("failed to encode wrapper value: %v", err)
		}

	case schema.WrapperInt64Value:
		actualValue, err := extractWrapperValue(value, "int64")
		if err != nil {
			return err
		}
		// Wire type: varint
		tag := MakeTag(FieldNumber(1), WireVarint)
		if err := ve.EncodeVarint(uint64(tag)); err != nil {
			return fmt.Errorf("failed to encode wrapper field tag: %v", err)
		}
		if err := ve.EncodeInt64(actualValue.(int64)); err != nil {
			return fmt.Errorf("failed to encode wrapper value: %v", err)
		}

	case schema.WrapperUInt64Value:
		actualValue, err := extractWrapperValue(value, "uint64")
		if err != nil {
			return err
		}
		// Wire type: varint
		tag := MakeTag(FieldNumber(1), WireVarint)
		if err := ve.EncodeVarint(uint64(tag)); err != nil {
			return fmt.Errorf("failed to encode wrapper field tag: %v", err)
		}
		if err := ve.EncodeUint64(actualValue.(uint64)); err != nil {
			return fmt.Errorf("failed to encode wrapper value: %v", err)
		}

	case schema.WrapperInt32Value:
		actualValue, err := extractWrapperValue(value, "int32")
		if err != nil {
			return err
		}
		// Wire type: varint
		tag := MakeTag(FieldNumber(1), WireVarint)
		if err := ve.EncodeVarint(uint64(tag)); err != nil {
			return fmt.Errorf("failed to encode wrapper field tag: %v", err)
		}
		if err := ve.EncodeInt32(actualValue.(int32)); err != nil {
			return fmt.Errorf("failed to encode wrapper value: %v", err)
		}

	case schema.WrapperUInt32Value:
		actualValue, err := extractWrapperValue(value, "uint32")
		if err != nil {
			return err
		}
		// Wire type: varint
		tag := MakeTag(FieldNumber(1), WireVarint)
		if err := ve.EncodeVarint(uint64(tag)); err != nil {
			return fmt.Errorf("failed to encode wrapper field tag: %v", err)
		}
		if err := ve.EncodeUint32(actualValue.(uint32)); err != nil {
			return fmt.Errorf("failed to encode wrapper value: %v", err)
		}

	case schema.WrapperBoolValue:
		actualValue, err := extractWrapperValue(value, "bool")
		if err != nil {
			return err
		}
		// Wire type: varint
		tag := MakeTag(FieldNumber(1), WireVarint)
		if err := ve.EncodeVarint(uint64(tag)); err != nil {
			return fmt.Errorf("failed to encode wrapper field tag: %v", err)
		}
		if err := ve.EncodeBool(actualValue.(bool)); err != nil {
			return fmt.Errorf("failed to encode wrapper value: %v", err)
		}

	case schema.WrapperStringValue:
		actualValue, err := extractWrapperValue(value, "string")
		if err != nil {
			return err
		}
		// Wire type: bytes
		tag := MakeTag(FieldNumber(1), WireBytes)
		if err := ve.EncodeVarint(uint64(tag)); err != nil {
			return fmt.Errorf("failed to encode wrapper field tag: %v", err)
		}
		be := NewBytesEncoder(wrapperEncoder)
		if err := be.EncodeString(actualValue.(string)); err != nil {
			return fmt.Errorf("failed to encode wrapper value: %v", err)
		}

	case schema.WrapperBytesValue:
		actualValue, err := extractWrapperValue(value, "[]byte")
		if err != nil {
			return err
		}
		// Wire type: bytes
		tag := MakeTag(FieldNumber(1), WireBytes)
		if err := ve.EncodeVarint(uint64(tag)); err != nil {
			return fmt.Errorf("failed to encode wrapper field tag: %v", err)
		}
		be := NewBytesEncoder(wrapperEncoder)
		if err := be.EncodeBytes(actualValue.([]byte)); err != nil {
			return fmt.Errorf("failed to encode wrapper value: %v", err)
		}

	default:
		return fmt.Errorf("unsupported wrapper type: %s", wrapperType)
	}

	// Now encode the wrapper message bytes as a length-delimited field
	be := NewBytesEncoder(encoder)
	return be.EncodeBytes(wrapperEncoder.Bytes())
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
			// If the map value is a message type, encode it first
			if field.Type.MapValue.Kind == schema.KindMessage {
				if messageData, ok := val.(map[string]interface{}); ok {
					// Get the message schema
					messageSchema, err := me.encoder.registry.GetMessage(field.Type.MapValue.MessageType)
					if err != nil {
						return fmt.Errorf("failed to get message schema for %s: %v", field.Type.MapValue.MessageType, err)
					}

					// Encode the message
					nestedEncoder := NewEncoder()
					nestedEncoder.registry = me.encoder.registry
					nestedMessageEncoder := NewMessageEncoder(nestedEncoder)
					if err := nestedMessageEncoder.EncodeMessage(messageData, messageSchema); err != nil {
						return fmt.Errorf("failed to encode nested message: %v", err)
					}

					mapData[k] = nestedEncoder.Bytes()
				} else {
					mapData[k] = val
				}
			} else {
				mapData[k] = val
			}
		}
	case map[string]string:
		mapData = make(map[interface{}]interface{})
		for k, val := range v {
			mapData[k] = val
		}
	case map[string]int64:
		mapData = make(map[interface{}]interface{})
		for k, val := range v {
			mapData[k] = val
		}
	case map[int32]string:
		mapData = make(map[interface{}]interface{})
		for k, val := range v {
			mapData[k] = val
		}
	case map[string]float64:
		mapData = make(map[interface{}]interface{})
		for k, val := range v {
			mapData[k] = val
		}
	default:
		return fmt.Errorf("unsupported map type: %T", value)
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
	case schema.KindWrapper:
		return WireBytes // Wrapper types are encoded as length-delimited messages
	default:
		return WireVarint
	}
}

// findFieldByName finds a field by name in a message
func (me *MessageEncoder) findFieldByName(msg *schema.Message, fieldName string) *schema.Field {
	for _, field := range msg.Fields {
		if field.Name == fieldName || field.JsonName == fieldName {
			return field
		}
	}
	for _, oneOf := range msg.OneofGroups {
		for _, field := range oneOf.Fields {
			if field.Name == fieldName || field.JsonName == fieldName {
				return field
			}
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

func isTypeMatch(val interface{}, expectedType string) bool {
	switch expectedType {
	case "float64":
		_, ok := val.(float64)
		return ok
	case "float32":
		_, ok := val.(float32)
		return ok
	case "int64":
		_, ok := val.(int64)
		return ok
	case "int32":
		_, ok := val.(int32)
		return ok
	case "uint64":
		_, ok := val.(uint64)
		return ok
	case "uint32":
		_, ok := val.(uint32)
		return ok
	case "bool":
		_, ok := val.(bool)
		return ok
	case "string":
		_, ok := val.(string)
		return ok
	case "[]byte":
		_, ok := val.([]byte)
		return ok
	default:
		return false
	}
}
