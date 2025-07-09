package benchmark

import (
	"testing"

	"context"

	"github.com/bufbuild/protocompile"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/anirudhraja/protolite"
	pb "github.com/anirudhraja/protolite/benchmark/generated"
)

// Global test data and clients
var (
	// Simple payload (basic fields only)
	simplePayload    []byte
	simpleDescriptor protoreflect.MessageDescriptor

	// Complex payload (nested, maps, repeated fields)
	complexPayload    []byte
	complexDescriptor protoreflect.MessageDescriptor

	// Protolite client
	protoliteClient protolite.Protolite

	runtimeSimpleDescriptor  protoreflect.MessageDescriptor
	runtimeComplexDescriptor protoreflect.MessageDescriptor
)

func init() {
	setupBenchmarkData()
	loadRuntimeDescriptors()
}

func setupBenchmarkData() {
	var err error

	// Setup Protolite client
	protoliteClient = protolite.NewProtolite()
	err = protoliteClient.LoadSchemaFromFile("proto/post.proto")
	if err != nil {
		panic("Failed to load post schema: " + err.Error())
	}

	err = protoliteClient.LoadSchemaFromFile("proto/user.proto")
	if err != nil {
		panic("Failed to load schema: " + err.Error())
	}


	// Create simple payload (basic fields only)
	simpleUser := &pb.User{
		Id:            123,
		Name:          "John Doe",
		Active:        true,
		ContactMethod: &pb.User_Email{Email: "john@example.com"},
	}
	simplePayload, err = proto.Marshal(simpleUser)
	if err != nil {
		panic("Failed to create simple payload: " + err.Error())
	}

	// Create complex payload (full featured)
	complexUser := createComplexUser()
	complexPayload, err = proto.Marshal(complexUser)
	if err != nil {
		panic("Failed to create complex payload: " + err.Error())
	}

	// Setup DynamicPB descriptors
	setupDynamicDescriptors()
}

func createComplexUser() *pb.User {
	return &pb.User{
		Id:            1,
		Name:          "John Doe",
		Active:        true,
		Status:        pb.UserStatus_USER_ACTIVE,
		UserType:      pb.UserType_USER_TYPE_PREMIUM,
		ContactMethod: &pb.User_Email{Email: "john@example.com"},
		Address: &pb.Address{
			Street:     "123 Main St",
			City:       "San Francisco",
			State:      "CA",
			Country:    "USA",
			PostalCode: "94105",
			Type:       pb.AddressType_ADDRESS_HOME,
			Coordinates: &pb.Coordinates{
				Latitude:  37.7749,
				Longitude: -122.4194,
				System:    pb.CoordinateSystem_COORD_WGS84,
			},
		},
		Metadata: map[string]string{
			"theme":    "dark",
			"language": "en",
			"timezone": "UTC-8",
			"plan":     "premium",
		},
		Statistics: map[string]int64{
			"login_count":   1042,
			"posts_created": 234,
			"comments_made": 567,
		},
		Preferences: map[int32]string{
			1: "email_notifications",
			2: "push_notifications",
			3: "weekly_digest",
		},
		Profiles: map[string]*pb.UserProfile{
			"main": {
				DisplayName: "John D.",
				Bio:         "Software engineer passionate about performance",
				AvatarUrl:   "https://example.com/avatar.jpg",
				Visibility:  pb.ProfileVisibility_PROFILE_PUBLIC,
				Interests:   []string{"golang", "protobuf", "performance"},
			},
		},
		Notifications: []*pb.Notification{
			{
				Id:        1,
				Title:     "Welcome!",
				Message:   "Welcome to our platform",
				Type:      pb.NotificationType_NOTIF_SYSTEM,
				Timestamp: 1640995200,
				Read:      false,
				NotificationData: &pb.Notification_SystemData{
					SystemData: &pb.SystemNotificationData{
						SystemMessage: "Account created successfully",
						ActionUrl:     "/dashboard",
						Priority:      pb.SystemPriority_PRIORITY_HIGH,
					},
				},
			},
		},
		Posts: []*pb.Post{
			{
				Id:     101,
				Title:  "Comprehensive Protobuf Guide",
				Status: pb.PostStatus_POST_PUBLISHED,
				Content: &pb.Post_TextContent{
					TextContent: &pb.TextContent{
						Body:   "This is a comprehensive guide to protobuf performance optimization...",
						Format: pb.TextFormat_TEXT_MARKDOWN,
					},
				},
				AuthorId:  1,
				Tags:      []string{"protobuf", "performance", "golang"},
				CreatedAt: 1640995200,
				UpdatedAt: 1640995200,
				ViewCount: 1250,
				Featured:  true,
			},
		},
		CreatedAt: 1640995200,
	}
}

