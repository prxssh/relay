package utils

import "fmt"

func MapGetString(m map[string]any, key string, required bool) (string, error) {
	raw, ok := m[key]
	if !ok {
		if required {
			return "", fmt.Errorf("missing required key %q", key)
		}
		return "", nil
	}

	s, ok := raw.(string)
	if !ok {
		if required {
			return "", fmt.Errorf("value is not a string")
		}
		return "", nil
	}

	return s, nil
}

func MapGetInt(m map[string]any, key string, required bool) (int64, error) {
	raw, ok := m[key]
	if !ok {
		if required {
			return 0, fmt.Errorf("missing required key %q", key)
		}
		return 0, nil
	}

	s, ok := raw.(int64)
	if !ok {
		if required {
			return 0, fmt.Errorf("value is not a string")
		}
		return 0, nil
	}

	return s, nil
}
