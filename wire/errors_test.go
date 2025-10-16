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
				baseErr := newEncodingError("message value must be map[string]interface{}, got float64")
				return wrapFieldError(baseErr, "latitude")
			},
			expectedPath: "latitude",
			expectedMsg:  "message value must be map[string]interface{}, got float64",
		},
		{
			name: "nested field error",
			buildError: func() error {
				baseErr := newEncodingError("message value must be map[string]interface{}, got float64")
				err := wrapFieldError(baseErr, "latitude")
				err = wrapFieldError(err, "target_location")
				err = wrapFieldError(err, "input")
				err = wrapFieldError(err, "field_args")
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
				baseErr := newEncodingError("expected string, got *int")
				err := wrapFieldError(baseErr, "name")
				err = wrapFieldError(err, "user")
				err = wrapFieldError(err, "profile")
				err = wrapFieldError(err, "data")
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

func TestNewEncodingError(t *testing.T) {
	err := newEncodingError("test error: %s", "details")
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !strings.Contains(err.Error(), "test error: details") {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestFieldErrorUnwrap(t *testing.T) {
	baseErr := newEncodingError("base error")
	fieldErr := wrapFieldError(baseErr, "field1")

	unwrapped := errors.Unwrap(fieldErr)
	if unwrapped == nil {
		t.Fatal("Unwrap should return non-nil")
	}
}
