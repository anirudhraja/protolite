package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/anirudhraja/protolite"
)

func main() {
	proto := protolite.NewProtolite([]string{"testdata",""})

	// Load proto files - load post.proto first since user.proto imports it
	err := proto.LoadSchemaFromFile("testdata/post.proto")
	if err != nil {
		log.Fatalf("Failed to load post.proto: %v", err)
	}

	err = proto.LoadSchemaFromFile("testdata/user.proto")
	if err != nil {
		log.Fatalf("Failed to load user.proto: %v", err)
	}

	fmt.Println("🚀 Protolite Sample App - Now with Google Protobuf Wrapper Types!")
	fmt.Println(strings.Repeat("=", 70))

	// Demonstrate wrapper types with multiple user examples
	demonstrateWrapperTypes(proto)

	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("📋 Complete User Demo with All Protobuf Features:")
	fmt.Println(strings.Repeat("=", 70))

	// Create comprehensive User data demonstrating all protobuf features
	userData := map[string]interface{}{
		"id":        int32(1),
		"name":      "John Doe",
		"active":    true,
		"status":    int32(1), // USER_ACTIVE
		"user_type": int32(1), // USER_TYPE_PREMIUM

		// oneof contact_method - using email
		"email": "john.doe@example.com",

		// 🎯 NEW: Wrapper types for optional/nullable fields
		"optional_nickname": "JohnnyDev",                         // StringValue - set
		"optional_age":      int32(29),                           // Int32Value - set
		"premium_member":    true,                                // BoolValue - set
		"account_balance":   float64(1250.75),                    // DoubleValue - set
		"bio":               "Go enthusiast and protobuf expert", // StringValue - set
		"last_login":        int64(1640995200),                   // Int64Value - set
		"reputation_score":  uint32(9850),                        // UInt32Value - set
		"rating":            float32(4.8),                        // FloatValue - set
		"profile_image":     []byte{0x89, 0x50, 0x4E, 0x47},      // BytesValue - set
		// Note: some wrapper fields intentionally omitted to show nil behavior

		// Nested Address message with deeply nested Coordinates
		"address": map[string]interface{}{
			"street":      "123 Main St",
			"city":        "San Francisco",
			"state":       "CA",
			"country":     "USA",
			"postal_code": "94105",
			"type":        int32(0), // ADDRESS_HOME

			// 🎯 NEW: Wrapper types in nested messages
			"apartment_number": "Apt 4B", // StringValue - set
			"is_primary":       true,     // BoolValue - set
			// special_instructions omitted - will be nil (not encoded)

			"coordinates": map[string]interface{}{
				"latitude":  37.7749,
				"longitude": -122.4194,
				"system":    int32(0), // COORD_WGS84
			},
		},

		// Different map types
		"metadata": map[string]string{ // map<string, string>
			"timezone":   "PST",
			"theme":      "dark",
			"language":   "en",
			"newsletter": "subscribed",
		},
		"statistics": map[string]int64{ // map<string, int64>
			"profile_views": int64(1250),
			"followers":     int64(890),
			"posts_liked":   int64(2340),
		},
		"preferences": map[int32]string{ // map<int32, string>
			int32(1): "email_notifications",
			int32(2): "dark_theme",
			int32(3): "auto_save",
		},
		"profiles": map[string]interface{}{ // map<string, UserProfile>
			"twitter": map[string]interface{}{
				"display_name": "John Doe 🚀",
				"bio":          "Software engineer passionate about Go and protobuf",
				"avatar_url":   "https://example.com/avatar.jpg",
				"visibility":   int32(0), // PROFILE_PUBLIC
				"interests":    []string{"golang", "protobuf", "distributed-systems"},
			},
			"linkedin": map[string]interface{}{
				"display_name": "John Doe",
				"bio":          "Senior Software Engineer",
				"avatar_url":   "https://linkedin.com/avatar.jpg",
				"visibility":   int32(2), // PROFILE_FRIENDS_ONLY
				"interests":    []string{"technology", "leadership", "innovation"},
			},
		},

		// Repeated nested notifications
		"notifications": []map[string]interface{}{
			{
				"id":        int32(1),
				"title":     "New follower",
				"message":   "Alice started following you",
				"type":      int32(4), // NOTIF_SOCIAL
				"timestamp": int64(1640995200),
				"read":      false,
				// oneof notification_data - using user_data
				"user_data": map[string]interface{}{
					"user_id":  int32(42),
					"username": "alice_codes",
					"action":   "followed",
				},
			},
			{
				"id":        int32(2),
				"title":     "System maintenance",
				"message":   "Scheduled maintenance tonight",
				"type":      int32(5), // NOTIF_SYSTEM
				"timestamp": int64(1641081600),
				"read":      true,
				// oneof notification_data - using system_data
				"system_data": map[string]interface{}{
					"system_message": "Database maintenance scheduled for 2AM-4AM PST",
					"action_url":     "https://status.example.com",
					"priority":       int32(1), // PRIORITY_MEDIUM
				},
			},
		},

		// Complex Posts with comprehensive features
		"posts": []map[string]interface{}{
			{
				"id":         int32(101),
				"title":      "Comprehensive Protobuf Guide",
				"author_id":  int32(1),
				"status":     int32(1), // POST_PUBLISHED
				"tags":       []string{"protobuf", "tutorial", "advanced"},
				"created_at": int64(1640995200),
				"updated_at": int64(1640995200),
				"view_count": int32(1500),
				"featured":   true,
				"rating":     int32(0),          // RATING_GENERAL
				"flags":      []int32{int32(0)}, // FLAG_NONE

				// oneof content - using text_content
				"text_content": map[string]interface{}{
					"body":       "This is a comprehensive guide to protobuf with oneof, maps, and nested messages...",
					"format":     int32(1), // TEXT_MARKDOWN
					"word_count": int32(2500),
					"footnotes":  []string{"Protocol Buffers documentation", "gRPC best practices"},
				},

				// Complex maps with different types
				"metrics": map[string]interface{}{ // map<string, PostMetric>
					"engagement": map[string]interface{}{
						"value":        85.5,
						"unit":         "percentage",
						"last_updated": int64(1641000000),
						"type":         int32(0), // METRIC_ENGAGEMENT
						"history": []map[string]interface{}{
							{"timestamp": int64(1640995200), "value": 82.1, "label": "day1"},
							{"timestamp": int64(1641081600), "value": 85.5, "label": "day2"},
						},
					},
				},
				"revisions": map[int32]string{ // map<int32, string>
					int32(1): "Initial draft",
					int32(2): "Added examples",
					int32(3): "Fixed typos",
				},
				"analytics": map[string]float64{ // map<string, double>
					"bounce_rate":  0.23,
					"time_on_page": 4.5,
					"conversion":   0.08,
				},
				"categories": map[string]interface{}{ // map<string, CategoryInfo>
					"technical": map[string]interface{}{
						"name":          "Technical Articles",
						"description":   "In-depth technical content",
						"type":          int32(0), // CATEGORY_PRIMARY
						"post_count":    int32(25),
						"subcategories": []string{"programming", "architecture", "tools"},
					},
				},

				// Nested repeated comments with recursive structure
				"comments": []map[string]interface{}{
					{
						"id":         int32(1),
						"user_id":    int32(2),
						"username":   "tech_reviewer",
						"content":    "Excellent guide! Very comprehensive.",
						"created_at": int64(1641000000),
						"updated_at": int64(1641000000),
						"status":     int32(0), // COMMENT_VISIBLE
						"likes":      int32(15),
						"pinned":     true,
						"metadata": map[string]string{
							"ip_address": "192.168.1.1",
							"user_agent": "Mozilla/5.0...",
						},
						// oneof comment_type - using text_comment
						"text_comment": map[string]interface{}{
							"formatted_text": "**Excellent** guide! Very comprehensive.",
							"format":         int32(1), // TEXT_MARKDOWN
							"mentions":       []string{"@john_doe"},
						},
						// Nested replies with recursive structure
						"replies": []map[string]interface{}{
							{
								"id":                int32(1),
								"user_id":           int32(1),
								"username":          "john_doe",
								"content":           "Thanks! Glad you found it helpful.",
								"created_at":        int64(1641003600),
								"updated_at":        int64(1641003600),
								"status":            int32(0), // COMMENT_VISIBLE
								"likes":             int32(5),
								"pinned":            false,
								"parent_comment_id": int32(1),
								"metadata": map[string]string{
									"ip_address": "10.0.0.1",
								},
								// oneof comment_type - using text_comment
								"text_comment": map[string]interface{}{
									"formatted_text": "Thanks! Glad you found it helpful.",
									"format":         int32(0), // TEXT_PLAIN
									"mentions":       []string{},
								},
								"replies": []map[string]interface{}{}, // Empty nested replies
							},
						},
					},
				},
			},
			{
				"id":         int32(102),
				"title":      "Advanced Protobuf Patterns",
				"author_id":  int32(1),
				"status":     int32(1), // POST_PUBLISHED
				"tags":       []string{"protobuf", "advanced", "patterns"},
				"created_at": int64(1641081600),
				"updated_at": int64(1641168000),
				"view_count": int32(275),
				"featured":   false,
				"rating":     int32(1),          // RATING_TEEN
				"flags":      []int32{int32(0)}, // FLAG_NONE

				// oneof content - using multimedia_content
				"multimedia_content": map[string]interface{}{
					"video_url":   "https://example.com/video.mp4",
					"duration":    int32(1800), // 30 minutes
					"quality":     int32(1),    // QUALITY_HD
					"thumbnails":  []string{"thumb1.jpg", "thumb2.jpg"},
					"captions":    []string{"English", "Spanish"},
					"resolution":  "1920x1080",
					"format":      "mp4",
					"file_size":   int64(157286400), // ~150MB
					"codec":       "H.264",
					"bitrate":     int32(2500),
					"frame_rate":  30.0,
					"audio_codec": "AAC",
				},

				"metrics": map[string]interface{}{ // map<string, PostMetric>
					"views": map[string]interface{}{
						"value":        275.0,
						"unit":         "count",
						"last_updated": int64(1641168000),
						"type":         int32(1), // METRIC_VIEWS
						"history": []map[string]interface{}{
							{"timestamp": int64(1641081600), "value": 100.0, "label": "launch"},
							{"timestamp": int64(1641168000), "value": 275.0, "label": "week1"},
						},
					},
				},
				"revisions": map[int32]string{ // map<int32, string>
					int32(1): "Initial version",
					int32(2): "Added video content",
				},
				"analytics": map[string]float64{ // map<string, double>
					"engagement_rate": 0.15,
					"completion_rate": 0.67,
					"likes_ratio":     0.89,
				},
				"categories": map[string]interface{}{ // map<string, CategoryInfo>
					"advanced": map[string]interface{}{
						"name":          "Advanced Topics",
						"description":   "Advanced technical deep-dives",
						"type":          int32(1), // CATEGORY_SECONDARY
						"post_count":    int32(8),
						"subcategories": []string{"patterns", "optimization", "best-practices"},
					},
				},
				"comments": []map[string]interface{}{}, // No comments yet
			},
		},

		"created_at": int64(1609459200), // 2021-01-01
	}

	// Marshal the user data
	encodedData, err := proto.MarshalWithSchema(userData, "User")
	if err != nil {
		log.Fatalf("Failed to marshal user data: %v", err)
	}

	fmt.Printf("\n📦 Encoded user data: %d bytes\n", len(encodedData))

	// Unmarshal back to verify
	result, err := proto.UnmarshalWithSchema(encodedData, "User")
	if err != nil {
		log.Fatalf("Failed to unmarshal user data: %v", err)
	}

	fmt.Println("\n✅ Successfully marshaled and unmarshaled comprehensive user data!")
	fmt.Printf("👤 User: %s (ID: %v)\n", result["name"], result["id"])
	fmt.Printf("📧 Email: %s\n", result["email"])
	fmt.Printf("🏠 Address: %s, %s, %s\n",
		result["address"].(map[string]interface{})["street"],
		result["address"].(map[string]interface{})["city"],
		result["address"].(map[string]interface{})["state"])

	// 🎯 Show wrapper type results
	showWrapperResults(result)

	fmt.Printf("📝 Posts: %d\n", len(result["posts"].([]interface{})))
	fmt.Printf("🔔 Notifications: %d\n", len(result["notifications"].([]interface{})))
	fmt.Printf("📊 Metadata entries: %d\n", len(result["metadata"].(map[interface{}]interface{})))

	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("🎉 All protobuf features working perfectly!")
	fmt.Println("✅ Primitive types, nested messages, maps, enums, oneof, repeated fields")
	fmt.Println("🎯 NEW: Google protobuf wrapper types with nullable semantics!")
	fmt.Println(strings.Repeat("=", 70))
}

