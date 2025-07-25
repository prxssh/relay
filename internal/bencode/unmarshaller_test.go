package bencode

import (
	"reflect"
	"strings"
	"testing"
)

func TestUnmarshal(t *testing.T) {
	testCases := []struct {
		name     string // The name of the test case
		input    string // The bencoded string input
		expected any    // The expected Go value
		err      bool   // The expected error, if any
	}{
		// --- Integer Tests ---
		{
			name:     "positive integer",
			input:    "i42e",
			expected: int64(42),
			err:      false,
		},
		{
			name:     "negative integer",
			input:    "i-42e",
			expected: int64(-42),
			err:      false,
		},
		{
			name:     "zero integer",
			input:    "i0e",
			expected: int64(0),
			err:      false,
		},
		{
			name:     "invalid integer format",
			input:    "i42ae",
			expected: nil,
			err:      true,
		},
		{
			name:     "integer not terminated",
			input:    "i42",
			expected: nil,
			err:      true,
		},

		// --- String Tests ---
		{
			name:     "simple string",
			input:    "5:hello",
			expected: "hello",
			err:      false,
		},
		{
			name:     "empty string",
			input:    "0:",
			expected: "",
			err:      false,
		},
		{
			name:     "string length too short",
			input:    "5:hell",
			expected: nil,
			err:      true,
		},
		{
			name:     "string with invalid negative length",
			input:    "-1:a",
			expected: nil,
			err:      true,
		},
		{
			name:     "string with no colon",
			input:    "5hello",
			expected: nil,
			err:      true,
		},

		// --- List Tests ---
		{
			name:     "list of strings",
			input:    "l4:spam4:eggse",
			expected: []any{"spam", "eggs"},
			err:      false,
		},
		{
			name:     "list of mixed types",
			input:    "l5:helloi42ee",
			expected: []any{"hello", int64(42)},
			err:      false,
		},
		{
			name:     "empty list",
			input:    "le",
			expected: []any{},
			err:      false,
		},
		{
			name:     "nested list",
			input:    "l4:spaml1:a1:bee",
			expected: []any{"spam", []any{"a", "b"}},
			err:      false,
		},
		{
			name:     "unterminated list",
			input:    "l4:spami1e",
			expected: nil,
			err:      true,
		},

		// --- Dictionary Tests ---
		{
			name:     "simple dictionary",
			input:    "d3:bar3:baz3:foo3:bare",
			expected: map[string]any{"foo": "bar", "bar": "baz"},
			err:      false,
		},
		{
			name:  "dictionary with mixed values",
			input: "d3:agei30e4:name8:John Doee",
			expected: map[string]any{
				"name": "John Doe",
				"age":  int64(30),
			},
			err: false,
		},
		{
			name:     "empty dictionary",
			input:    "de",
			expected: map[string]any{},
			err:      false,
		},
		{
			name:  "dictionary with nested list",
			input: "d4:listl1:a1:be3:numi10ee",
			expected: map[string]any{
				"list": []any{"a", "b"},
				"num":  int64(10),
			},
			err: false,
		},
		{
			name:     "unterminated dictionary",
			input:    "d3:key5:value",
			expected: nil,
			err:      true,
		},
		{
			name:     "dictionary with odd number of elements",
			input:    "d3:keye",
			expected: nil,
			err:      true,
		},

		// --- General Error Tests ---
		{
			name:     "empty input",
			input:    "",
			expected: nil,
			err:      true,
		},
		{
			name:     "invalid top-level type",
			input:    "x",
			expected: nil,
			err:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := strings.NewReader(tc.input)
			u := NewUnmarshaller(r)

			got, err := u.Unmarshal()

			if !tc.err && (err != nil) {
				t.Fatalf("expected no error, but got '%v", err)
			}

			if tc.err && err == nil {
				t.Fatalf("expected error, but got none")
			}

			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf(
					"unmarshalled value is incorrect:\ngot:      %#v (%T)\nexpected: %#v (%T)",
					got,
					got,
					tc.expected,
					tc.expected,
				)
			}
		})
	}
}
