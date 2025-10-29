package wire

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

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
		// Return error directly to avoid repetitive wrapping in recursive calls
		return nil, err
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
func (me *MessageEncoder) EncodeMessage(data interface{}, msg *schema.Message) error {
	var (
		messageData map[string]interface{}
		ok          bool
	)
	if data == nil {
		return nil
	}
	if msg.IsWrapper {
		// mostly a wrapper has single field, except the wrapper of an union.
		var field *schema.Field
		if len(msg.Fields) > 0 {
			field = msg.Fields[0]
		}
		if dataMap, ok := data.(map[string]interface{}); ok {
			if iTypeName, ok := dataMap[gqlTypeNameField]; ok {
				if oneOfField := getOneOfField(msg, iTypeName.(string)); oneOfField != nil {
					field = oneOfField
				}
			}
		}
		if field == nil {
			return fmt.Errorf("missing union field in %s", msg.Name)
		}
		messageData = map[string]interface{}{getFieldName(field): data}
	} else {
		// If it's a map, we need to encode it as a message
		messageData, ok = data.(map[string]interface{})
		if !ok {
			return fmt.Errorf("message value for field %s must be map[string]interface{}, got %T", msg.Name, data)
		}
	}
	return me.encodeMessage(messageData, msg)
}

func getOneOfField(msg *schema.Message, typeName string) *schema.Field {
	for _, oneOf := range msg.OneofGroups {
		for _, field := range oneOf.Fields {
			// json_name is overloaded in union wrapper to store __typename as it was unused.
			if field.JsonName == typeName {
				return field
			}
		}
	}
	return nil
}

// EncodeMessage encodes a message with the given data
func (me *MessageEncoder) encodeMessage(data map[string]interface{}, msg *schema.Message) error {
	// Encode each field
	// To iterate over data in a sorted manner by field number, collect valid fields first.
	type fieldEntry struct {
		name   string
		value  interface{}
		number int32
		field  *schema.Field
	}
	var entries []fieldEntry
	nullFields := make([]int32, 0)
	for fieldName, fieldValue := range data {
		field := me.findFieldByName(msg, fieldName)
		if field == nil {
			continue // Skip unknown fields
		}
		// if there is no value , no need to iterate over the key
		if fieldValue == nil {
			nullFields = append(nullFields, field.Number)
			continue
		}

		entries = append(entries, fieldEntry{
			name:   fieldName,
			value:  fieldValue,
			number: field.Number,
			field:  field,
		})
	}
	if msg.TrackNull {
		nullTrackerField := me.findFieldByName(msg, schema.NullTrackerFieldName)
		if nullTrackerField == nil {
			return fmt.Errorf("message %s is configured to track nulls but missing null tracker field", msg.Name)
		}
		entries = append(entries, fieldEntry{
			name: schema.NullTrackerFieldName,
			value: map[string]interface{}{
				schema.NullTrackerWrapperInternalFieldName: map[string]interface{}{
					schema.NullTrackerNullFieldsFieldName: nullFields,
				},
			},
			number: nullTrackerField.Number,
			field:  nullTrackerField,
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
			if err := me.encodeMapField(fieldValue, field); err != nil {
				return wrapWithField(err, fieldName)
			}
			continue
		}

		// For repeated fields, encodeFieldValue handles the field tags
		if field.Label == schema.LabelRepeated {
			if err := me.encodeFieldValue(fieldValue, field); err != nil {
				return wrapWithField(err, fieldName)
			}

		} else {
			// For non-repeated fields, encode field tag first
			ve := NewVarintEncoder(me.encoder)
			wireType := me.getWireType(&field.Type)
			tag := MakeTag(FieldNumber(field.Number), wireType)
			ve.EncodeVarint(uint64(tag))

			// Encode field value
			if err := me.encodeFieldValue(fieldValue, field); err != nil {
				return wrapWithField(err, fieldName)
			}
		}
	}

	return nil
}

// encodeFieldValue encodes a field value based on its type
func (me *MessageEncoder) encodeFieldValue(value interface{}, field *schema.Field) error {
	// Handle repeated fields
	if field.Label == schema.LabelRepeated {
		return me.encodeRepeatedField(value, field)
	}
	if field.JSONString {
		b, _ := json.Marshal(value)
		value = string(b)
	}
	switch field.Type.Kind {
	case schema.KindPrimitive:
		return me.encodePrimitiveField(value, field.Type.PrimitiveType)
	case schema.KindMessage:
		return me.encodeMessageField(value, field.Type.MessageType)
	case schema.KindEnum:
		return me.encodeEnumField(value, field.Type)
	case schema.KindWrapper:
		return me.encodeWrapperField(value, field.Type.WrapperType)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Type.Kind)
	}
}

