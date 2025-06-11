# 🚀 Protolite

A powerful Go library for working with Protocol Buffers **without generated code**. Protolite provides schema-less parsing, schema-based marshaling/unmarshaling, and automatic Go struct mapping with reflection.

## ✨ Features

- 🔍 **Schema-less Parsing** - Inspect any protobuf data without knowing the schema
- 📋 **Schema-based Operations** - Marshal/unmarshal with `.proto` file schemas  
- 🏗️ **Automatic Struct Mapping** - Populate Go structs using reflection
- 🎯 **Wire Format Support** - All protobuf wire types (varint, fixed32, fixed64, bytes)
- 🔄 **Type Safety** - Proper Go type conversions with error handling
- 🌊 **Nested Messages** - Support for recursive and complex message structures
- 📊 **Maps & Enums** - Full support for protobuf maps and enumerations
- 🧠 **Smart Field Matching** - Handles CamelCase ↔ snake_case conversions

## 📦 Installation

```bash
go get github.com/protolite
```

## 🎯 Quick Start

```go
package main

import (
    "fmt"
    "github.com/protolite"
)

func main() {
    // Create Protolite instance
    proto := protolite.NewProtolite()
    
    // Schema-less parsing (no .proto file needed)
    result, err := proto.Parse(protobufData)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Unknown protobuf contains: %+v\n", result)
    
    // Schema-based operations (requires .proto file)
    err = proto.LoadSchemaFromFile("user.proto")
    if err != nil {
        panic(err)
    }
    
    // Unmarshal to map
    userMap, err := proto.UnmarshalWithSchema(protobufData, "User")
    fmt.Printf("User data: %+v\n", userMap)
    
    // Unmarshal to Go struct
    var user User
    err = proto.UnmarshalToStruct(protobufData, "User", &user)
    fmt.Printf("User struct: %+v\n", user)
}
```

## 🎯 Comprehensive Sample App

**Want to see ALL protobuf features in action?** Check out our comprehensive sample app that demonstrates every advanced protobuf feature:

```bash
cd sampleapp/
go run main.go
```

**🚀 Features Demonstrated:**
- ✅ **oneof fields** - Union types (contact_method, content types, notification_data)
- ✅ **Nested messages** - Deep nesting (User → Address → Coordinates)
- ✅ **Nested repeated** - Comments with recursive replies structure
- ✅ **Multiple map types** - string→string, string→int64, int32→string, string→Message
- ✅ **Comprehensive enums** - 12+ different enum types with proper scoping
- ✅ **Recursive structures** - Comments containing nested replies infinitely deep

**📁 Sample App Structure:**
```
sampleapp/
├── main.go                    # Comprehensive demo application
└── testdata/
    ├── user.proto            # Advanced User message with all features
    └── post.proto            # Complex Post message with oneof, maps, recursion
```

**📊 Sample Output:**
```
=== Comprehensive Protobuf Demo ===
✅ oneof fields (contact_method, content, notification_data, comment_type)
✅ Nested messages (Address -> Coordinates, deep nesting)  
✅ Nested repeated (notifications, comments with recursive replies)
✅ Multiple map types (string->string, string->int64, int32->string, string->Message)
✅ Comprehensive enums (12+ different enum types)
✅ Recursive structures (Reply -> nested_replies)

Marshaling comprehensive user data...
✅ Encoded data size: 419 bytes

✅ User: John Doe (ID: 1)
✅ Nested Address: San Francisco, CA
✅ Deeply nested Coordinates: 37.7749, -122.4194
✅ Posts: 1 items
✅ Comments in first post: 1 items
✅ Replies in first comment: 1 items
✅ Nested replies (recursive): 1 items

🎉 Comprehensive Protobuf demo completed successfully!
```

The sample app is the **perfect reference** for implementing complex protobuf schemas with Protolite!

## 📖 API Reference

### Core Interface

```go
type Protolite interface {
    // Schema-less parsing
    Parse(data []byte) (map[string]interface{}, error)
    
    // Schema-based operations  
    LoadSchemaFromFile(protoPath string) error
    MarshalWithSchema(data map[string]interface{}, messageName string) ([]byte, error)
    UnmarshalWithSchema(data []byte, messageName string) (map[string]interface{}, error)
    UnmarshalToStruct(data []byte, messageName string, v interface{}) error
}
```

---

## 🔍 Method Comparison

| Method | Schema Required | Output Type | Field Keys | Use Case |
|--------|----------------|-------------|------------|----------|
| **`Parse`** | ❌ No | `map[string]interface{}` | `field_1`, `field_2` | Debug unknown protobuf |
| **`UnmarshalWithSchema`** | ✅ Yes | `map[string]interface{}` | `id`, `name`, `email` | Dynamic processing |
| **`UnmarshalToStruct`** | ✅ Yes | Go struct | Struct fields | Type-safe application code |

---

## 📋 Detailed Usage

### 1. 🔍 Schema-less Parsing

**When to use:** Debug unknown protobuf data, inspect wire format, reverse engineering.

```go
proto := protolite.NewProtolite()

// Parse any protobuf data without schema
result, err := proto.Parse(unknownProtobufData)
if err != nil {
    log.Fatal(err)
}

// Output shows wire format structure
fmt.Printf("Parsed: %+v\n", result)
// Output: map[field_1:map[type:varint value:123] field_2:map[type:bytes value:[104 101 108 108 111]]]
```

**Output format:**
```go
map[string]interface{}{
    "field_1": map[string]interface{}{
        "type":  "varint",   // Wire type: varint, fixed32, fixed64, bytes
        "value": uint64(123), // Raw decoded value
    },
    "field_2": map[string]interface{}{
        "type":  "bytes",
        "value": []byte("hello"),
    },
}
```

