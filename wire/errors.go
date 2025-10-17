package wire

import (
	"fmt"
	"strings"
)

// FieldError represents an encoding/decoding error with a field path.
type FieldError struct {
	FieldPath  []string // e.g., ["field_args", "input", "target_location", "latitude"]
	Err        error    // underlying error
}

// Error implements the error interface.
func (e *FieldError) Error() string {
	if len(e.FieldPath) == 0 {
		return e.Err.Error()
	}

	return fmt.Sprintf("error at proto path %s: %v", strings.Join(e.FieldPath, "."), e.Err)
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

// wrapWithField wraps an error with a field name
func wrapWithField(err error, fieldName string) error {
	if err == nil {
		return nil
	}

	if fe, ok := err.(*FieldError); ok {
		return &FieldError{
			FieldPath:  append([]string{fieldName}, fe.FieldPath...),
			Err:        fe.Err,
		}
	}

	return &FieldError{
		FieldPath:  []string{fieldName},
		Err:        err,
	}
}