// encodeRepeatedField encodes a repeated field
func (me *MessageEncoder) encodeRepeatedField(value interface{}, field *schema.Field) error {
	if value == nil {
		return nil
	}

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
		case []json.Number:
			slice = make([]interface{}, len(v))
			for i, val := range v {
				slice[i] = val
			}
		default:
			return fmt.Errorf("repeated field value must be a slice, got %T", value)
		}
	}
	if field.JSONString {
		for i := 0; i < len(slice); i++ {
			b, _ := json.Marshal(slice[i])
			slice[i] = string(b)
		}
	}

	var packed bool
	if field.Type.Kind == schema.KindPrimitive {
		packed = schema.IsPackedType(field.Type.PrimitiveType)
	} else if field.Type.Kind == schema.KindEnum {
		packed = true
	}

	if packed {
		tag := MakeTag(FieldNumber(field.Number), WireBytes)
		NewVarintEncoder(me.encoder).EncodeVarint(uint64(tag))
		b := NewMessageEncoder(NewEncoderWithRegistry(me.encoder.registry))
		switch field.Type.Kind {
		case schema.KindPrimitive:
			for _, v := range slice {
				if err := b.encodePrimitiveField(v, field.Type.PrimitiveType); err != nil {
					return err
				}
			}
		case schema.KindEnum:
			for _, v := range slice {
				if err := b.encodeEnumField(v, field.Type); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("unexpected type %s for packed encoding", field.Type.Kind)
		}
		NewBytesEncoder(me.encoder).EncodeBytes(b.encoder.Bytes())
		return nil
	}

	// For each element in the slice, encode field tag + value
	for _, element := range slice {
		ve := NewVarintEncoder(me.encoder)
		// Encode field tag for each element
		wireType := me.getWireType(&field.Type)
		tag := MakeTag(FieldNumber(field.Number), wireType)
		ve.EncodeVarint(uint64(tag))

		// Encode the element value
		switch field.Type.Kind {
		case schema.KindPrimitive:
			if err := me.encodePrimitiveField(element, field.Type.PrimitiveType); err != nil {
				return err
			}
		case schema.KindMessage:
			if err := me.encodeMessageField(element, field.Type.MessageType); err != nil {
				return err
			}
		case schema.KindEnum:
			if err := me.encodeEnumField(element, field.Type); err != nil {
				return err
			}
		case schema.KindWrapper:
			if err := me.encodeWrapperField(element, field.Type.WrapperType); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported repeated field type: %s", field.Type.Kind)
		}
	}

	return nil
}

