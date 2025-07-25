package bencode

import (
	"fmt"
	"io"
	"sort"
	"strconv"
)

type Marshaller struct {
	w io.Writer
}

func NewMarshaller(w io.Writer) *Marshaller {
	return &Marshaller{w: w}
}

func (m *Marshaller) Marshal(v any) error {
	switch vt := v.(type) {
	case int:
		return m.marshalInteger(int64(vt))
	case int64:
		return m.marshalInteger(vt)
	case string:
		return m.marshalString(vt)
	case []any:
		return m.marshalList(vt)
	case map[string]any:
		return m.marshalDict(vt)
	default:
		return fmt.Errorf("bencode: unsupported type %T", vt)
	}
}

/////////////// Private ///////////////

func (m *Marshaller) marshalInteger(val int64) error {
	_, err := m.w.Write([]byte("i" + strconv.FormatInt(val, 10)))
	return err
}

func (m *Marshaller) marshalString(s string) error {
	_, err := m.w.Write([]byte(strconv.Itoa(len(s)) + ":" + s))
	return err
}

func (m *Marshaller) marshalList(list []any) error {
	if _, err := m.w.Write([]byte("l")); err != nil {
		return err
	}

	for _, item := range list {
		if err := m.Marshal(item); err != nil {
			return err
		}
	}

	_, err := m.w.Write([]byte("e"))
	return err
}

func (m *Marshaller) marshalDict(dict map[string]any) error {
	if _, err := m.w.Write([]byte("d")); err != nil {
		return err
	}

	// Bencode spec requires keys sorted lexicographically
	keys := make([]string, 0, len(dict))
	for k := range dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if err := m.marshalString(k); err != nil {
			return err
		}
		if err := m.Marshal(dict[k]); err != nil {
			return err
		}
	}

	_, err := m.w.Write([]byte("e"))
	return err
}
