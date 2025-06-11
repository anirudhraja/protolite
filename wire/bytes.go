package wire

import (
	"fmt"
)

// BytesDecoder handles length-delimited bytes decoding operations
type BytesDecoder struct {
	decoder *Decoder
}

// BytesEncoder handles length-delimited bytes encoding operations
type BytesEncoder struct {
	encoder *Encoder
}

// NewBytesDecoder creates a new bytes decoder
func NewBytesDecoder(d *Decoder) *BytesDecoder {
	return &BytesDecoder{decoder: d}
}

// NewBytesEncoder creates a new bytes encoder
func NewBytesEncoder(e *Encoder) *BytesEncoder {
	return &BytesEncoder{encoder: e}
}

// DECODER METHODS

// DecodeBytes decodes a length-delimited byte array
func (bd *BytesDecoder) DecodeBytes() ([]byte, error) {
	// First decode the length as a varint
	vd := NewVarintDecoder(bd.decoder)
	length, err := vd.DecodeVarint()
	if err != nil {
		return nil, fmt.Errorf("failed to decode bytes length: %v", err)
	}

	d := bd.decoder
	if d.pos+int(length) > len(d.buf) {
		return nil, fmt.Errorf("bytes truncated: need %d bytes, have %d", length, len(d.buf)-d.pos)
	}

	// Copy the data to avoid sharing the underlying buffer
	data := make([]byte, length)
	copy(data, d.buf[d.pos:d.pos+int(length)])
	d.pos += int(length)

	return data, nil
}

// DecodeString decodes a length-delimited string
func (bd *BytesDecoder) DecodeString() (string, error) {
	data, err := bd.DecodeBytes()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// DecodeRawBytes decodes bytes without copying (shares buffer)
func (bd *BytesDecoder) DecodeRawBytes() ([]byte, error) {
	// First decode the length as a varint
	vd := NewVarintDecoder(bd.decoder)
	length, err := vd.DecodeVarint()
	if err != nil {
		return nil, fmt.Errorf("failed to decode bytes length: %v", err)
	}

	d := bd.decoder
	if d.pos+int(length) > len(d.buf) {
		return nil, fmt.Errorf("bytes truncated: need %d bytes, have %d", length, len(d.buf)-d.pos)
	}

	// Return a slice that shares the underlying buffer
	data := d.buf[d.pos : d.pos+int(length)]
	d.pos += int(length)

	return data, nil
}

// SkipBytes skips over a length-delimited byte array
func (bd *BytesDecoder) SkipBytes() error {
	// Decode length and skip that many bytes
	vd := NewVarintDecoder(bd.decoder)
	length, err := vd.DecodeVarint()
	if err != nil {
		return err
	}

	d := bd.decoder
	if d.pos+int(length) > len(d.buf) {
		return fmt.Errorf("cannot skip %d bytes: only %d available", length, len(d.buf)-d.pos)
	}

	d.pos += int(length)
	return nil
}

// ENCODER METHODS

// EncodeBytes encodes a byte array as length-delimited
func (be *BytesEncoder) EncodeBytes(data []byte) error {
	// First encode the length as a varint
	ve := NewVarintEncoder(be.encoder)
	if err := ve.EncodeVarint(uint64(len(data))); err != nil {
		return fmt.Errorf("failed to encode bytes length: %v", err)
	}

	// Then append the data
	be.encoder.buf = append(be.encoder.buf, data...)
	return nil
}

// EncodeString encodes a string as length-delimited bytes
func (be *BytesEncoder) EncodeString(s string) error {
	return be.EncodeBytes([]byte(s))
}

// UTILITY FUNCTIONS

// BytesSize returns the size needed to encode the given bytes
func BytesSize(data []byte) int {
	return VarintSize(uint64(len(data))) + len(data)
}

// StringSize returns the size needed to encode the given string
func StringSize(s string) int {
	return BytesSize([]byte(s))
}

// Convenience methods for direct access (maintains backward compatibility)

// DecodeBytes - convenience method for main decoder
func (d *Decoder) DecodeBytes() ([]byte, error) {
	bd := NewBytesDecoder(d)
	return bd.DecodeBytes()
}

// EncodeBytes - convenience method for main encoder
func (e *Encoder) EncodeBytes(data []byte) error {
	be := NewBytesEncoder(e)
	return be.EncodeBytes(data)
}

// EncodeString - convenience method for main encoder
func (e *Encoder) EncodeString(s string) error {
	be := NewBytesEncoder(e)
	return be.EncodeString(s)
}
