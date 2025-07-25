package bencode

import (
	"bufio"
	"errors"
	"io"
	"strconv"
)

type Unmarshaller struct {
	r *bufio.Reader
}

type bencodedType byte

const (
	bInteger    bencodedType = 'i'
	bDict       bencodedType = 'd'
	bList       bencodedType = 'l'
	bTerminator bencodedType = 'e'
)

func NewUnmarshaller(r io.Reader) *Unmarshaller {
	return &Unmarshaller{r: bufio.NewReader(r)}
}

func (u *Unmarshaller) Unmarshal() (any, error) {
	btype, err := u.r.ReadByte()
	if err != nil {
		return nil, err
	}

	var val any
	var unmarshalErr error

	switch btype {
	case byte(bInteger):
		val, unmarshalErr = u.unmarshalInteger()
	case byte(bDict):
		val, unmarshalErr = u.unmarshalDict()
	case byte(bList):
		val, unmarshalErr = u.unmarshalList()
	default:
		if err := u.r.UnreadByte(); err != nil {
			return nil, err
		}
		val, unmarshalErr = u.unmarshalString()
	}

	if unmarshalErr != nil {
		return nil, unmarshalErr
	}
	return val, nil
}

/////////////// Private ///////////////

func (u *Unmarshaller) unmarshalInteger() (int64, error) {
	return u.readInteger(bTerminator)
}

func (u *Unmarshaller) unmarshalString() (string, error) {
	size, err := u.readInteger(':')
	if err != nil {
		return "", err
	}

	if size == 0 {
		return "", nil
	}

	if size < 0 {
		return "", errors.New(
			"bencode: invalid string, negative length",
		)
	}

	buf := make([]byte, size)
	if _, err := io.ReadFull(u.r, buf); err != nil {
		return "", err
	}

	return string(buf), nil
}

func (u *Unmarshaller) unmarshalList() ([]any, error) {
	list := make([]any, 0)

	for {
		peek, err := u.r.Peek(1)
		if err != nil {
			return nil, err
		}

		if peek[0] == byte(bTerminator) {
			u.r.ReadByte()
			break
		}

		v, err := u.Unmarshal()
		if err != nil {
			return nil, err
		}
		list = append(list, v)
	}

	return list, nil
}

func (u *Unmarshaller) unmarshalDict() (map[string]any, error) {
	dict := make(map[string]any)

	for {
		peek, err := u.r.Peek(1)
		if err != nil {
			return nil, err
		}

		if peek[0] == byte(bTerminator) {
			u.r.ReadByte()
			break
		}

		key, err := u.unmarshalString()
		if err != nil {
			return nil, err
		}

		val, err := u.Unmarshal()
		if err != nil {
			return nil, err
		}

		dict[string(key)] = val
	}

	return dict, nil
}

func (u *Unmarshaller) readInteger(delim bencodedType) (int64, error) {
	read, err := u.r.ReadBytes(byte(delim))
	if err != nil {
		return 0, err
	}

	sint := string(read[:len(read)-1])
	return strconv.ParseInt(sint, 10, 64)
}
