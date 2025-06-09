package wire

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Encoder handles low-level protobuf wire format encoding
type Encoder struct {
	buf []byte
}

// NewEncoder creates a new wire format encoder
func NewEncoder() *Encoder {
	return &Encoder{
		buf: make([]byte, 0, 1024),
	}
}

// Bytes returns the encoded bytes
func (e *Encoder) Bytes() []byte {
	return e.buf
}

// Reset clears the encoder buffer
func (e *Encoder) Reset() {
	e.buf = e.buf[:0]
}

// EncodeField encodes a field with the given field number, wire type, and data
func (e *Encoder) EncodeField(fieldNumber FieldNumber, wireType WireType, data interface{}) error {
	// Encode tag (field number + wire type)
	tag := MakeTag(fieldNumber, wireType)
	e.EncodeVarint(uint64(tag))

	// Encode data based on wire type
	switch wireType {
	case WireVarint:
		switch v := data.(type) {
		case uint64:
			e.EncodeVarint(v)
		case int64:
			e.EncodeVarint(uint64(v))
		case uint32:
			e.EncodeVarint(uint64(v))
		case int32:
			e.EncodeVarint(uint64(v))
		case bool:
			if v {
				e.EncodeVarint(1)
			} else {
				e.EncodeVarint(0)
			}
		default:
			return fmt.Errorf("invalid varint data type: %T", data)
		}
	case WireFixed32:
		switch v := data.(type) {
		case uint32:
			e.EncodeFixed32(v)
		case float32:
			e.EncodeFixed32(math.Float32bits(v))
		default:
			return fmt.Errorf("invalid fixed32 data type: %T", data)
		}
	case WireFixed64:
		switch v := data.(type) {
		case uint64:
			e.EncodeFixed64(v)
		case float64:
			e.EncodeFixed64(math.Float64bits(v))
		default:
			return fmt.Errorf("invalid fixed64 data type: %T", data)
		}
	case WireBytes:
		switch v := data.(type) {
		case []byte:
			e.EncodeBytes(v)
		case string:
			e.EncodeBytes([]byte(v))
		default:
			return fmt.Errorf("invalid bytes data type: %T", data)
		}
	default:
		return fmt.Errorf("unknown wire type: %d", wireType)
	}

	return nil
}

// EncodeVarint encodes a varint
func (e *Encoder) EncodeVarint(value uint64) {
	for value >= 0x80 {
		e.buf = append(e.buf, byte(value)|0x80)
		value >>= 7
	}
	e.buf = append(e.buf, byte(value))
}

// EncodeFixed32 encodes a 32-bit fixed-width value
func (e *Encoder) EncodeFixed32(value uint32) {
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, value)
	e.buf = append(e.buf, bytes...)
}

// EncodeFixed64 encodes a 64-bit fixed-width value
func (e *Encoder) EncodeFixed64(value uint64) {
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, value)
	e.buf = append(e.buf, bytes...)
}

// EncodeBytes encodes a length-delimited byte array
func (e *Encoder) EncodeBytes(data []byte) {
	e.EncodeVarint(uint64(len(data)))
	e.buf = append(e.buf, data...)
}

// EncodeSignedVarint encodes a signed varint using zigzag encoding
func EncodeSignedVarint(value int64) uint64 {
	return uint64(value<<1) ^ uint64(value>>63)
}
