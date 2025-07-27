package relay

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/prxssh/relay/internal/torrent"
)

// Client represents a struct which manages the complete state of of the
// torrents
type Client struct {
	ID       [sha1.Size]byte
	torrents map[[sha1.Size]byte]*torrentSession
}

const clientIDPrefix string = "-RELAY-"

func NewClient() (*Client, error) {
	clientID, err := generatePeerID()
	if err != nil {
		return nil, err
	}

	return &Client{ID: clientID}, nil
}

func (c *Client) ReadTorrentFromPath(path string) (*torrent.Torrent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	torrent, err := torrent.New(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	return torrent, nil
}

/////////////// Private /////////////////

func generatePeerID() ([sha1.Size]byte, error) {
	var clientID [sha1.Size]byte

	copy(clientID[:], []byte(clientIDPrefix))
	if _, err := rand.Read(clientID[len(clientIDPrefix):]); err != nil {
		return [sha1.Size]byte{}, fmt.Errorf("failed generated peer id: %w", err)
	}

	return clientID, nil
}
