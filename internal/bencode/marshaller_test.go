package bencode

import (
	"bytes"
	"fmt"
	"testing"
)

func TestMarshal(t *testing.T) {
	testCases := []struct {
		name     string
		input    any
		expected string
		hasErr   bool
	}{
		// --- Integers ---
		{
			name:     "positive integer",
			input:    42,
			expected: "i42e",
			hasErr:   false,
		},
		{
			name:     "negative integer",
			input:    -42,
			expected: "i-42e",
			hasErr:   false,
		},
		{
			name:     "zero integer",
			input:    0,
			expected: "i0e",
			hasErr:   false,
		},
		{
			name:     "positive int64",
			input:    int64(1234567890),
			expected: "i1234567890e",
			hasErr:   false,
		},
		{
			name:     "simple string",
			input:    "hello",
			expected: "5:hello",
			hasErr:   false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "0:",
			hasErr:   false,
		},
		// --- Lists ---
		{
			name:     "list of strings",
			input:    []any{"spam", "eggs"},
			expected: "l4:spam4:eggse",
			hasErr:   false,
		},
		{
			name:     "list of integers",
			input:    []any{1, 2, 3},
			expected: "li1ei2ei3ee",
			hasErr:   false,
		},
		{
			name:     "empty list",
			input:    []any{},
			expected: "le",
			hasErr:   false,
		},
		{
			name:     "mixed list",
			input:    []any{"hello", 42, "world"},
			expected: "l5:helloi42e5:worlde",
			hasErr:   false,
		},
		// --- Dictionaries ---
		{
			name:     "simple dictionary",
			input:    map[string]any{"foo": "bar", "bar": "baz"},
			expected: "d3:bar3:baz3:foo3:bare",
			hasErr:   false,
		},
		{
			name:     "dictionary with mixed values",
			input:    map[string]any{"name": "John Doe", "age": 30},
			expected: "d3:agei30e4:name8:John Doee",
			hasErr:   false,
		},
		{
			name:     "empty dictionary",
			input:    map[string]any{},
			expected: "de",
			hasErr:   false,
		},
		// --- Nested Structures ---
		{
			name: "list with nested dictionary",
			input: []any{
				"a",
				"b",
				map[string]any{"d": 4, "c": 3},
			},
			expected: "l1:a1:bd1:ci3e1:di4eee",
			hasErr:   false,
		},
		{
			name: "dictionary with nested list",
			input: map[string]any{
				"names": []any{"Alice", "Bob"},
				"ages":  []any{25, 35},
			},
			expected: "d4:agesli25ei35ee5:namesl5:Alice3:Bobee",
			hasErr:   false,
		},
		// --- Error Cases ---
		{
			name:     "unsupported type float32",
			input:    float32(1.5),
			expected: "",
			hasErr:   true,
		},
		{
			name:     "unsupported type float64",
			input:    1.5,
			expected: "",
			hasErr:   true,
		},
		{
			name:     "unsupported type struct",
			input:    struct{ A int }{A: 1},
			expected: "",
			hasErr:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			m := NewMarshaller(&buf)

			err := m.Marshal(tc.input)

			if !tc.hasErr && err != nil {
				t.Fatalf("expected no error, but got: %v", err)
			}

			if tc.hasErr {
				if err == nil {
					t.Fatalf(
						"expected an error, but got nil",
					)
				}
				expectedErr := fmt.Sprintf(
					"bencode: unsupported type %T",
					tc.input,
				)
				if err.Error() != expectedErr {
					t.Errorf(
						"expected error message '%s', but got '%s'",
						expectedErr,
						err.Error(),
					)
				}
				return
			}

			got := buf.String()
			if got != tc.expected {
				t.Errorf(
					"unexpected bencode output:\ngot:    %s\nwant:   %s",
					got,
					tc.expected,
				)
			}
		})
	}
}
