# protolite

**Schema-aware protobuf encoding/decoding without generated code**

Protolite allows you to work with protobuf messages dynamically using schema definitions, inspired by [thrift-iterator/go](https://github.com/thrift-iterator/go) but designed for Protocol Buffers.

## 🎯 **Key Features**

- ✅ **No code generation** - Work with protobuf messages dynamically
- ✅ **Schema-aware parsing** - Load .proto definitions and get field names
- ✅ **Field name access** - `result["user_name"]` instead of `result[1]`
- ✅ **Type safety** - Schema validation during encoding/decoding  
- ✅ **API Gateway friendly** - Perfect for proxies that need to modify protobuf messages
- ✅ **Debugging tools** - Decode any protobuf message with schema info

## 🚀 **Quick Start**

```go
import "github.com/protolite"

// Create protolite instance
p := protolite.New()

// Load schema (parsed from .proto files)
err := p.LoadRepo(protoRepo)

// Parse protobuf bytes - returns map[string]interface{} with field names!
result, err := p.Parse(protobufData, "User")
userName := result["name"]        // Field name access
userEmail := result["email"]      // No more field numbers!

// Marshal map back to protobuf
userData := map[string]interface{}{
    "name":  "John Doe", 
    "email": "john@example.com",
    "profile": map[string]interface{}{  // Nested messages supported!
        "bio": "Software engineer",
    },
}
protobufBytes, err := p.Marshal(userData, "User")

// Struct binding also works
type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}
var user User
err = p.Unmarshal(protobufBytes, &user)
```

## 🏗️ **Architecture**

Following the proven tzero pattern:

```
📁 schema/     - Protobuf schema type definitions
📁 registry/   - Schema management and symbol table  
📁 wire/       - Wire format types and constants
📄 api.go      - Main public API
```

## 🎯 **Use Cases**

### **API Gateways**
```go
// Modify protobuf messages without knowing schema at compile time
result, _ := p.Parse(request, "UserRequest") 
result["trace_id"] = generateTraceID()  // Add tracing
modifiedRequest, _ := p.Marshal(result, "UserRequest")
```

### **Debugging Tools**
```go
// Decode any protobuf message for inspection
messages := p.ListMessages()  // ["User", "Order", "Payment"]
result, _ := p.Parse(unknownData, "User")
// Get human-readable field names instead of numbers
```

### **Dynamic Systems**
```go
// Work with protobuf messages loaded at runtime
err := p.LoadSchema("dynamic_service.proto")
result, _ := p.Parse(messageBytes, "DynamicMessage")
```

## 🔄 **Development Status**

**Phase 1: Schema-Aware Architecture** ✅ (Complete)
- ✅ Schema type definitions
- ✅ Registry system  
- ✅ API structure
- ✅ Wire format parsing
- ⏳ .proto file parser (TODO)

**Phase 2: Core Implementation** ✅ (Complete)
- ✅ Schema-aware parsing and marshaling
- ✅ Wire format encoder/decoder
- ✅ Field validation and type checking
- ✅ Nested message support
- ✅ Enum support
- ✅ Reflection-based struct binding

**Phase 3: Advanced Features** ⏳ (Future)
- Performance optimizations  
- Proto2/Proto3 feature support
- Repeated fields support
- Map fields support
- .proto file parser

## 🧪 **Test Results**

The comprehensive test with nested User → Posts structure demonstrates:

✅ **Successful Marshal/Unmarshal** of complex nested data (199 bytes)  
✅ **Perfect Round-trip Fidelity** (original = round-trip)  
✅ **Field Name Resolution** (`result["name"]` vs `result[1]`)  
✅ **Type Preservation** (int32, int64, string, bool all correct)  
✅ **Nested Message Support** (User → UserProfile → Post)  
✅ **Hand-crafted Protobuf Parsing** (validates wire format compatibility)  
✅ **Error Handling** (graceful unknown message type handling)  

**Sample Output:**
```
✅ Loaded messages: [example.Post example.UserProfile example.User]
✅ Marshaled 199 bytes: 1a11616c696365406578616d706c652e636f6d22460a1d...
✅ Parsed result:
  User ID: 42 (type: int32)
  Name: Alice Johnson (type: string)  
  Profile:
    Bio: Software engineer and blogger (type: string)
  Latest Post:
    ID: 1001 (type: int64)
    Title: Getting Started with Protobuf (type: string)
    Published: true (type: bool)
```

Run the test: `cd cmd/test && go run main.go`