// encodePrimitiveField encodes a primitive field
func (me *MessageEncoder) encodePrimitiveField(value interface{}, primitiveType schema.PrimitiveType) error {
	encoder := me.encoder
	switch primitiveType {
	case schema.TypeString:
		v, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
		NewBytesEncoder(encoder).EncodeString(v)
		return nil
	case schema.TypeBytes:
		v, ok := value.([]byte)
		if !ok {
			return fmt.Errorf("expected []byte, got %T", value)
		}
		NewBytesEncoder(encoder).EncodeBytes(v)
		return nil
	case schema.TypeInt32:
		v, ok := value.(int32)
		if !ok {
			jsonVal, ok := value.(json.Number)
			if !ok {
				return fmt.Errorf("expected int32, got %T", value)
			}
			val, err := strconv.Atoi(jsonVal.String())
			if err != nil {
				return err
			}
			v = int32(val)
		}
		NewVarintEncoder(encoder).EncodeInt32(v)
		return nil
	case schema.TypeInt64:
		v, ok := value.(int64)
		if !ok {
			jsonVal, ok := value.(json.Number)
			if !ok {
				return fmt.Errorf("expected int64, got %T", value)
			}
			val, err := jsonVal.Int64()
			if err != nil {
				return err
			}
			v = val
		}
		NewVarintEncoder(encoder).EncodeInt64(v)
		return nil
	case schema.TypeUint32:
		v, ok := value.(uint32)
		if !ok {
			jsonVal, ok := value.(json.Number)
			if !ok {
				return fmt.Errorf("expected uint32, got %T", value)
			}
			val, err := strconv.ParseUint(jsonVal.String(), 10, 32)
			if err != nil {
				return err
			}
			v = uint32(val)
		}
		NewVarintEncoder(encoder).EncodeUint32(v)
		return nil
	case schema.TypeUint64:
		v, ok := value.(uint64)
		if !ok {
			jsonVal, ok := value.(json.Number)
			if !ok {
				return fmt.Errorf("expected uint64, got %T", value)
			}
			val, err := strconv.ParseUint(jsonVal.String(), 10, 64)
			if err != nil {
				return err
			}
			v = uint64(val)
		}
		NewVarintEncoder(encoder).EncodeUint64(v)
		return nil
	case schema.TypeBool:
		v, ok := value.(bool)
		if !ok {
			return fmt.Errorf("expected bool, got %T", value)
		}
		NewVarintEncoder(encoder).EncodeBool(v)
		return nil
	case schema.TypeFloat:
		v, ok := value.(float32)
		if !ok {
			jsonVal, ok := value.(json.Number)
			if !ok {
				return fmt.Errorf("expected float32, got %T", value)
			}
			val, err := strconv.ParseFloat(jsonVal.String(), 32)
			if err != nil {
				return err
			}
			v = float32(val)
		}
		return NewFixedEncoder(encoder).EncodeFloat32(v)
	case schema.TypeDouble:
		v, ok := value.(float64)
		if !ok {
			jsonVal, ok := value.(json.Number)
			if !ok {
				return fmt.Errorf("expected float64, got %T", value)
			}
			val, err := strconv.ParseFloat(jsonVal.String(), 64)
			if err != nil {
				return err
			}
			v = val
		}
		return NewFixedEncoder(encoder).EncodeFloat64(v)
	case schema.TypeFixed32:
		v, ok := value.(uint32)
		if !ok {
			jsonVal, ok := value.(json.Number)
			if !ok {
				return fmt.Errorf("expected uint32, got %T", value)
			}
			val, err := strconv.ParseUint(jsonVal.String(), 10, 32)
			if err != nil {
				return err
			}
			v = uint32(val)
		}
		return NewFixedEncoder(encoder).EncodeFixed32(v)
	case schema.TypeFixed64:
		v, ok := value.(uint64)
		if !ok {
			jsonVal, ok := value.(json.Number)
			if !ok {
				return fmt.Errorf("expected uint64, got %T", value)
			}
			val, err := strconv.ParseUint(jsonVal.String(), 10, 64)
			if err != nil {
				return err
			}
			v = uint64(val)
		}
		return NewFixedEncoder(encoder).EncodeFixed64(v)
	case schema.TypeSfixed32:
		v, ok := value.(int32)
		if !ok {
			jsonVal, ok := value.(json.Number)
			if !ok {
				return fmt.Errorf("expected int32, got %T", value)
			}
			val, err := strconv.Atoi(jsonVal.String())
			if err != nil {
				return err
			}
			v = int32(val)
		}
		return NewFixedEncoder(encoder).EncodeSfixed32(v)
	case schema.TypeSfixed64:
		v, ok := value.(int64)
		if !ok {
			jsonVal, ok := value.(json.Number)
			if !ok {
				return fmt.Errorf("expected int64, got %T", value)
			}
			val, err := jsonVal.Int64()
			if err != nil {
				return err
			}
			v = val
		}
		return NewFixedEncoder(encoder).EncodeSfixed64(v)
	case schema.TypeSint32:
		v, ok := value.(int32)
		if !ok {
			jsonVal, ok := value.(json.Number)
			if !ok {
				return fmt.Errorf("expected int32, got %T", value)
			}
			val, err := strconv.Atoi(jsonVal.String())
			if err != nil {
				return err
			}
			v = int32(val)
		}
		return NewFixedEncoder(encoder).EncodeSfixed32(v)
	case schema.TypeSint64:
		v, ok := value.(int64)
		if !ok {
			jsonVal, ok := value.(json.Number)
			if !ok {
				return fmt.Errorf("expected int64, got %T", value)
			}
			val, err := jsonVal.Int64()
			if err != nil {
				return err
			}
			v = val
		}
		return NewFixedEncoder(encoder).EncodeSfixed64(v)
	default:
		return fmt.Errorf("unsupported primitive type: %s", primitiveType)
	}
}

