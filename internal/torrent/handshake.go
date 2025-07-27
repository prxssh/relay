package torrent

import (
	"crypto/sha1"
	"errors"
	"io"
)

type handshake struct {
	pstr     string
	infoHash [sha1.Size]byte
	peerID   [sha1.Size]byte
}

const szReservedBytes = 8

func newHandshake(infoHash, peerID [sha1.Size]byte) *handshake {
	return &handshake{
		pstr:     "BitTorrent protocol",
		infoHash: infoHash,
		peerID:   peerID,
	}
}

func (h *handshake) serialize() []byte {
	buf := make([]byte, len(h.pstr)+49)

	// <pstrlen><pstr><reserved><info_hash><peer_id>
	buf[0] = byte(len(h.pstr))
	offset := 1
	offset += copy(buf[offset:], []byte(h.pstr))
	offset += copy(buf[offset:], make([]byte, szReservedBytes))
	offset += copy(buf[offset:], h.infoHash[:])
	offset += copy(buf[offset:], h.peerID[:])

	return buf
}

func readHanshake(r io.Reader) (*handshake, error) {
	sizeBuf := make([]byte, 1)
	_, err := io.ReadFull(r, sizeBuf)
	if err != nil {
		return nil, err
	}

	pstrlen := sizeBuf[0]
	if pstrlen == 0 {
		return nil, errors.New("pstrlen can't be 0")
	}

	handshakeBuf := make([]byte, 48+pstrlen)
	if _, err := io.ReadFull(r, handshakeBuf); err != nil {
		return nil, err
	}

	var infoHash, peerID [sha1.Size]byte

	// <pstrlen><pstr><reserved><info_hash><peer_id>
	copy(
		infoHash[:],
		handshakeBuf[pstrlen+szReservedBytes:pstrlen+szReservedBytes+sha1.Size],
	)
	copy(peerID[:], handshakeBuf[pstrlen+szReservedBytes+sha1.Size:])

	return &handshake{
		pstr:     string(handshakeBuf[0:pstrlen]),
		infoHash: infoHash,
		peerID:   peerID,
	}, nil
}
