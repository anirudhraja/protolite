package schema

// ProtoRepo represents a collection of .proto files and their definitions.
type ProtoRepo struct {
	ProtoFiles map[string]*ProtoFile `json:"proto_files"`
}

// ProtoFile represents a single .proto file
type ProtoFile struct {
	Name     string     `json:"name"`     // file.proto
	Package  string     `json:"package"`  // package name
	Syntax   string     `json:"syntax"`   // proto2 or proto3
	Imports  []*Import  `json:"imports"`  // imported files
	Messages []*Message `json:"messages"` // message definitions
	Enums    []*Enum    `json:"enums"`    // enum definitions
	Services []*Service `json:"services"` // service definitions
}

// Import represents an import statement
type Import struct {
	Path   string `json:"path"`   // "google/protobuf/timestamp.proto"
	Public bool   `json:"public"` // public import
	Weak   bool   `json:"weak"`   // weak import
}

// Message represents a protobuf message definition
type Message struct {
	Name        string     `json:"name"`         // "User"
	Fields      []*Field   `json:"fields"`       // message fields
	NestedTypes []*Message `json:"nested_types"` // nested messages
	NestedEnums []*Enum    `json:"nested_enums"` // nested enums
	Extensions  []*Field   `json:"extensions"`   // extension fields
	OneofGroups []*Oneof   `json:"oneof_groups"` // oneof groups
	MapEntry    bool       `json:"map_entry"`    // is this a map entry?
	IsWrapper   bool       `json:"is_wrapper"`   // is this a wrapper?
	ShowNull    bool       `json:"show_null"`    // should show null in decode
}

// Field represents a message field
type Field struct {
	Name            string     `json:"name"`          // "user_name"
	Number          int32      `json:"number"`        // 1
	Label           FieldLabel `json:"label"`         // optional, required, repeated
	Type            FieldType  `json:"type"`          // field type information
	DefaultValue    string     `json:"default_value"` // default value (proto2)
	JsonName        string     `json:"json_name"`     // JSON field name
	OneofIndex      int32      `json:"oneof_index"`   // oneof group index (-1 if not in oneof)
	JSONString      bool       `json:"json_string"`   // when set raw json string is used to transport gql scalars on wire.
	WrapperFieldKey string     `json:"wrapper_field"` // when set, indicates the field is a wrapper around another field. e.g., `google.protobuf.StringValue` wraps around `string`
}

// Oneof represents a oneof group
type Oneof struct {
	Name   string   `json:"name"`   // "user_info"
	Fields []*Field `json:"fields"` // fields in this oneof
}

// FieldLabel represents field labels
type FieldLabel string

const (
	LabelOptional FieldLabel = "optional"
	LabelRequired FieldLabel = "required"
	LabelRepeated FieldLabel = "repeated"
)

// FieldType represents field type information
type FieldType struct {
	Kind          TypeKind      `json:"kind"`                     // primitive, message, enum, map, wrapper
	PrimitiveType PrimitiveType `json:"primitive_type,omitempty"` // for primitive types
	MessageType   string        `json:"message_type,omitempty"`   // for message types: "User", "google.protobuf.Timestamp"
	EnumType      string        `json:"enum_type,omitempty"`      // for enum types
	WrapperType   WrapperType   `json:"wrapper_type,omitempty"`   // for wrapper types
	MapKey        *FieldType    `json:"map_key,omitempty"`        // for map key type
	MapValue      *FieldType    `json:"map_value,omitempty"`      // for map value type
	ElementType   *FieldType    `json:"element_type,omitempty"`   // for repeated element type
}

// TypeKind represents the kind of field type
type TypeKind string

const (
	KindPrimitive TypeKind = "primitive"
	KindMessage   TypeKind = "message"
	KindEnum      TypeKind = "enum"
	KindMap       TypeKind = "map"
	KindWrapper   TypeKind = "wrapper"
)

// PrimitiveType represents protobuf primitive types
type PrimitiveType string

const (
	TypeDouble   PrimitiveType = "double"
	TypeFloat    PrimitiveType = "float"
	TypeInt64    PrimitiveType = "int64"
	TypeUint64   PrimitiveType = "uint64"
	TypeInt32    PrimitiveType = "int32"
	TypeFixed64  PrimitiveType = "fixed64"
	TypeFixed32  PrimitiveType = "fixed32"
	TypeBool     PrimitiveType = "bool"
	TypeString   PrimitiveType = "string"
	TypeBytes    PrimitiveType = "bytes"
	TypeUint32   PrimitiveType = "uint32"
	TypeSfixed32 PrimitiveType = "sfixed32"
	TypeSfixed64 PrimitiveType = "sfixed64"
	TypeSint32   PrimitiveType = "sint32"
	TypeSint64   PrimitiveType = "sint64"
)

var packedEligible = map[PrimitiveType]struct{}{
	TypeDouble:   {},
	TypeFloat:    {},
	TypeInt64:    {},
	TypeUint64:   {},
	TypeInt32:    {},
	TypeFixed64:  {},
	TypeFixed32:  {},
	TypeBool:     {},
	TypeUint32:   {},
	TypeSfixed32: {},
	TypeSfixed64: {},
	TypeSint32:   {},
	TypeSint64:   {},
}

// IsPackedType checks and returns if the Primitive type is packed for repeated label
func IsPackedType(t PrimitiveType) bool {
	_, ok := packedEligible[t]
	return ok
}

// WrapperType represents protobuf wrapper types
type WrapperType string

const (
	WrapperDoubleValue WrapperType = "google.protobuf.DoubleValue"
	WrapperFloatValue  WrapperType = "google.protobuf.FloatValue"
	WrapperInt64Value  WrapperType = "google.protobuf.Int64Value"
	WrapperUInt64Value WrapperType = "google.protobuf.UInt64Value"
	WrapperInt32Value  WrapperType = "google.protobuf.Int32Value"
	WrapperUInt32Value WrapperType = "google.protobuf.UInt32Value"
	WrapperBoolValue   WrapperType = "google.protobuf.BoolValue"
	WrapperStringValue WrapperType = "google.protobuf.StringValue"
	WrapperBytesValue  WrapperType = "google.protobuf.BytesValue"
)

// Enum represents an enum definition
type Enum struct {
	Name       string       `json:"name"`        // "Status"
	Values     []*EnumValue `json:"values"`      // enum values
	AllowAlias bool         `json:"allow_alias"` // allow_alias option
}

// EnumValue represents an enum value
type EnumValue struct {
	Name     string `json:"name"`      // "ACTIVE"
	Number   int32  `json:"number"`    // 1
	JsonName string `json:"json_name"` // JSON field name
}

// Service represents a service definition
type Service struct {
	Name    string    `json:"name"`    // "UserService"
	Methods []*Method `json:"methods"` // service methods
}

// Method represents a service method
type Method struct {
	Name            string `json:"name"`             // "GetUser"
	InputType       string `json:"input_type"`       // "GetUserRequest"
	OutputType      string `json:"output_type"`      // "GetUserResponse"
	ClientStreaming bool   `json:"client_streaming"` // stream input
	ServerStreaming bool   `json:"server_streaming"` // stream output
}
