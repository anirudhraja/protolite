package wire

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