func setupDynamicDescriptors() {
	// Simple descriptor
	simpleFileDesc := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("user.proto"),
		Package: proto.String("benchmark"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("User"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: proto.String("id"), Number: proto.Int32(1), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum()},
					{Name: proto.String("name"), Number: proto.Int32(2), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
					{Name: proto.String("active"), Number: proto.Int32(4), Type: descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum()},
				},
			},
		},
	}

	files, err := protodesc.NewFiles(&descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{simpleFileDesc},
	})
	if err != nil {
		panic("Failed to create simple file descriptor: " + err.Error())
	}

	fileDescriptor, err := files.FindFileByPath("user.proto")
	if err != nil {
		panic("Failed to find simple file descriptor: " + err.Error())
	}

	simpleDescriptor = fileDescriptor.Messages().ByName("User")
	complexDescriptor = simpleDescriptor // Use same descriptor for complex (simplified for DynamicPB)
}

func loadRuntimeDescriptors() {
	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{
			ImportPaths: []string{"proto"},
		},
	}
	files, err := compiler.Compile(context.Background(), "user.proto", "post.proto")
	if err != nil {
		panic("Failed to compile proto files: " + err.Error())
	}
	fileDesc := files[0]
	runtimeSimpleDescriptor = fileDesc.Messages().ByName("User")
	runtimeComplexDescriptor = runtimeSimpleDescriptor // For this benchmark, use the same
}

// ===== SIMPLE PAYLOAD BENCHMARKS =====

func BenchmarkSimple_Protolite(b *testing.B) {
	b.ReportMetric(float64(len(simplePayload)), "payload_bytes")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result, err := protoliteClient.UnmarshalWithSchema(simplePayload, "benchmark.User")
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	}
}

func BenchmarkSimple_Protoc(b *testing.B) {
	b.ReportMetric(float64(len(simplePayload)), "payload_bytes")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		user := &pb.User{}
		err := proto.Unmarshal(simplePayload, user)
		if err != nil {
			b.Fatal(err)
		}
		_ = user
	}
}

func BenchmarkSimple_DynamicPB(b *testing.B) {
	b.ReportMetric(float64(len(simplePayload)), "payload_bytes")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		message := dynamicpb.NewMessage(simpleDescriptor)
		err := proto.Unmarshal(simplePayload, message)
		if err != nil {
			b.Fatal(err)
		}
		_ = message
	}
}

func BenchmarkSimple_DynamicPB_RuntimeDesc(b *testing.B) {
	b.ReportMetric(float64(len(simplePayload)), "payload_bytes")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		message := dynamicpb.NewMessage(runtimeSimpleDescriptor)
		err := proto.Unmarshal(simplePayload, message)
		if err != nil {
			b.Fatal(err)
		}
		_ = message
	}
}

// ===== COMPLEX PAYLOAD BENCHMARKS =====

func BenchmarkComplex_Protolite(b *testing.B) {
	b.ReportMetric(float64(len(complexPayload)), "payload_bytes")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result, err := protoliteClient.UnmarshalWithSchema(complexPayload, "benchmark.User")
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	}
}

func BenchmarkComplex_Protoc(b *testing.B) {
	b.ReportMetric(float64(len(complexPayload)), "payload_bytes")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		user := &pb.User{}
		err := proto.Unmarshal(complexPayload, user)
		if err != nil {
			b.Fatal(err)
		}
		_ = user
	}
}

func BenchmarkComplex_DynamicPB(b *testing.B) {
	b.ReportMetric(float64(len(complexPayload)), "payload_bytes")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		message := dynamicpb.NewMessage(complexDescriptor)
		err := proto.Unmarshal(complexPayload, message)
		if err != nil {
			b.Fatal(err)
		}
		_ = message
	}
}

func BenchmarkComplex_DynamicPB_RuntimeDesc(b *testing.B) {
	b.ReportMetric(float64(len(complexPayload)), "payload_bytes")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		message := dynamicpb.NewMessage(runtimeComplexDescriptor)
		err := proto.Unmarshal(complexPayload, message)
		if err != nil {
			b.Fatal(err)
		}
		_ = message
	}
}

// ===== VERIFICATION TESTS =====

