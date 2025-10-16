package wire

import (
	"errors"
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
				baseErr := newFieldError("message value must be map[string]interface{}, got float64")
				return wrapEncodingFieldError(baseErr, "latitude")
			},
			expectedPath: "latitude",
			expectedMsg:  "message value must be map[string]interface{}, got float64",
		},
		{
			name: "nested field error",
			buildError: func() error {
				baseErr := newFieldError("message value must be map[string]interface{}, got float64")
				err := wrapEncodingFieldError(baseErr, "latitude")
				err = wrapEncodingFieldError(err, "target_location")
				err = wrapEncodingFieldError(err, "input")
				err = wrapEncodingFieldError(err, "field_args")
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
				baseErr := newFieldError("expected string, got *int")
				err := wrapEncodingFieldError(baseErr, "name")
				err = wrapEncodingFieldError(err, "user")
				err = wrapEncodingFieldError(err, "profile")
				err = wrapEncodingFieldError(err, "data")
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

func TestNewFieldError(t *testing.T) {
	err := newFieldError("test error: %s", "details")
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !strings.Contains(err.Error(), "test error: details") {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestFieldErrorUnwrap(t *testing.T) {
	baseErr := newFieldError("base error")
	fieldErr := wrapEncodingFieldError(baseErr, "field1")

	unwrapped := errors.Unwrap(fieldErr)
	if unwrapped == nil {
		t.Fatal("Unwrap should return non-nil")
	}
}

func TestDecodingFieldError(t *testing.T) {
	tests := []struct {
		name          string
		buildError    func() error
		expectedPath  string
		expectedMsg   string
		containsWords []string
		shouldContain string
		shouldNotContain string
	}{
		{
			name: "single field decoding error",
			buildError: func() error {
				baseErr := errors.New("bytes truncated: need 120 bytes, have 82")
				return wrapDecodingFieldError(baseErr, "social_media")
			},
			expectedPath:  "social_media",
			expectedMsg:   "bytes truncated",
			shouldContain: "decoding error",
			shouldNotContain: "encoding error",
		},
		{
			name: "nested field decoding error",
			buildError: func() error {
				baseErr := errors.New("unexpected EOF while reading varint")
				err := wrapDecodingFieldError(baseErr, "id")
				err = wrapDecodingFieldError(err, "author")
				err = wrapDecodingFieldError(err, "post")
				err = wrapDecodingFieldError(err, "comments")
				return err
			},
			expectedPath: "comments.post.author.id",
			expectedMsg:  "unexpected EOF",
			containsWords: []string{
				"comments.post.author.id",
				"unexpected EOF",
				"decoding error",
			},
			shouldNotContain: "encoding error",
		},
		{
			name: "deeply nested decoding error - no repetition",
			buildError: func() error {
				baseErr := errors.New("invalid varint encoding")
				err := wrapDecodingFieldError(baseErr, "latitude")
				err = wrapDecodingFieldError(err, "coordinates")
				err = wrapDecodingFieldError(err, "location")
				err = wrapDecodingFieldError(err, "address")
				err = wrapDecodingFieldError(err, "user")
				return err
			},
			expectedPath: "user.address.location.coordinates.latitude",
			expectedMsg:  "invalid varint encoding",
			shouldContain: "decoding error",
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

			// Check IsDecoding flag
			if !fieldErr.IsDecoding {
				t.Error("expected IsDecoding to be true for decoding errors")
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

			// Check that it says "decoding error" not "encoding error"
			if tt.shouldContain != "" && !strings.Contains(errMsg, tt.shouldContain) {
				t.Errorf("error message should contain %q, got: %s", tt.shouldContain, errMsg)
			}
			if tt.shouldNotContain != "" && strings.Contains(errMsg, tt.shouldNotContain) {
				t.Errorf("error message should NOT contain %q, got: %s", tt.shouldNotContain, errMsg)
			}

			// Check that repetitive phrases are NOT present
			repetitivePatterns := []string{
				"failed to decode field",
				"failed to decode message",
				"decoding error at field path",
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

func TestEncodingVsDecodingErrors(t *testing.T) {
	baseErr := errors.New("base error message")

	// Create encoding error
	encodingErr := wrapEncodingFieldError(baseErr, "field1")
	encodingErr = wrapEncodingFieldError(encodingErr, "field2")

	// Create decoding error
	decodingErr := wrapDecodingFieldError(baseErr, "field1")
	decodingErr = wrapDecodingFieldError(decodingErr, "field2")

	// Check encoding error
	var encFieldErr *FieldError
	if errors.As(encodingErr, &encFieldErr) {
		if encFieldErr.IsDecoding {
			t.Error("encoding error should have IsDecoding=false")
		}
		if !strings.Contains(encodingErr.Error(), "encoding error") {
			t.Errorf("encoding error message should contain 'encoding error', got: %s", encodingErr.Error())
		}
		if strings.Contains(encodingErr.Error(), "decoding error") {
			t.Errorf("encoding error message should NOT contain 'decoding error', got: %s", encodingErr.Error())
		}
	} else {
		t.Fatal("expected FieldError for encoding")
	}

	// Check decoding error
	var decFieldErr *FieldError
	if errors.As(decodingErr, &decFieldErr) {
		if !decFieldErr.IsDecoding {
			t.Error("decoding error should have IsDecoding=true")
		}
		if !strings.Contains(decodingErr.Error(), "decoding error") {
			t.Errorf("decoding error message should contain 'decoding error', got: %s", decodingErr.Error())
		}
		if strings.Contains(decodingErr.Error(), "encoding error") {
			t.Errorf("decoding error message should NOT contain 'encoding error', got: %s", decodingErr.Error())
		}
	} else {
		t.Fatal("expected FieldError for decoding")
	}

	t.Logf("Encoding error: %s", encodingErr.Error())
	t.Logf("Decoding error: %s", decodingErr.Error())
}
