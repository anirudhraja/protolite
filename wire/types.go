package wire

// ===== PROTOBUF WIRE FORMAT TYPES =====

// WireType represents protobuf wire format types
type WireType int32

const (
	WireVarint  WireType = 0 // int32, int64, uint32, uint64, sint32, sint64, bool, enum
	WireFixed64 WireType = 1 // fixed64, sfixed64, double
	WireBytes   WireType = 2 // string, bytes, embedded messages, packed repeated fields
	WireFixed32 WireType = 5 // fixed32, sfixed32, float
)

// FieldNumber represents a protobuf field number
type FieldNumber int32

// Tag represents a protobuf field tag (field number + wire type)
type Tag uint64

// MakeTag creates a tag from field number and wire type
func MakeTag(fieldNumber FieldNumber, wireType WireType) Tag {
	return Tag(uint64(fieldNumber)<<3 | uint64(wireType))
}

// ParseTag parses a tag into field number and wire type
func ParseTag(tag Tag) (FieldNumber, WireType) {
	return FieldNumber(tag >> 3), WireType(tag & 0x7)
}

// MessageHeader represents the header of a protobuf message field
type MessageHeader struct {
	FieldNumber FieldNumber
	WireType    WireType
	Length      uint64 // For length-delimited fields
}

// Value represents a decoded protobuf value
type Value struct {
	FieldNumber FieldNumber
	WireType    WireType
	Data        interface{} // Actual value
}

// RawValue represents a raw (undecoded) protobuf value
type RawValue struct {
	FieldNumber FieldNumber
	WireType    WireType
	RawData     []byte // Raw bytes
}
