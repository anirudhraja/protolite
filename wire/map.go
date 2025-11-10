package wire

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"math"

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

    // Apply defaults if key or value missing per protobuf map semantics
    if key == nil {
        key = defaultValueForType(keyType)
    }
    if value == nil {
        value = defaultValueForType(valueType)
    }
    return key, value, nil
}

// ENCODER METHODS

// EncodeMapEntry encodes a map entry (key-value pair)
func (me *MapEncoder) EncodeMapEntry(key, value interface{}, keyType, valueType *schema.FieldType) error {
	// Create a temporary encoder for the entry
	entryEncoder := NewEncoder()
	entryEncoder.registry = me.encoder.registry

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

// NormalizeGenericMap converts any map[K]V into map[interface{}]interface{} with keys
// coerced to the appropriate protobuf key type based on the provided field schema.
func (me *MapEncoder) NormalizeGenericMap(value interface{}, field *schema.Field) (map[interface{}]interface{}, error) {
	rv := reflect.ValueOf(value)
	if !rv.IsValid() || rv.Kind() != reflect.Map {
		return nil, fmt.Errorf("unsupported map type: %T", value)
	}
	out := make(map[interface{}]interface{}, rv.Len())
	for _, mk := range rv.MapKeys() {
		// Convert key to appropriate type based on schema
		var keyIface interface{}
		switch field.Type.MapKey.PrimitiveType {
		case schema.TypeBool:
			if mk.Kind() == reflect.Bool { keyIface = mk.Bool() } else { keyIface = false }
		case schema.TypeString:
			if mk.Kind() == reflect.String { keyIface = mk.String() } else { keyIface = fmt.Sprintf("%v", mk.Interface()) }
		case schema.TypeInt32, schema.TypeSint32, schema.TypeSfixed32:
			keyIface = int32(mk.Convert(reflect.TypeOf(int64(0))).Int())
		case schema.TypeInt64, schema.TypeSint64, schema.TypeSfixed64:
			keyIface = mk.Convert(reflect.TypeOf(int64(0))).Int()
		case schema.TypeUint32, schema.TypeFixed32:
			keyIface = uint32(mk.Convert(reflect.TypeOf(uint64(0))).Uint())
		case schema.TypeUint64, schema.TypeFixed64:
			keyIface = mk.Convert(reflect.TypeOf(uint64(0))).Uint()
		default:
			keyIface = fmt.Sprintf("%v", mk.Interface())
		}
		mv := rv.MapIndex(mk).Interface()
		// If value is a map for message type, try convert to map[string]interface{}
		if field.Type.MapValue.Kind == schema.KindMessage {
			if mm, ok := mv.(map[string]interface{}); ok {
				out[keyIface] = mm
			} else {
				vm := reflect.ValueOf(mv)
				if vm.IsValid() && vm.Kind() == reflect.Map && vm.Type().Key().Kind() == reflect.String {
					inner := make(map[string]interface{}, vm.Len())
					for _, k2 := range vm.MapKeys() {
						inner[k2.String()] = vm.MapIndex(k2).Interface()
					}
					out[keyIface] = inner
				} else {
					out[keyIface] = mv
				}
			}
		} else {
			out[keyIface] = mv
		}
	}
	return out, nil
}

// defaultValueForType returns the protobuf default for a given field type.
func defaultValueForType(t *schema.FieldType) interface{} {
    switch t.Kind {
    case schema.KindPrimitive:
        switch t.PrimitiveType {
        case schema.TypeBytes:
            return []byte{}
        default:
            return getDefaultValue(t.PrimitiveType)
        }
    case schema.KindEnum:
        // Default enum value is 0
        return int32(0)
    case schema.KindMessage:
        // Empty message
        return map[string]interface{}{}
    default:
        return nil
    }
}

// parseStringToInt64 parses a string to int64 for map key conversion.
func parseStringToInt64(s string) (int64, error) {
	if strings.ContainsAny(s, ".eE") {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil { return 0, err }
		if f != math.Trunc(f) { return 0, fmt.Errorf("non-integer map key") }
		return int64(f), nil
	}
	return strconv.ParseInt(s, 10, 64)
}

// parseStringToUint64 parses a string to uint64 for map key conversion.
func parseStringToUint64(s string) (uint64, error) {
	if strings.ContainsAny(s, ".eE") {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil { return 0, err }
		if f < 0 || f != math.Trunc(f) { return 0, fmt.Errorf("non-integer map key") }
		return uint64(f), nil
	}
	return strconv.ParseUint(s, 10, 64)
}
