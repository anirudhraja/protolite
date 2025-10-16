package wire

import (
	"fmt"
	"strings"
)

// FieldError represents an encoding error with a field path
type FieldError struct {
	FieldPath []string // Path to the field (e.g., ["field_args", "input", "target_location", "latitude"])
	Err       error    // The underlying error
}

// Error implements the error interface
func (e *FieldError) Error() string {
	if len(e.FieldPath) == 0 {
		return e.Err.Error()
	}
	return fmt.Sprintf("encoding error at field path '%s': %v", strings.Join(e.FieldPath, "."), e.Err)
}

// Unwrap returns the underlying error
func (e *FieldError) Unwrap() error {
	return e.Err
}

// wrapFieldError wraps an error with a field name, building the field path
func wrapFieldError(err error, fieldName string) error {
	if err == nil {
		return nil
	}

	// If it's already a FieldError, prepend the field name to the path
	if fe, ok := err.(*FieldError); ok {
		return &FieldError{
			FieldPath: append([]string{fieldName}, fe.FieldPath...),
			Err:       fe.Err,
		}
	}

	// Otherwise, create a new FieldError with this field as the start of the path
	return &FieldError{
		FieldPath: []string{fieldName},
		Err:       err,
	}
}

// newEncodingError creates a base encoding error (without field path)
func newEncodingError(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}
