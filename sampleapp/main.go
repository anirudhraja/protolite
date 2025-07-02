package main

import (
	"fmt"
	"log"

	"github.com/anirudhraja/protolite"
)

func main() {
	proto := protolite.NewProtolite()

	// Load proto files - load post.proto first since user.proto imports it
	err := proto.LoadSchemaFromFile("testdata/post.proto")
	if err != nil {
		log.Fatalf("Failed to load post.proto: %v", err)
	}

	err = proto.LoadSchemaFromFile("testdata/user.proto")
	if err != nil {
		log.Fatalf("Failed to load user.proto: %v", err)
	}

	// Create comprehensive User data demonstrating all protobuf features
	userData := map[string]interface{}{
		"id":        int32(1),
		"name":      "John Doe",
		"active":    true,
		"status":    int32(1), // USER_ACTIVE
		"user_type": int32(1), // USER_TYPE_PREMIUM

		// oneof contact_method - using email
		"email": "john.doe@example.com",

		// Nested Address message with deeply nested Coordinates
		"address": map[string]interface{}{
			"street":      "123 Main St",
			"city":        "San Francisco",
			"state":       "CA",
			"country":     "USA",
			"postal_code": "94105",
			"type":        int32(0), // ADDRESS_HOME
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
								"parent_comment_id": int32(1),
								"type":              int32(0), // REPLY_DIRECT
								"likes":             int32(8),
								"nested_replies": []map[string]interface{}{
									{
										"id":                int32(2),
										"user_id":           int32(2),
										"username":          "tech_reviewer",
										"content":           "Looking forward to more content!",
										"created_at":        int64(1641007200),
										"parent_comment_id": int32(1),
										"type":              int32(1), // REPLY_MENTION
										"likes":             int32(3),
										"nested_replies":    []map[string]interface{}{}, // Empty but shows recursive structure
									},
								},
							},
						},
					},
				},

				// Post metadata with nested structures
				"metadata": map[string]interface{}{
					"seo_title":       "Complete Protobuf Guide | Advanced Techniques",
					"seo_description": "Learn advanced protobuf techniques including oneof, maps, and nested messages",
					"keywords":        []string{"protobuf", "grpc", "serialization", "golang"},
					"custom_fields": map[string]string{
						"canonical_url": "https://blog.example.com/protobuf-guide",
						"amp_url":       "https://blog.example.com/amp/protobuf-guide",
					},
					"social_meta": map[string]interface{}{
						"og_title":       "Comprehensive Protobuf Guide",
						"og_description": "Advanced protobuf techniques and best practices",
						"og_image":       "https://blog.example.com/images/protobuf-cover.jpg",
						"twitter_card":   "summary_large_image",
						"platform_specific": map[string]string{
							"twitter:creator": "@john_doe",
							"fb:app_id":       "123456789",
						},
					},
					"collaborators": []map[string]interface{}{
						{
							"user_id":     int32(3),
							"username":    "editor_alice",
							"role":        int32(2), // ROLE_EDITOR
							"permissions": []string{"edit", "comment", "suggest"},
						},
						{
							"user_id":     int32(4),
							"username":    "reviewer_bob",
							"role":        int32(1), // ROLE_COMMENTER
							"permissions": []string{"comment", "view"},
						},
					},
				},
			},
		},

		"created_at": int64(1609459200),
	}

	fmt.Println("=== Comprehensive Protobuf Demo ===")
	fmt.Printf("Features demonstrated:\n")
	fmt.Printf("✅ oneof fields (contact_method, content, notification_data, comment_type)\n")
	fmt.Printf("✅ Nested messages (Address -> Coordinates, deep nesting)\n")
	fmt.Printf("✅ Nested repeated (notifications, comments with recursive replies)\n")
	fmt.Printf("✅ Multiple map types (string->string, string->int64, int32->string, string->Message)\n")
	fmt.Printf("✅ Comprehensive enums (12+ different enum types)\n")
	fmt.Printf("✅ Recursive structures (Reply -> nested_replies)\n\n")

	// Marshal with schema
	fmt.Println("Marshaling comprehensive user data...")
	encodedData, err := proto.MarshalWithSchema(userData, "User")
	if err != nil {
		log.Fatalf("Failed to marshal: %v", err)
	}

	fmt.Printf("✅ Encoded data size: %d bytes\n\n", len(encodedData))

	// Parse without schema (shows raw field numbers)
	fmt.Println("Schema-less parsing (field numbers only)...")
	parsedData, err := proto.Parse(encodedData)
	if err != nil {
		log.Fatalf("Failed to parse: %v", err)
	}

	fmt.Printf("✅ Parsed %d top-level fields\n", len(parsedData))
	for field := range parsedData {
		fmt.Printf("  - %s\n", field)
	}
	fmt.Println()

	// Unmarshal with schema (shows proper field names)
	fmt.Println("Schema-based unmarshaling (proper field names)...")
	userMap, err := proto.UnmarshalWithSchema(encodedData, "User")
	if err != nil {
		log.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify key features
	fmt.Printf("✅ User: %s (ID: %v)\n", userMap["name"], userMap["id"])

	if address, ok := userMap["address"].(map[string]interface{}); ok {
		fmt.Printf("✅ Nested Address: %s, %s\n", address["city"], address["state"])
		if coords, ok := address["coordinates"].(map[string]interface{}); ok {
			fmt.Printf("✅ Deeply nested Coordinates: %v, %v\n", coords["latitude"], coords["longitude"])
		}
	}

	if stats, ok := userMap["statistics"].(map[interface{}]interface{}); ok {
		fmt.Printf("✅ Statistics map: %d entries\n", len(stats))
	}

	if notifications, ok := userMap["notifications"].([]interface{}); ok {
		fmt.Printf("✅ Notifications: %d items\n", len(notifications))
	}

	if posts, ok := userMap["posts"].([]interface{}); ok {
		fmt.Printf("✅ Posts: %d items\n", len(posts))
		if len(posts) > 0 {
			if post, ok := posts[0].(map[string]interface{}); ok {
				if comments, ok := post["comments"].([]interface{}); ok {
					fmt.Printf("✅ Comments in first post: %d items\n", len(comments))
					if len(comments) > 0 {
						if comment, ok := comments[0].(map[string]interface{}); ok {
							if replies, ok := comment["replies"].([]interface{}); ok {
								fmt.Printf("✅ Replies in first comment: %d items\n", len(replies))
								if len(replies) > 0 {
									if reply, ok := replies[0].(map[string]interface{}); ok {
										if nested, ok := reply["nested_replies"].([]interface{}); ok {
											fmt.Printf("✅ Nested replies (recursive): %d items\n", len(nested))
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	fmt.Println("\n🎉 Comprehensive Protobuf demo completed successfully!")
	fmt.Println("All advanced features working: oneof, nested messages, recursive structures, multiple map types, and comprehensive enums!")
}