// encodeMessageField encodes a nested message field
func (me *MessageEncoder) encodeMessageField(value interface{}, messageTypeName string) error {
	encoder := me.encoder
	// If it's already bytes, encode directly
	if messageBytes, ok := value.([]byte); ok {
		be := NewBytesEncoder(encoder)
		be.EncodeBytes(messageBytes)
		return nil
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
	if err := nestedMessageEncoder.EncodeMessage(value, messageSchema); err != nil {
		return err
	}

	// Encode the nested message bytes
	be := NewBytesEncoder(encoder)
	be.EncodeBytes(nestedEncoder.Bytes())
	return nil
}

// encodeEnumField encodes an enum field
func (me *MessageEncoder) encodeEnumField(value interface{}, fieldType schema.FieldType) error {
	enumValue, ok := value.(string)
	if !ok {
		// check if the enum value is a JSON Number
		enumValueInJsonNumber, isJsonNumber := value.(json.Number)
		if !isJsonNumber {
			return fmt.Errorf("enum value must be string for %s field", fieldType.EnumType)
		}
		enumValue = enumValueInJsonNumber.String()
	}
	enum, err := me.encoder.registry.GetEnum(fieldType.EnumType)
	if err != nil {
		return fmt.Errorf("unknown enum %s received for enum, with value %v", fieldType.EnumType, value)
	}
	for _, en := range enum.Values {
		if en.Name == enumValue || en.JsonName == enumValue {
			ve := NewVarintEncoder(me.encoder)
			ve.EncodeEnum(en.Number)
			return nil
		}
	}
	return fmt.Errorf("cannot find field value %s in the enum %v", enumValue, enum.Values)

}

// encodeWrapperField encodes a wrapper field
func (me *MessageEncoder) encodeWrapperField(value interface{}, wrapperType schema.WrapperType) error {
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
	extractWrapperValue := func(v interface{}) (interface{}, error) {
		// If it's already a map with "value" key, extract the value
		if mapVal, ok := v.(map[string]interface{}); ok {
			if actualValue, exists := mapVal["value"]; exists {
				return actualValue, nil
			}
			return nil, fmt.Errorf("wrapper map must contain 'value' field")
		}
		return v, nil
	}

	// Determine the wire type and encode the value based on wrapper type
	switch wrapperType {
	case schema.WrapperDoubleValue:
		actualValue, err := extractWrapperValue(value)
		if err != nil {
			return err
		}
		var val float64
		switch v := actualValue.(type) {
		case float64:
			val = v
		case json.Number:
			val, err = v.Float64()
			if err != nil {
				return fmt.Errorf("invalid float64: %v", err)
			}
		default:
			return fmt.Errorf("unexpected type for float64: %T", actualValue)
		}
		tag := MakeTag(FieldNumber(1), WireFixed64)
		ve.EncodeVarint(uint64(tag))
		fe := NewFixedEncoder(wrapperEncoder)
		if err := fe.EncodeFloat64(val); err != nil {
			return err
		}

	case schema.WrapperFloatValue:
		actualValue, err := extractWrapperValue(value)
		if err != nil {
			return err
		}
		var val float32
		switch v := actualValue.(type) {
		case float32:
			val = v
		case json.Number:
			f64, err := strconv.ParseFloat(v.String(), 32)
			if err != nil {
				return fmt.Errorf("invalid float32: %v", err)
			}
			val = float32(f64)
		default:
			return fmt.Errorf("unexpected type for float32: %T", actualValue)
		}
		tag := MakeTag(FieldNumber(1), WireFixed32)
		ve.EncodeVarint(uint64(tag))
		fe := NewFixedEncoder(wrapperEncoder)
		if err := fe.EncodeFloat32(val); err != nil {
			return err
		}

	case schema.WrapperInt64Value:
		actualValue, err := extractWrapperValue(value)
		if err != nil {
			return err
		}
		var val int64
		switch v := actualValue.(type) {
		case int64:
			val = v
		case json.Number:
			val, err = v.Int64()
			if err != nil {
				return fmt.Errorf("invalid int64: %v", err)
			}
		default:
			return fmt.Errorf("unexpected type for int64: %T", actualValue)
		}
		tag := MakeTag(FieldNumber(1), WireVarint)
		ve.EncodeVarint(uint64(tag))
		ve.EncodeInt64(val)

	case schema.WrapperUInt64Value:
		actualValue, err := extractWrapperValue(value)
		if err != nil {
			return err
		}
		var val uint64
		switch v := actualValue.(type) {
		case uint64:
			val = v
		case json.Number:
			val, err = strconv.ParseUint(v.String(), 10, 64)
			if err != nil {
				return fmt.Errorf("invalid uint64: %v", err)
			}
		default:
			return fmt.Errorf("unexpected type for uint64: %T", actualValue)
		}
		tag := MakeTag(FieldNumber(1), WireVarint)
		ve.EncodeVarint(uint64(tag))
		ve.EncodeUint64(val)

	case schema.WrapperInt32Value:
		actualValue, err := extractWrapperValue(value)
		if err != nil {
			return err
		}
		var val int32
		switch v := actualValue.(type) {
		case int32:
			val = v
		case json.Number:
			var i64 int64
			i64, err = v.Int64()
			if err != nil {
				return fmt.Errorf("invalid int32: %v", err)
			}
			val = int32(i64)
		default:
			return fmt.Errorf("unexpected type for int32: %T", actualValue)
		}
		tag := MakeTag(FieldNumber(1), WireVarint)
		ve.EncodeVarint(uint64(tag))
		ve.EncodeInt32(val)

	case schema.WrapperUInt32Value:
		actualValue, err := extractWrapperValue(value)
		if err != nil {
			return err
		}
		var val uint32
		switch v := actualValue.(type) {
		case uint32:
			val = v
		case json.Number:
			var u64 uint64
			u64, err = strconv.ParseUint(v.String(), 10, 32)
			if err != nil {
				return fmt.Errorf("invalid uint32: %v", err)
			}
			val = uint32(u64)
		default:
			return fmt.Errorf("unexpected type for uint32: %T", actualValue)
		}
		tag := MakeTag(FieldNumber(1), WireVarint)
		ve.EncodeVarint(uint64(tag))
		ve.EncodeUint32(val)

	case schema.WrapperBoolValue:
		actualValue, err := extractWrapperValue(value)
		if err != nil {
			return err
		}
		val, ok := actualValue.(bool)
		if !ok {
			return fmt.Errorf("unexpected type for bool: %T", actualValue)
		}
		tag := MakeTag(FieldNumber(1), WireVarint)
		ve.EncodeVarint(uint64(tag))
		ve.EncodeBool(val)

	case schema.WrapperStringValue:
		actualValue, err := extractWrapperValue(value)
		if err != nil {
			return err
		}
		val, ok := actualValue.(string)
		if !ok {
			return fmt.Errorf("unexpected type for string: %T", actualValue)
		}
		tag := MakeTag(FieldNumber(1), WireBytes)
		ve.EncodeVarint(uint64(tag))
		be := NewBytesEncoder(wrapperEncoder)
		be.EncodeString(val)

	case schema.WrapperBytesValue:
		actualValue, err := extractWrapperValue(value)
		if err != nil {
			return err
		}
		val, ok := actualValue.([]byte)
		if !ok {
			return fmt.Errorf("unexpected type for bytes: %T", actualValue)
		}
		tag := MakeTag(FieldNumber(1), WireBytes)
		ve.EncodeVarint(uint64(tag))
		be := NewBytesEncoder(wrapperEncoder)
		be.EncodeBytes(val)

	default:
		return fmt.Errorf("unsupported wrapper type: %s", wrapperType)
	}

	// Now encode the wrapper message bytes as a length-delimited field
	be := NewBytesEncoder(me.encoder)
	be.EncodeBytes(wrapperEncoder.Bytes())
	return nil
}

// encodeMapField encodes a map field
func (me *MessageEncoder) encodeMapField(value interface{}, field *schema.Field) error {
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
						return err
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
	mapEncoder := NewMapEncoder(me.encoder)
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