// demonstrateWrapperTypes shows the key differences between regular proto3 fields and wrapper types
func demonstrateWrapperTypes(proto protolite.Protolite) {
	fmt.Println("\n🎯 Wrapper Types Demo - Nullable vs Non-Nullable Fields")
	fmt.Println(strings.Repeat("-", 60))

	// Example 1: User with some wrapper fields set
	fmt.Println("\n1️⃣ User with SOME optional fields set:")
	user1 := map[string]interface{}{
		"id":        int32(100),
		"name":      "Alice Smith",
		"active":    true,
		"status":    int32(1),
		"user_type": int32(0),

		// Wrapper fields - some set, some omitted (will be nil)
		"optional_nickname": "Alice_Codes",                            // StringValue - SET
		"optional_age":      int32(25),                                // Int32Value - SET
		"premium_member":    true,                                     // BoolValue - SET
		"bio":               "Frontend developer passionate about UX", // StringValue - SET
		// account_balance, last_login, reputation_score, rating, profile_image - OMITTED (nil)

		"email": "alice@example.com",
	}

	encoded1, err := proto.MarshalWithSchema(user1, "User")
	if err != nil {
		log.Fatalf("Failed to marshal user1: %v", err)
	}

	decoded1, err := proto.UnmarshalWithSchema(encoded1, "User")
	if err != nil {
		log.Fatalf("Failed to unmarshal user1: %v", err)
	}

	fmt.Printf("   📦 Encoded: %d bytes\n", len(encoded1))
	fmt.Printf("   👤 Name: %s, Nickname: %v\n", decoded1["name"], decoded1["optional_nickname"])
	fmt.Printf("   🎂 Age: %v, Premium: %v\n", decoded1["optional_age"], decoded1["premium_member"])
	fmt.Printf("   💰 Balance: %v (nil = not set)\n", decoded1["account_balance"])
	fmt.Printf("   ⭐ Rating: %v (nil = not set)\n", decoded1["rating"])

	// Example 2: User with NO wrapper fields set (all nil)
	fmt.Println("\n2️⃣ User with NO optional fields set:")
	user2 := map[string]interface{}{
		"id":        int32(101),
		"name":      "Bob Johnson",
		"active":    false,
		"status":    int32(2), // INACTIVE
		"user_type": int32(4), // GUEST
		"email":     "bob@example.com",
		// ALL wrapper fields omitted - they will be nil
	}

	encoded2, err := proto.MarshalWithSchema(user2, "User")
	if err != nil {
		log.Fatalf("Failed to marshal user2: %v", err)
	}

	decoded2, err := proto.UnmarshalWithSchema(encoded2, "User")
	if err != nil {
		log.Fatalf("Failed to unmarshal user2: %v", err)
	}

	fmt.Printf("   📦 Encoded: %d bytes (smaller - no optional fields!)\n", len(encoded2))
	fmt.Printf("   👤 Name: %s, Nickname: %v\n", decoded2["name"], decoded2["optional_nickname"])
	fmt.Printf("   🎂 Age: %v, Premium: %v\n", decoded2["optional_age"], decoded2["premium_member"])
	fmt.Printf("   💰 Balance: %v, Bio: %v\n", decoded2["account_balance"], decoded2["bio"])

	// Example 3: User with zero values in wrapper fields (different from nil!)
	fmt.Println("\n3️⃣ User with ZERO VALUES in optional fields (not nil!):")
	user3 := map[string]interface{}{
		"id":        int32(102),
		"name":      "Charlie Brown",
		"active":    true,
		"status":    int32(1),
		"user_type": int32(0),

		// Wrapper fields with explicit zero values (different from omitting!)
		"optional_nickname": "",           // StringValue - EMPTY STRING (not nil!)
		"optional_age":      int32(0),     // Int32Value - ZERO (not nil!)
		"premium_member":    false,        // BoolValue - FALSE (not nil!)
		"account_balance":   float64(0.0), // DoubleValue - ZERO (not nil!)
		"bio":               "",           // StringValue - EMPTY (not nil!)
		"reputation_score":  uint32(0),    // UInt32Value - ZERO (not nil!)
		"rating":            float32(0.0), // FloatValue - ZERO (not nil!)

		"email": "charlie@example.com",
	}

	encoded3, err := proto.MarshalWithSchema(user3, "User")
	if err != nil {
		log.Fatalf("Failed to marshal user3: %v", err)
	}

	decoded3, err := proto.UnmarshalWithSchema(encoded3, "User")
	if err != nil {
		log.Fatalf("Failed to unmarshal user3: %v", err)
	}

	fmt.Printf("   📦 Encoded: %d bytes (larger - all wrapper fields encoded!)\n", len(encoded3))
	fmt.Printf("   👤 Name: %s, Nickname: '%v' (empty string, not nil)\n", decoded3["name"], decoded3["optional_nickname"])
	fmt.Printf("   🎂 Age: %v (zero, not nil), Premium: %v (false, not nil)\n", decoded3["optional_age"], decoded3["premium_member"])
	fmt.Printf("   💰 Balance: %v (0.0, not nil)\n", decoded3["account_balance"])
	fmt.Printf("   ⭐ Rating: %v (0.0, not nil)\n", decoded3["rating"])

	fmt.Println("\n💡 Key Insight: Wrapper types distinguish between:")
	fmt.Println("   • nil (field not set/omitted) - saves space, not encoded")
	fmt.Println("   • zero value (field explicitly set to 0/false/empty) - encoded")
	fmt.Println("   • Regular proto3 fields always have default values, never nil")
}