### 2. 📋 Schema-based to Map

**When to use:** Dynamic processing, JSON conversion, generic data handling.

```go
proto := protolite.NewProtolite()

// Load schema first
err := proto.LoadSchemaFromFile("schemas/user.proto")
if err != nil {
    log.Fatal(err)
}

// Unmarshal with proper field names and types
userMap, err := proto.UnmarshalWithSchema(protobufData, "User")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("User: %+v\n", userMap)
// Output: map[id:123 name:John Doe email:john@example.com active:true]

// Convert to JSON easily
jsonData, _ := json.Marshal(userMap)
fmt.Printf("JSON: %s\n", jsonData)
```

### 3. 🏗️ Schema-based to Go Struct

**When to use:** Type-safe application code, direct struct usage, compile-time safety.

```go
type User struct {
    ID     int32  `json:"id"`
    Name   string `json:"name"`
    Email  string `json:"email"`
    Active bool   `json:"active"`
}

proto := protolite.NewProtolite()
err := proto.LoadSchemaFromFile("schemas/user.proto")
if err != nil {
    log.Fatal(err)
}

// Direct struct population with reflection
var user User
err = proto.UnmarshalToStruct(protobufData, "User", &user)
if err != nil {
    log.Fatal(err)
}

// Use struct fields directly with type safety
fmt.Printf("User ID: %d, Name: %s, Active: %t\n", user.ID, user.Name, user.Active)
```

**Smart Field Matching:**
- `ID` → matches `id` or `ID`
- `UserName` → matches `user_name`, `username`, `UserName`
- `EmailAddress` → matches `email_address`, `EmailAddress`

### 4. 📤 Schema-based Marshaling

```go
proto := protolite.NewProtolite()
err := proto.LoadSchemaFromFile("schemas/user.proto")

// Create data to marshal
userData := map[string]interface{}{
    "id":     int32(456),
    "name":   "Jane Smith",
    "email":  "jane@example.com", 
    "active": true,
}

// Marshal to protobuf bytes
protobufData, err := proto.MarshalWithSchema(userData, "User")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Encoded %d bytes\n", len(protobufData))
```

---

## 🧪 Supported Types

### Primitive Types
- ✅ `int32`, `int64`, `uint32`, `uint64`
- ✅ `bool`, `string`, `bytes`
- ✅ `float`, `double`
- ✅ `enum` values

### Complex Types  
- ✅ **Nested Messages** - Recursive message structures
- ✅ **Maps** - `map<string, int32>`, `map<string, string>`, etc.
- ✅ **Enums** - Named constants with validation
- ✅ **Repeated Fields** - Arrays and lists

### Wire Format Support
- ✅ **Varint** - Variable-length integers
- ✅ **Fixed32** - 4-byte fixed-width (float, fixed32, sfixed32)  
- ✅ **Fixed64** - 8-byte fixed-width (double, fixed64, sfixed64)
- ✅ **Bytes** - Length-delimited (string, bytes, messages)

---

## 📁 Example Schema

```protobuf
// user.proto
syntax = "proto3";

message User {
    int32 id = 1;
    string name = 2;
    string email = 3;
    bool active = 4;
    Status status = 5;
    map<string, string> metadata = 6;
    Address address = 7;
}

enum Status {
    UNKNOWN = 0;
    ACTIVE = 1;
    INACTIVE = 2;
}

message Address {
    string street = 1;
    string city = 2;
    int32 zip_code = 3;
}
```

---

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Protolite     │    │   Wire Format    │    │   Schema        │
│   Interface     │────│   Decoders       │────│   Registry      │  
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         │              ┌─────────────────┐              │
         └──────────────│   Reflection    │──────────────┘
                        │   Engine        │
                        └─────────────────┘
```

### Components

1. **🎯 Protolite Interface** - High-level API for users
2. **🔧 Wire Format Decoders** - Low-level protobuf parsing (varint, fixed, bytes)  
3. **📚 Schema Registry** - `.proto` file loading and message definitions
4. **🪞 Reflection Engine** - Go struct mapping with type conversion

---

## ⚡ Performance & Limitations

### ✅ Strengths
- **Zero code generation** - No `protoc` compilation needed
- **Dynamic schemas** - Load `.proto` files at runtime  
- **Type safety** - Proper Go type conversions
- **Flexible parsing** - Works with unknown protobuf data

### ⚠️ Limitations
- **Runtime schema loading** - Slightly slower than generated code
- **No proto2 extensions** - Focus on proto3 features
- **Reflection overhead** - Struct mapping uses reflection

---

## 🧪 Testing

Run the comprehensive test suite:

```bash
# Run all tests
go test -v ./...

# Run specific component tests
go test -v ./wire      # Wire format tests
go test -v ./registry  # Schema registry tests  
go test -v .           # API tests

# Run with coverage
go test -cover ./...
```

### Test Coverage
- ✅ **All primitive types** - Complete wire format coverage
- ✅ **Nested messages** - Recursive structures  
- ✅ **Maps and enums** - Complex type support
- ✅ **Edge cases** - Empty messages, zero values, extreme values
- ✅ **Error handling** - Invalid data, missing schemas

---

## 🚀 Examples

Check out the comprehensive **sample app** for advanced usage examples:

```bash
cd sampleapp/
go run main.go
```

The sample app demonstrates all protobuf features including oneof, nested messages, maps, enums, and recursive structures!

---

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Ensure all tests pass (`go test -v ./...`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---