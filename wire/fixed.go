package wire

import (
	"encoding/binary"
	"fmt"
	"math"
)

// FixedDecoder handles fixed-width decoding operations
type FixedDecoder struct {
	decoder *Decoder
}

// FixedEncoder handles fixed-width encoding operations
type FixedEncoder struct {
	encoder *Encoder
}

// NewFixedDecoder creates a new fixed decoder
func NewFixedDecoder(d *Decoder) *FixedDecoder {
	return &FixedDecoder{decoder: d}
}

// NewFixedEncoder creates a new fixed encoder
func NewFixedEncoder(e *Encoder) *FixedEncoder {
	return &FixedEncoder{encoder: e}
}

// DECODER METHODS

// DecodeFixed32 decodes a 32-bit fixed-width value
func (fd *FixedDecoder) DecodeFixed32() (uint32, error) {
	d := fd.decoder
	if d.pos+4 > len(d.buf) {
		return 0, fmt.Errorf("not enough data for fixed32")
	}

	value := binary.LittleEndian.Uint32(d.buf[d.pos:])
	d.pos += 4
	return value, nil
}

// DecodeFixed64 decodes a 64-bit fixed-width value
func (fd *FixedDecoder) DecodeFixed64() (uint64, error) {
	d := fd.decoder
	if d.pos+8 > len(d.buf) {
		return 0, fmt.Errorf("not enough data for fixed64")
	}

	value := binary.LittleEndian.Uint64(d.buf[d.pos:])
	d.pos += 8
	return value, nil
}

// DecodeSfixed32 decodes a signed 32-bit fixed-width value
func (fd *FixedDecoder) DecodeSfixed32() (int32, error) {
	v, err := fd.DecodeFixed32()
	if err != nil {
		return 0, err
	}
	return int32(v), nil
}

// DecodeSfixed64 decodes a signed 64-bit fixed-width value
func (fd *FixedDecoder) DecodeSfixed64() (int64, error) {
	v, err := fd.DecodeFixed64()
	if err != nil {
		return 0, err
	}
	return int64(v), nil
}

// DecodeFloat32 decodes a 32-bit float from fixed32 data
func (fd *FixedDecoder) DecodeFloat32() (float32, error) {
	v, err := fd.DecodeFixed32()
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(v), nil
}

// DecodeFloat64 decodes a 64-bit float from fixed64 data
func (fd *FixedDecoder) DecodeFloat64() (float64, error) {
	v, err := fd.DecodeFixed64()
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(v), nil
}

// ENCODER METHODS

// EncodeFixed32 encodes a 32-bit fixed-width value
func (fe *FixedEncoder) EncodeFixed32(v uint32) error {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, v)
	fe.encoder.buf = append(fe.encoder.buf, buf...)
	return nil
}

// EncodeFixed64 encodes a 64-bit fixed-width value
func (fe *FixedEncoder) EncodeFixed64(v uint64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, v)
	fe.encoder.buf = append(fe.encoder.buf, buf...)
	return nil
}

// EncodeSfixed32 encodes a signed 32-bit fixed-width value
func (fe *FixedEncoder) EncodeSfixed32(v int32) error {
	return fe.EncodeFixed32(uint32(v))
}

// EncodeSfixed64 encodes a signed 64-bit fixed-width value
func (fe *FixedEncoder) EncodeSfixed64(v int64) error {
	return fe.EncodeFixed64(uint64(v))
}

// EncodeFloat32 encodes a 32-bit float as fixed32
func (fe *FixedEncoder) EncodeFloat32(v float32) error {
	return fe.EncodeFixed32(math.Float32bits(v))
}

// EncodeFloat64 encodes a 64-bit float as fixed64
func (fe *FixedEncoder) EncodeFloat64(v float64) error {
	return fe.EncodeFixed64(math.Float64bits(v))
}

// UTILITY FUNCTIONS

// Fixed32Size returns the size of a fixed32 value (always 4 bytes)
func Fixed32Size() int {
	return 4
}

// Fixed64Size returns the size of a fixed64 value (always 8 bytes)
func Fixed64Size() int {
	return 8
}

// Convenience methods for direct access (maintains backward compatibility)

// DecodeFixed32 - convenience method for main decoder
func (d *Decoder) DecodeFixed32() (uint32, error) {
	fd := NewFixedDecoder(d)
	return fd.DecodeFixed32()
}

// DecodeFixed64 - convenience method for main decoder
func (d *Decoder) DecodeFixed64() (uint64, error) {
	fd := NewFixedDecoder(d)
	return fd.DecodeFixed64()
}

// EncodeFixed32 - convenience method for main encoder
func (e *Encoder) EncodeFixed32(v uint32) error {
	fe := NewFixedEncoder(e)
	return fe.EncodeFixed32(v)
}

// EncodeFixed64 - convenience method for main encoder
func (e *Encoder) EncodeFixed64(v uint64) error {
	fe := NewFixedEncoder(e)
	return fe.EncodeFixed64(v)
}