// showWrapperResults displays the wrapper type fields from the decoded result
func showWrapperResults(result map[string]interface{}) {
	fmt.Println("\n🎯 Wrapper Type Results:")
	fmt.Println(strings.Repeat("-", 40))

	wrapperFields := []string{
		"optional_nickname", "optional_age", "premium_member",
		"account_balance", "bio", "last_login",
		"reputation_score", "rating", "profile_image",
	}

	for _, field := range wrapperFields {
		if value, exists := result[field]; exists {
			if value == nil {
				fmt.Printf("   %s: <nil> (not set)\n", field)
			} else {
				switch v := value.(type) {
				case []byte:
					fmt.Printf("   %s: %v (bytes, length: %d)\n", field, v, len(v))
				default:
					fmt.Printf("   %s: %v (%T)\n", field, value, value)
				}
			}
		} else {
			fmt.Printf("   %s: <not present>\n", field)
		}
	}

	// Show wrapper types in nested address
	if addr, ok := result["address"].(map[string]interface{}); ok {
		fmt.Println("\n🏠 Address Wrapper Fields:")
		addressWrappers := []string{"apartment_number", "is_primary", "special_instructions"}

		for _, field := range addressWrappers {
			if value, exists := addr[field]; exists {
				if value == nil {
					fmt.Printf("   %s: <nil> (not set)\n", field)
				} else {
					fmt.Printf("   %s: %v (%T)\n", field, value, value)
				}
			} else {
				fmt.Printf("   %s: <not present>\n", field)
			}
		}
	}
}
