package wire

import (
	"github.com/protolite/registry"
	"github.com/protolite/schema"
)

// Encoder handles low-level protobuf wire format encoding
type Encoder struct {
	buf      []byte
	registry *registry.Registry
}

// NewEncoder creates a new wire format encoder
func NewEncoder() *Encoder {
	return &Encoder{
		buf: make([]byte, 0),
	}
}

// NewEncoderWithRegistry creates an encoder with schema registry
func NewEncoderWithRegistry(registry *registry.Registry) *Encoder {
	return &Encoder{
		buf:      make([]byte, 0),
		registry: registry,
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

// EncodeMessage encodes a message using schema - main entry point
func EncodeMessage(data map[string]interface{}, msg *schema.Message, registry *registry.Registry) ([]byte, error) {
	encoder := NewEncoderWithRegistry(registry)
	me := NewMessageEncoder(encoder)
	err := me.EncodeMessage(data, msg)
	if err != nil {
		return nil, err
	}
	return encoder.Bytes(), nil
}
