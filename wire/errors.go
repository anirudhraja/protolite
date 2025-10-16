package wire

import (
	"fmt"
	"strings"
)

const (
	encodingError = "encoding"
	decodingError = "decoding"
)

// FieldError represents an encoding/decoding error with a field path.
type FieldError struct {
	FieldPath  []string // e.g., ["field_args", "input", "target_location", "latitude"]
	Err        error    // underlying error
	IsDecoding bool     // true if decoding, false if encoding
}

// Error implements the error interface.
func (e *FieldError) Error() string {
	if len(e.FieldPath) == 0 {
		return e.Err.Error()
	}

	errType := encodingError
	if e.IsDecoding {
		errType = decodingError
	}

	return fmt.Sprintf("%s error at '%s': %v", errType, strings.Join(e.FieldPath, "."), e.Err)
}

// Unwrap returns the underlying error.
func (e *FieldError) Unwrap() error {
	return e.Err
}

// Is implements errors.Is for compatibility.
func (e *FieldError) Is(target error) bool {
	_, ok := target.(*FieldError)
	return ok
}

// wrapFieldErrorWithMode wraps an error with a field name and mode (encoding/decoding).
func wrapFieldErrorWithMode(err error, fieldName string, isDecoding bool) error {
	if err == nil {
		return nil
	}

	if fe, ok := err.(*FieldError); ok {
		return &FieldError{
			FieldPath:  append([]string{fieldName}, fe.FieldPath...),
			Err:        fe.Err,
			IsDecoding: isDecoding || fe.IsDecoding,
		}
	}

	return &FieldError{
		FieldPath:  []string{fieldName},
		Err:        err,
		IsDecoding: isDecoding,
	}
}

// wrapEncodingFieldError wraps an encoding error.
func wrapEncodingFieldError(err error, fieldName string) error {
	return wrapFieldErrorWithMode(err, fieldName, false)
}

// wrapDecodingFieldError wraps a decoding error.
func wrapDecodingFieldError(err error, fieldName string) error {
	return wrapFieldErrorWithMode(err, fieldName, true)
}

// newFieldError creates a formatted base error.
func newFieldError(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}
