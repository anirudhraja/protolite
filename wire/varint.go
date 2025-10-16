package wire

import (
	"errors"
)

// Varint encoding/decoding errors
var (
	ErrVarintOverflow = errors.New("varint overflow")
	ErrVarintTooLong  = errors.New("varint too long")
	ErrUnexpectedEOF  = errors.New("unexpected EOF while reading varint")
)

// VarintDecoder handles varint decoding operations
type VarintDecoder struct {
	decoder *Decoder
}

// VarintEncoder handles varint encoding operations
type VarintEncoder struct {
	encoder *Encoder
}

// NewVarintDecoder creates a new varint decoder
func NewVarintDecoder(d *Decoder) *VarintDecoder {
	return &VarintDecoder{decoder: d}
}

// NewVarintEncoder creates a new varint encoder
func NewVarintEncoder(e *Encoder) *VarintEncoder {
	return &VarintEncoder{encoder: e}
}

// DECODER METHODS

// DecodeVarint decodes a varint from the current position
func (vd *VarintDecoder) DecodeVarint() (uint64, error) {
	d := vd.decoder
	if d.pos >= len(d.buf) {
		return 0, ErrUnexpectedEOF
	}

	var result uint64
	var shift uint

	for i := 0; i < 10; i++ { // Max 10 bytes for 64-bit varint
		if d.pos >= len(d.buf) {
			return 0, ErrUnexpectedEOF
		}

		b := d.buf[d.pos]
		d.pos++

		// Check for overflow before shifting
		if shift >= 64 {
			return 0, ErrVarintOverflow
		}

		// Add the lower 7 bits to result
		result |= uint64(b&0x7F) << shift

		// If MSB is not set, we're done
		if (b & 0x80) == 0 {
			return result, nil
		}

		shift += 7
	}

	return 0, ErrVarintTooLong
}

// DecodeInt32 decodes a varint as int32
func (vd *VarintDecoder) DecodeInt32() (int32, error) {
	v, err := vd.DecodeVarint()
	if err != nil {
		return 0, err
	}
	return int32(v), nil
}

// DecodeInt64 decodes a varint as int64
func (vd *VarintDecoder) DecodeInt64() (int64, error) {
	v, err := vd.DecodeVarint()
	if err != nil {
		return 0, err
	}
	return int64(v), nil
}

// DecodeSint32 decodes a zigzag-encoded signed varint as int32
func (vd *VarintDecoder) DecodeSint32() (int32, error) {
	v, err := vd.DecodeVarint()
	if err != nil {
		return 0, err
	}
	return DecodeZigZag32(v), nil
}

// DecodeSint64 decodes a zigzag-encoded signed varint as int64
func (vd *VarintDecoder) DecodeSint64() (int64, error) {
	v, err := vd.DecodeVarint()
	if err != nil {
		return 0, err
	}
	return DecodeZigZag64(v), nil
}

// DecodeBool decodes a varint as bool
func (vd *VarintDecoder) DecodeBool() (bool, error) {
	v, err := vd.DecodeVarint()
	if err != nil {
		return false, err
	}
	return v != 0, nil
}

// DecodeEnum decodes a varint as enum value
func (vd *VarintDecoder) DecodeEnum() (int32, error) {
	v, err := vd.DecodeVarint()
	if err != nil {
		return 0, err
	}
	return int32(v), nil
}

// SkipVarint skips over a varint without decoding it
func (vd *VarintDecoder) SkipVarint() error {
	d := vd.decoder
	for {
		if d.pos >= len(d.buf) {
			return ErrUnexpectedEOF
		}

		b := d.buf[d.pos]
		d.pos++

		if (b & 0x80) == 0 {
			return nil
		}
	}
}

// ENCODER METHODS

// EncodeVarint encodes a uint64 as varint
func (ve *VarintEncoder) EncodeVarint(v uint64) {
	for v >= 0x80 {
		ve.encoder.buf = append(ve.encoder.buf, byte(v)|0x80)
		v >>= 7
	}
	ve.encoder.buf = append(ve.encoder.buf, byte(v))
}

// EncodeInt32 encodes an int32 as varint
func (ve *VarintEncoder) EncodeInt32(v int32) {
	ve.EncodeVarint(uint64(v))
}

// EncodeInt64 encodes an int64 as varint
func (ve *VarintEncoder) EncodeInt64(v int64) {
	ve.EncodeVarint(uint64(v))
}

// EncodeUint32 encodes a uint32 as varint
func (ve *VarintEncoder) EncodeUint32(v uint32) {
	ve.EncodeVarint(uint64(v))
}

// EncodeUint64 encodes a uint64 as varint
func (ve *VarintEncoder) EncodeUint64(v uint64) {
	ve.EncodeVarint(v)
}

// EncodeSint32 encodes a signed int32 with zigzag encoding
func (ve *VarintEncoder) EncodeSint32(v int32) {
	ve.EncodeVarint(EncodeZigZag32(v))
}

// EncodeSint64 encodes a signed int64 with zigzag encoding
func (ve *VarintEncoder) EncodeSint64(v int64) {
	ve.EncodeVarint(EncodeZigZag64(v))
}

// EncodeBool encodes a bool as varint
func (ve *VarintEncoder) EncodeBool(v bool) {
	if v {
		ve.EncodeVarint(1)
	} else {
		ve.EncodeVarint(0)
	}
}

// EncodeEnum encodes an enum value as varint
func (ve *VarintEncoder) EncodeEnum(v int32) {
	ve.EncodeVarint(uint64(v))
}

// UTILITY FUNCTIONS

// DecodeZigZag32 decodes a zigzag-encoded 32-bit integer
func DecodeZigZag32(encoded uint64) int32 {
	return int32((uint32(encoded) >> 1) ^ uint32(-int32(encoded&1)))
}

// DecodeZigZag64 decodes a zigzag-encoded 64-bit integer
func DecodeZigZag64(encoded uint64) int64 {
	return int64((encoded >> 1) ^ uint64(-int64(encoded&1)))
}

// EncodeZigZag32 encodes a signed 32-bit integer using zigzag encoding
func EncodeZigZag32(v int32) uint64 {
	return uint64((uint32(v) << 1) ^ uint32(v>>31))
}

// EncodeZigZag64 encodes a signed 64-bit integer using zigzag encoding
func EncodeZigZag64(v int64) uint64 {
	return uint64((v << 1) ^ (v >> 63))
}

// VarintSize returns the number of bytes needed to encode the given varint
func VarintSize(v uint64) int {
	switch {
	case v < 1<<7:
		return 1
	case v < 1<<14:
		return 2
	case v < 1<<21:
		return 3
	case v < 1<<28:
		return 4
	case v < 1<<35:
		return 5
	case v < 1<<42:
		return 6
	case v < 1<<49:
		return 7
	case v < 1<<56:
		return 8
	case v < 1<<63:
		return 9
	default:
		return 10
	}
}

// Convenience methods for direct access (maintains backward compatibility)

// DecodeVarint - convenience method for main decoder
func (d *Decoder) DecodeVarint() (uint64, error) {
	vd := NewVarintDecoder(d)
	return vd.DecodeVarint()
}

// EncodeVarint - convenience method for main encoder
func (e *Encoder) EncodeVarint(v uint64) {
	ve := NewVarintEncoder(e)
	ve.EncodeVarint(v)
}
