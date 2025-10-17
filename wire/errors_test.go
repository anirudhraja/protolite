package wire

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestFieldError(t *testing.T) {
	tests := []struct {
		name          string
		buildError    func() error
		expectedPath  string
		expectedMsg   string
		containsWords []string
	}{
		{
			name: "single field error",
			buildError: func() error {
				baseErr := fmt.Errorf("message value must be map[string]interface{}, got float64")
				return wrapWithField(baseErr, "latitude")
			},
			expectedPath: "latitude",
			expectedMsg:  "message value must be map[string]interface{}, got float64",
		},
		{
			name: "nested field error",
			buildError: func() error {
				baseErr := fmt.Errorf("message value must be map[string]interface{}, got float64")
				err := wrapWithField(baseErr, "latitude")
				err = wrapWithField(err, "target_location")
				err = wrapWithField(err, "input")
				err = wrapWithField(err, "field_args")
				return err
			},
			expectedPath: "field_args.input.target_location.latitude",
			expectedMsg:  "message value must be map[string]interface{}, got float64",
			containsWords: []string{
				"field_args.input.target_location.latitude",
				"message value must be map[string]interface{}",
			},
		},
		{
			name: "deeply nested error - no repetition",
			buildError: func() error {
				baseErr := fmt.Errorf("expected string, got *int")
				err := wrapWithField(baseErr, "name")
				err = wrapWithField(err, "user")
				err = wrapWithField(err, "profile")
				err = wrapWithField(err, "data")
				return err
			},
			expectedPath: "data.profile.user.name",
			expectedMsg:  "expected string, got *int",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.buildError()

			// Check that it's a FieldError
			var fieldErr *FieldError
			if !errors.As(err, &fieldErr) {
				t.Fatalf("expected FieldError, got %T", err)
			}

			// Check the field path
			actualPath := strings.Join(fieldErr.FieldPath, ".")
			if actualPath != tt.expectedPath {
				t.Errorf("expected path %q, got %q", tt.expectedPath, actualPath)
			}

			// Check the error message format
			errMsg := err.Error()
			if !strings.Contains(errMsg, tt.expectedPath) {
				t.Errorf("error message should contain path %q, got: %s", tt.expectedPath, errMsg)
			}
			if !strings.Contains(errMsg, tt.expectedMsg) {
				t.Errorf("error message should contain %q, got: %s", tt.expectedMsg, errMsg)
			}

			// Check that repetitive phrases are NOT present
			repetitivePatterns := []string{
				"failed to encode field",
				"failed to encode nested message:",
			}
			for _, pattern := range repetitivePatterns {
				count := strings.Count(errMsg, pattern)
				if count > 1 {
					t.Errorf("error message contains repetitive pattern %q %d times: %s", pattern, count, errMsg)
				}
			}

			// Check for specific words if provided
			if len(tt.containsWords) > 0 {
				for _, word := range tt.containsWords {
					if !strings.Contains(errMsg, word) {
						t.Errorf("error message should contain %q, got: %s", word, errMsg)
					}
				}
			}

			// Unwrap should work
			unwrapped := errors.Unwrap(err)
			if unwrapped == nil {
				t.Error("Unwrap should return the underlying error")
			}
		})
	}
}

func TestFieldErrorUnwrap(t *testing.T) {
	baseErr := errors.New("base error")
	fieldErr := wrapWithField(baseErr, "field1")

	unwrapped := errors.Unwrap(fieldErr)
	if unwrapped == nil {
		t.Fatal("Unwrap should return non-nil")
	}
}

func TestNestedFieldError(t *testing.T) {
	tests := []struct {
		name          string
		buildError    func() error
		expectedPath  string
		expectedMsg   string
		containsWords []string
	}{
		{
			name: "single field error",
			buildError: func() error {
				baseErr := errors.New("bytes truncated: need 120 bytes, have 82")
				return wrapWithField(baseErr, "social_media")
			},
			expectedPath: "social_media",
			expectedMsg:  "bytes truncated",
		},
		{
			name: "nested field error",
			buildError: func() error {
				baseErr := errors.New("unexpected EOF while reading varint")
				err := wrapWithField(baseErr, "id")
				err = wrapWithField(err, "author")
				err = wrapWithField(err, "post")
				err = wrapWithField(err, "comments")
				return err
			},
			expectedPath: "comments.post.author.id",
			expectedMsg:  "unexpected EOF",
			containsWords: []string{
				"comments.post.author.id",
				"unexpected EOF",
			},
		},
		{
			name: "deeply nested error - no repetition",
			buildError: func() error {
				baseErr := errors.New("invalid varint encoding")
				err := wrapWithField(baseErr, "latitude")
				err = wrapWithField(err, "coordinates")
				err = wrapWithField(err, "location")
				err = wrapWithField(err, "address")
				err = wrapWithField(err, "user")
				return err
			},
			expectedPath: "user.address.location.coordinates.latitude",
			expectedMsg:  "invalid varint encoding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.buildError()

			// Check that it's a FieldError
			var fieldErr *FieldError
			if !errors.As(err, &fieldErr) {
				t.Fatalf("expected FieldError, got %T", err)
			}

			// Check the field path
			actualPath := strings.Join(fieldErr.FieldPath, ".")
			if actualPath != tt.expectedPath {
				t.Errorf("expected path %q, got %q", tt.expectedPath, actualPath)
			}

			// Check the error message format
			errMsg := err.Error()
			t.Logf("Error message: %s", errMsg)

			if !strings.Contains(errMsg, tt.expectedPath) {
				t.Errorf("error message should contain path %q, got: %s", tt.expectedPath, errMsg)
			}
			if !strings.Contains(errMsg, tt.expectedMsg) {
				t.Errorf("error message should contain %q, got: %s", tt.expectedMsg, errMsg)
			}

			// Check for specific words if provided
			if len(tt.containsWords) > 0 {
				for _, word := range tt.containsWords {
					if !strings.Contains(errMsg, word) {
						t.Errorf("error message should contain %q, got: %s", word, errMsg)
					}
				}
			}

			// Unwrap should work
			unwrapped := errors.Unwrap(err)
			if unwrapped == nil {
				t.Error("Unwrap should return the underlying error")
			}
		})
	}
}

func TestFieldErrorWrapping(t *testing.T) {
	baseErr := errors.New("base error message")

	// Create wrapped error
	wrappedErr := wrapWithField(baseErr, "field1")
	wrappedErr = wrapWithField(wrappedErr, "field2")

	// Check wrapped error
	var fieldErr *FieldError
	if errors.As(wrappedErr, &fieldErr) {
		expectedPath := "field2.field1"
		actualPath := strings.Join(fieldErr.FieldPath, ".")
		if actualPath != expectedPath {
			t.Errorf("expected path %q, got %q", expectedPath, actualPath)
		}

		if !strings.Contains(wrappedErr.Error(), "field2.field1") {
			t.Errorf("error message should contain 'field2.field1', got: %s", wrappedErr.Error())
		}
	} else {
		t.Fatal("expected FieldError")
	}

	t.Logf("Wrapped error: %s", wrappedErr.Error())
}