func TestBenchmarkVerification(t *testing.T) {
	t.Logf("📦 Simple payload: %d bytes", len(simplePayload))
	t.Logf("📦 Complex payload: %d bytes", len(complexPayload))

	// Verify simple payload
	t.Log("\n🔍 SIMPLE PAYLOAD VERIFICATION:")

	result1, err := protoliteClient.UnmarshalWithSchema(simplePayload, "benchmark.User")
	if err != nil {
		t.Errorf("Protolite simple failed: %v", err)
	} else {
		t.Logf("✅ Protolite: %d fields", len(result1))
	}

	user2 := &pb.User{}
	err = proto.Unmarshal(simplePayload, user2)
	if err != nil {
		t.Errorf("Protoc simple failed: %v", err)
	} else {
		t.Logf("✅ Protoc: ID=%d, Name=%s", user2.Id, user2.Name)
	}

	message3 := dynamicpb.NewMessage(simpleDescriptor)
	err = proto.Unmarshal(simplePayload, message3)
	if err != nil {
		t.Errorf("DynamicPB simple failed: %v", err)
	} else {
		t.Logf("✅ DynamicPB: Message decoded")
	}

	// Verify complex payload
	t.Log("\n🔍 COMPLEX PAYLOAD VERIFICATION:")

	result4, err := protoliteClient.UnmarshalWithSchema(complexPayload, "benchmark.User")
	if err != nil {
		t.Errorf("Protolite complex failed: %v", err)
	} else {
		t.Logf("✅ Protolite: %d fields", len(result4))
	}

	user5 := &pb.User{}
	err = proto.Unmarshal(complexPayload, user5)
	if err != nil {
		t.Errorf("Protoc complex failed: %v", err)
	} else {
		t.Logf("✅ Protoc: ID=%d, Name=%s, Maps=%d, Posts=%d",
			user5.Id, user5.Name, len(user5.Metadata), len(user5.Posts))
	}

	message6 := dynamicpb.NewMessage(complexDescriptor)
	err = proto.Unmarshal(complexPayload, message6)
	if err != nil {
		t.Errorf("DynamicPB complex failed: %v", err)
	} else {
		t.Logf("✅ DynamicPB: Message decoded")
	}
}

// BenchmarkCompare_1K runs comprehensive allocation benchmarks with 1000 iterations.
// This provides meaningful results while keeping the test duration reasonable.
// The benchmark compares Protolite, Protoc-generated code, and DynamicPB
// for both simple and complex payloads.
func BenchmarkCompare_1K(b *testing.B) {
	const N = 1000
	b.Logf("Running each decode %d times\n", N)

	// --- SIMPLE PAYLOAD ---
	b.Log("\n--- SIMPLE PAYLOAD ---")

	b.StartTimer()
	start := testing.AllocsPerRun(N, func() {
		for i := 0; i < N; i++ {
			_, err := protoliteClient.UnmarshalWithSchema(simplePayload, "benchmark.User")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	b.StopTimer()
	b.Logf("Protolite.UnmarshalWithSchema: %d allocs/op", int(start))

	b.StartTimer()
	start = testing.AllocsPerRun(N, func() {
		for i := 0; i < N; i++ {
			user := &pb.User{}
			err := proto.Unmarshal(simplePayload, user)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	b.StopTimer()
	b.Logf("Protoc-generated Unmarshal: %d allocs/op", int(start))

	b.StartTimer()
	start = testing.AllocsPerRun(N, func() {
		for i := 0; i < N; i++ {
			msg := dynamicpb.NewMessage(runtimeSimpleDescriptor)
			err := proto.Unmarshal(simplePayload, msg)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	b.StopTimer()
	b.Logf("DynamicPB (runtime desc): %d allocs/op", int(start))

	// --- COMPLEX PAYLOAD ---
	b.Log("\n--- COMPLEX PAYLOAD ---")

	b.StartTimer()
	start = testing.AllocsPerRun(N, func() {
		for i := 0; i < N; i++ {
			_, err := protoliteClient.UnmarshalWithSchema(complexPayload, "benchmark.User")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	b.StopTimer()
	b.Logf("Protolite.UnmarshalWithSchema: %d allocs/op", int(start))

	b.StartTimer()
	start = testing.AllocsPerRun(N, func() {
		for i := 0; i < N; i++ {
			user := &pb.User{}
			err := proto.Unmarshal(complexPayload, user)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	b.StopTimer()
	b.Logf("Protoc-generated Unmarshal: %d allocs/op", int(start))

	b.StartTimer()
	start = testing.AllocsPerRun(N, func() {
		for i := 0; i < N; i++ {
			msg := dynamicpb.NewMessage(runtimeComplexDescriptor)
			err := proto.Unmarshal(complexPayload, msg)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	b.StopTimer()
	b.Logf("DynamicPB (runtime desc): %d allocs/op", int(start))
}
