//Chat_Service\db\util.go

package db

import "fmt"

// pqIntArray — scanner для PostgreSQL массива INTEGER[].
type pqIntArray []int

func (a *pqIntArray) Scan(src interface{}) error {
	if src == nil {
		*a = nil
		return nil
	}

	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into pqIntArray", src)
	}

	// Формат: {1,2,3}
	if len(b) < 2 || b[0] != '{' || b[len(b)-1] != '}' {
		return fmt.Errorf("invalid array format")
	}

	content := string(b[1 : len(b)-1])
	if content == "" {
		*a = []int{}
		return nil
	}

	parts := splitArray(content)
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		var n int
		if _, err := fmt.Sscanf(p, "%d", &n); err != nil {
			return fmt.Errorf("cannot parse int %q: %w", p, err)
		}
		result = append(result, n)
	}

	*a = result
	return nil
}

func splitArray(s string) []string {
	var result []string
	var current string
	inQuotes := false

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			inQuotes = !inQuotes
		case ',':
			if !inQuotes {
				result = append(result, current)
				current = ""
				continue
			}
			current += string(c)
		default:
			current += string(c)
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}
