package wire

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// toLowerCamel converts snake_case to lowerCamelCase
func toLowerCamel(s string) string {
	if s == "" {
		return s
	}
	// Fast path: no underscore
	hasUnderscore := false
	for i := 0; i < len(s); i++ {
		if s[i] == '_' {
			hasUnderscore = true
			break
		}
	}
	if !hasUnderscore {
		// ensure lower first char
		if s[0] >= 'A' && s[0] <= 'Z' {
			return string(s[0]-'A'+'a') + s[1:]
		}
		return s
	}
	out := make([]byte, 0, len(s))
	upperNext := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '_' {
			upperNext = true
			continue
		}
		if len(out) == 0 {
			// first rune lowercased
			if c >= 'A' && c <= 'Z' {
				c = c - 'A' + 'a'
			}
			out = append(out, c)
			upperNext = false
			continue
		}
		if upperNext {
			if c >= 'a' && c <= 'z' {
				c = c - 'a' + 'A'
			}
			upperNext = false
		}
		out = append(out, c)
	}
	return string(out)
}

// Helpers to coerce JSON inputs to integers (accept exponent/float forms if integral)
func coerceToInt64(v interface{}) (int64, error) {
	switch t := v.(type) {
	case int64:
		return t, nil
	case int32:
		return int64(t), nil
	case json.Number:
		// Try integer first
		if iv, err := t.Int64(); err == nil {
			return iv, nil
		}
		// Fallback: parse as float and check integral
		f, err := strconv.ParseFloat(t.String(), 64)
		if err != nil {
			return 0, err
		}
		if f != math.Trunc(f) {
			return 0, fmt.Errorf("non-integer numeric for integer field")
		}
		return int64(f), nil
	case float64:
		if t != math.Trunc(t) {
			return 0, fmt.Errorf("non-integer numeric for integer field")
		}
		return int64(t), nil
	case string:
		// allow explicit integer strings
		if strings.ContainsAny(t, ".eE") {
			f, err := strconv.ParseFloat(t, 64)
			if err != nil {
				return 0, err
			}
			if f != math.Trunc(f) {
				return 0, fmt.Errorf("non-integer numeric for integer field")
			}
			return int64(f), nil
		}
		iv, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return 0, err
		}
		return iv, nil
	default:
		return 0, fmt.Errorf("expected integer-like, got %T", v)
	}
}

func coerceToUint64(v interface{}) (uint64, error) {
	switch t := v.(type) {
	case uint64:
		return t, nil
	case uint32:
		return uint64(t), nil
	case json.Number:
		if uv, err := strconv.ParseUint(t.String(), 10, 64); err == nil {
			return uv, nil
		}
		f, err := strconv.ParseFloat(t.String(), 64)
		if err != nil {
			return 0, err
		}
		if f < 0 || f != math.Trunc(f) {
			return 0, fmt.Errorf("non-integer numeric for unsigned field")
		}
		return uint64(f), nil
	case float64:
		if t < 0 || t != math.Trunc(t) {
			return 0, fmt.Errorf("non-integer numeric for unsigned field")
		}
		return uint64(t), nil
	case string:
		if strings.ContainsAny(t, ".eE") {
			f, err := strconv.ParseFloat(t, 64)
			if err != nil {
				return 0, err
			}
			if f < 0 || f != math.Trunc(f) {
				return 0, fmt.Errorf("non-integer numeric for unsigned field")
			}
			return uint64(f), nil
		}
		uv, err := strconv.ParseUint(t, 10, 64)
		if err != nil {
			return 0, err
		}
		return uv, nil
	default:
		return 0, fmt.Errorf("expected unsigned-integer-like, got %T", v)
	}
}
