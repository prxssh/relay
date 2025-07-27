package relay

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/prxssh/relay/internal/torrent"
)

// Client represents a struct which manages the complete state of the torrents.
type Client struct {
	// Unique 20-byte identifier for this client.
	ID [sha1.Size]byte
	// Mapping of a torrent's info hash to its active session.
	torrents map[[sha1.Size]byte]*session
}

const clientIDPrefix string = "-RL0001-"

func NewClient() (*Client, error) {
	clientID, err := generatePeerID()
	if err != nil {
		return nil, err
	}

	return &Client{
		ID:       clientID,
		torrents: make(map[[sha1.Size]byte]*session),
	}, nil
}

func (c *Client) AddTorrentFile(path string) (*session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	torrent, err := torrent.New(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	session, err := newSession(context.Background(), c.ID, torrent)
	if err != nil {
		return nil, err
	}

	c.torrents[torrent.Info.Hash] = session
	return session, nil
}

/////////////// Private /////////////////

func generatePeerID() ([sha1.Size]byte, error) {
	var clientID [sha1.Size]byte

	copy(clientID[:], []byte(clientIDPrefix))
	if _, err := rand.Read(clientID[len(clientIDPrefix):]); err != nil {
		return [sha1.Size]byte{}, fmt.Errorf(
			"failed generated peer id: %w",
			err,
		)
	}

	return clientID, nil
}
