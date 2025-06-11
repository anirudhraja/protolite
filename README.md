# 🚀 Protolite

A powerful Go library for working with Protocol Buffers **without generated code**. Protolite provides schema-less parsing, schema-based marshaling/unmarshaling, and automatic Go struct mapping with reflection.

## ✨ Features

- 🔍 **Schema-less Parsing** - Inspect any protobuf data without knowing the schema
- 📋 **Schema-based Operations** - Marshal/unmarshal with `.proto` file schemas  
- 🏗️ **Automatic Struct Mapping** - Populate Go structs using reflection
- 🎯 **Wire Format Support** - All protobuf wire types (varint, fixed32, fixed64, bytes)
- 🔄 **Type Safety** - Proper Go type conversions with error handling
- 🌊 **Nested Messages** - Support for recursive and complex message structures

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
- ✅ **Oneof Fields** - Union types for mutually exclusive fields

### Wire Format Support
- ✅ **Varint** - Variable-length integers
- ✅ **Fixed32** - 4-byte fixed-width (float, fixed32, sfixed32)  
- ✅ **Fixed64** - 8-byte fixed-width (double, fixed64, sfixed64)
- ✅ **Bytes** - Length-delimited (string, bytes, messages)

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

## ⚡ Performance Benchmarks

### 🏆 **Comprehensive Performance Comparison**

We benchmarked Protolite against **protoc-generated code** and **Google's DynamicPB** using real protobuf payloads with 1 million iterations for maximum precision.

#### **Simple Payload (32 bytes - basic fields)**
| Library | Time/Operation | Memory/Op | Allocations/Op |
|---------|----------------|-----------|----------------|
| **Protoc Generated** | **273 ns** | 232 B | 4 allocs |
| **DynamicPB** | 589 ns | 576 B | 11 allocs |
| **Protolite** | 1,072 ns | 440 B | 10 allocs |

#### **Complex Payload (695 bytes - nested maps, arrays, messages)**  
| Library | Time/Operation | Memory/Op | Allocations/Op |
|---------|----------------|-----------|----------------|
| **Protolite** | **1,089 ns** | 440 B | 10 allocs |
| **DynamicPB** | 1,852 ns | 2,784 B | 16 allocs |
| **Protoc Generated** | 4,902 ns | 3,536 B | 102 allocs |


### 🧪 **Run Benchmarks Yourself**

```bash
cd benchmark/
go test -bench=. -benchmem -benchtime=100000x
```

**Benchmark Environment:**  
- Apple M2 Pro (12-core ARM64)
- Go 1.21.x

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

## ❌ **What's Not Supported** (Proto3 Focus)

### **Protocol Buffer Features**
- ❌ **Services/RPC** - No gRPC service definitions or method calls
- ❌ **Custom Options** - Proto file custom options not parsed
- ❌ **Import Public** - `import public` statements not handled

### **Well-Known Types**
- ❌ **google.protobuf.Any** - Type erasure/dynamic types
- ❌ **google.protobuf.Timestamp** - No automatic time conversion
- ❌ **google.protobuf.Duration** - No automatic duration parsing
- ❌ **google.protobuf.Wrapper** - Value wrapper types (StringValue, Int32Value, etc.)
- ❌ **google.protobuf.FieldMask** - Field selection masks
- ❌ **google.protobuf.Struct** - Dynamic JSON-like structures
- ❌ **google.protobuf.Empty** - Empty message type

### **Advanced Wire Format**
- ❌ **Packed Repeated Optimization** - Packed encoding for repeated primitives
- ❌ **Unknown Field Preservation** - Unknown fields are skipped, not preserved

### **Schema Features**
- ❌ **Nested Type Definitions** - Messages/enums defined inside other messages
- ❌ **Reserved Fields** - Reserved field numbers and names not validated
- ❌ **Deprecated Fields** - No deprecation warnings or handling
- ❌ **Multi-file Type Resolution** - Complex cross-file imports may not resolve
- ❌ **JSON Mapping Options** - Custom JSON field names beyond basic conversion

### **Performance Optimizations**
- ❌ **Zero-Copy Parsing** - All data is copied during parsing
- ❌ **Lazy Loading** - No lazy field evaluation
- ❌ **Memory Pooling** - No object reuse or memory pools
- ❌ **Streaming Parser** - Must load entire message into memory

---

## ⚠️ **Limitations**

### **Performance Trade-offs**
- **Runtime schema loading** - Slightly slower than generated code for simple data
- **Reflection overhead** - Struct mapping uses reflection for flexibility

### **Protocol Buffer Support**  
- **Focus on Proto3** - Full proto3 support, limited proto2 features
- **Simple .proto parsing** - Basic proto file parsing, not full protoc compatibility

---

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

---