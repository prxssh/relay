package torrent

import (
	"encoding/binary"
	"io"
)

// messageid identifies the type of a message from a peer.
type messageid uint8

const (
	msgChoke         messageid = 0
	msgUnchoke       messageid = 1
	msgInterested    messageid = 2
	msgNotInterested messageid = 3
	msgHave          messageid = 4
	msgBitfield      messageid = 5
	msgRequest       messageid = 6
	msgPiece         messageid = 7
	msgCancel        messageid = 8
)

// message represents a message exchanged between BitTorrent peers
type message struct {
	id      messageid
	payload []byte
}

// marshal serializes a message into a byte slice in the format:
// <length prefix><message id><payload>.
// A nil message is serialized as a keep-alive message (4 bytes of zeros).
func (m *message) marshal() []byte {
	if m == nil { // keep-alive message
		return make([]byte, 4)
	}

	// <length prefix><message id><payload>
	length := uint32(len(m.payload) + 1) // +1 for id
	buf := make([]byte, 4+length)

	binary.BigEndian.PutUint32(buf[0:4], length)
	buf[4] = byte(m.id)
	copy(buf[5:], m.payload)

	return buf
}

// unmarshalMessage reads from an io.Reader and deserializes it into a message.
// It returns a nil message for keep-alives.
func unmarshalMessage(r io.Reader) (*message, error) {
	var length uint32

	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}

	// keep-alive message
	if length == 0 {
		return nil, nil
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}

	return &message{id: messageid(buf[0]), payload: buf[1:]}, nil
}

func messageChoke() *message {
	return &message{id: msgChoke}
}

func messageUnchoke() *message {
	return &message{id: msgUnchoke}
}

func messageInterested() *message {
	return &message{id: msgInterested}
}

func messageNotInterested() *message {
	return &message{id: msgNotInterested}
}

func messageHave(index int) *message {
	payload := make([]byte, 4)

	binary.BigEndian.PutUint32(payload, uint32(index))

	return &message{id: msgHave, payload: payload}
}

func messageRequest(index, begin, length int) *message {
	payload := make([]byte, 12)

	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	return &message{id: msgRequest, payload: payload}
}

func messagePiece(index, begin int, block []byte) *message {
	payload := make([]byte, 8+len(block))

	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	copy(payload[8:], block)

	return &message{id: msgPiece, payload: payload}
}

func messageCancel(index, begin, length int) *message {
	payload := make([]byte, 12)

	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	return &message{id: msgCancel, payload: payload}
}
